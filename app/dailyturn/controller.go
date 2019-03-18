/*
 * INTEL CONFIDENTIAL
 * Copyright (2018) Intel Corporation.
 *
 * The source code contained or described herein and all documents related to the source code ("Material")
 * are owned by Intel Corporation or its suppliers or licensors. Title to the Material remains with
 * Intel Corporation or its suppliers and licensors. The Material may contain trade secrets and proprietary
 * and confidential information of Intel Corporation and its suppliers and licensors, and is protected by
 * worldwide copyright and trade secret laws and treaty provisions. No part of the Material may be used,
 * copied, reproduced, modified, published, uploaded, posted, transmitted, distributed, or disclosed in
 * any way without Intel/'s prior express written permission.
 * No license under any patent, copyright, trade secret or other intellectual property right is granted
 * to or conferred upon you by disclosure or delivery of the Materials, either expressly, by implication,
 * inducement, estoppel or otherwise. Any license under such intellectual property rights must be express
 * and approved by Intel in writing.
 * Unless otherwise agreed by Intel in writing, you may not remove or alter this notice or any other
 * notice embedded in Materials by Intel or Intel's suppliers or licensors in any way.
 */

package dailyturn

import (
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	db "github.impcloud.net/RSP-Inventory-Suite/go-dbWrapper"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/tag"
	"reflect"
	"time"
)

const (
	historyCollection = "dailyturnhistory"
	tagCollection     = "tags"
)

type PresentOrDepartedResults struct {
	ProductId    string `bson:"_id"`
	DepartedTags int    `bson:"departedTags"`
	PresentTags  int    `bson:"presentTags"`
}

func Upsert(dbs *db.DB, history History) error {
	// Metrics
	metrics.GetOrRegisterGaugeCollection(`Inventory.DailyTurn.Upsert.Attempt`, nil).Add(1)
	mSuccess := metrics.GetOrRegisterGaugeCollection(`Inventory.DailyTurn.Upsert.Success`, nil)
	mUpsertErr := metrics.GetOrRegisterGaugeCollection(`Inventory.DailyTurn.Upsert.Error`, nil)
	mUpsertLatency := metrics.GetOrRegisterTimer(`Inventory.DailyTurn.Upsert.Latency`, nil)

	execFunc := func(collection *mgo.Collection) error {
		_, err := collection.Upsert(bson.M{"product_id": history.ProductID}, history)
		return err
	}

	upsertTimer := time.Now()
	err := dbs.Execute(historyCollection, execFunc)
	mUpsertLatency.Update(time.Since(upsertTimer))

	if err != nil {
		mUpsertErr.Add(1)
		return errors.Wrap(err, "db.dailyturnhistory.Upsert()")
	}

	mSuccess.Add(1)
	return nil
}

func computeDailyTurnRecord(dbs *db.DB, productId string) error {
	history, err := FindHistoryByProductId(dbs, productId)
	if err != nil {
		return err
	}

	now := helper.UnixMilliNow()

	log.Debugf("computeDailyTurnRecord: %s", productId)

	if reflect.DeepEqual(history, History{}) {
		// create a new history with 0 records, but a timestamp
		history = History{
			ProductID: productId,
			DailyTurn: 0,
			Timestamp: now,
			Records:   []Record{},
		}
		log.Debugf("Create new history: %s", productId)
	} else {
		result, err := FindPresentOrDepartedTagsSinceTimestamp(dbs, productId, history.Timestamp)
		if err != nil {
			return err
		}

		record := Record{
			Timestamp:         now,
			PreviousTimestamp: history.Timestamp,
			Departed:          result.DepartedTags,
			Present:           result.PresentTags,
		}

		if err := record.ComputeDailyTurn(); err != nil {
			return err
		}

		// insert new record at beginning
		history.Records = append([]Record{record}, history.Records...)

		// truncate to maximum record size if needed
		if len(history.Records) > config.AppConfig.DailyTurnHistoryMaximum {
			history.Records = history.Records[:config.AppConfig.DailyTurnHistoryMaximum]
		}

		history.ComputeAverageDailyTurn()
		history.Timestamp = now
	}

	return Upsert(dbs, history)
}

// ProcessIncomingASNList takes a list of advance shipping notices and ingests it into the
// database for calculating the dailyturn. NOTE: this should be called AFTER ingesting the
// new tags into the database. The reason for this is we don't want to double count EPCs already
// in the database by simply adding quantity to the inventory count, so we let the inventory count
// fill up via the processing already in place.
func ProcessIncomingASNList(dbs *db.DB, asnList []tag.AdvanceShippingNotice) {
	// Metrics
	metrics.GetOrRegisterGaugeCollection(`Inventory.DailyTurn.ProcessIncomingASNList.Attempt`, nil).Add(1)
	mProcessLatency := metrics.GetOrRegisterTimer(`Inventory.DailyTurn.ProcessIncomingASNList.Process-Latency`, nil)

	log.Debug("Process incoming ASN")
	beginTimer := time.Now()

	for _, asn := range asnList {
		for _, asnItem := range asn.Items {
			if err := computeDailyTurnRecord(dbs, asnItem.ItemGTIN); err != nil {
				// this is not an error because the data may not be ready yet to compute the daily turn
				log.Infof("Unable to compute the daily turn for product_id %s: %v", asnItem.ItemGTIN, err.Error())
				continue
			}
		}
	}

	mProcessLatency.Update(time.Since(beginTimer))
}

// CreateHistoryMap builds a map[string] based of array of product ids for search efficiency
func CreateHistoryMap(dbs *db.DB, tags *[]tag.Tag) map[string]History {
	historyMap := make(map[string]History)

	log.Debugf("Creating daily turn history map")

	for i := 0; i < len(*tags); i++ {
		productId := (*tags)[i].ProductID

		if _, alreadyExists := historyMap[productId]; alreadyExists == true {
			// skip lookup for products we already have
			continue
		}

		history, err := FindHistoryByProductId(dbs, productId)
		if err != nil {
			log.Errorf("Error adding daily turn history to map for product %s: %v", productId, err.Error())
			continue
		}

		historyMap[productId] = history
	}

	return historyMap
}

// FindHistoryByProductId searches DB for tag based on the productId value
// Returns the History if found or empty History if it does not exist
func FindHistoryByProductId(dbs *db.DB, productId string) (History, error) {

	// Metrics
	metrics.GetOrRegisterGaugeCollection(`Inventory.FindHistoryByProductId.Attempt`, nil).Add(1)
	mSuccess := metrics.GetOrRegisterGaugeCollection(`Inventory.FindHistoryByProductId.Success`, nil)
	mNotFound := metrics.GetOrRegisterGaugeCollection(`Inventory.FindHistoryByProductId.NotFound`, nil)
	mFindErr := metrics.GetOrRegisterGaugeCollection(`Inventory.FindHistoryByProductId.Find-Error`, nil)
	mFindLatency := metrics.GetOrRegisterTimer(`Inventory.FindHistoryByProductId.Find-Latency`, nil)

	var history History

	execFunc := func(collection *mgo.Collection) error {
		return collection.Find(bson.M{"product_id": productId}).One(&history)
	}
	retrieveTimer := time.Now()
	if err := dbs.Execute(historyCollection, execFunc); err != nil {
		// If the error was because item does not exist, return empty History and no error
		if err == mgo.ErrNotFound {
			mNotFound.Add(1)
			return History{}, nil
		}
		mFindErr.Add(1)
		return History{}, errors.Wrap(err, "db.dailyturnhistory.find()")
	}
	mFindLatency.Update(time.Since(retrieveTimer))

	mSuccess.Add(1)
	return history, nil
}

func FindPresentOrDepartedTagsSinceTimestamp(dbs *db.DB, productId string, sinceTimestamp int64) (PresentOrDepartedResults, error) {
	// Metrics
	metrics.GetOrRegisterGaugeCollection(`Inventory.FindPresentOrDepartedTagsSinceTimestamp.Attempt`, nil).Add(1)
	mSuccess := metrics.GetOrRegisterGaugeCollection(`Inventory.FindPresentOrDepartedTagsSinceTimestamp.Success`, nil)
	mNotFound := metrics.GetOrRegisterGaugeCollection(`Inventory.FindPresentOrDepartedTagsSinceTimestamp.NotFound`, nil)
	mFindErr := metrics.GetOrRegisterGaugeCollection(`Inventory.FindPresentOrDepartedTagsSinceTimestamp.Find-Error`, nil)
	mFindLatency := metrics.GetOrRegisterTimer(`Inventory.FindPresentOrDepartedTagsSinceTimestamp.Find-Latency`, nil)

	result := PresentOrDepartedResults{}

	execFunc := func(collection *mgo.Collection) error {
		// product_id == productId && last_read > 0 && ( event != departed || last_read > sinceTimestamp )
		return collection.Pipe([]bson.M{{
			"$match": bson.M{
				"$and": []bson.M{
					{"product_id": bson.M{"$eq": productId}},
					{"last_read": bson.M{"$gt": 0}},
					{
						"$or": []bson.M{
							{"event": bson.M{"$ne": "departed"}},
							{"last_read": bson.M{"$gt": sinceTimestamp}},
						},
					},
				},
			}}, {
			"$group": bson.M{
				"_id": "$product_id",
				"presentTags": bson.M{"$sum": bson.M{"$cond": []interface{}{
					bson.M{"$ne": []string{"$event", "departed"}}, 1, 0,
				}}},
				"departedTags": bson.M{"$sum": bson.M{"$cond": []interface{}{
					bson.M{"$eq": []string{"$event", "departed"}}, 1, 0,
				}}},
			}},
		}).One(&result)
	}

	retrieveTimer := time.Now()
	if err := dbs.Execute(tagCollection, execFunc); err != nil {
		if err == mgo.ErrNotFound {
			mNotFound.Add(1)
			return PresentOrDepartedResults{}, nil
		}

		mFindErr.Add(1)
		return PresentOrDepartedResults{}, errors.Wrap(err, "db.tags.aggregate()")
	}
	mFindLatency.Update(time.Since(retrieveTimer))

	mSuccess.Add(1)
	return result, nil
}
