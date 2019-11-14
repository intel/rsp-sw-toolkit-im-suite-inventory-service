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
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
	"reflect"

	log "github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/tag"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
	"time"
)

const (
	historyTable    = "dailyturnhistory"
	tagsTable       = "tags"
	jsonb           = "data"
	productIdColumn = "product_id"
	eventColumn     = "event"
	lastReadColumn  = "last_read"
	departedEvent   = "departed"
)

type PresentOrDepartedResults struct {
	ProductId    string `db:"_id"`
	DepartedTags int    `db:"departedTags"`
	PresentTags  int    `db:"presentTags"`
}

func Upsert(dbs *sql.DB, history History) error {
	// Metrics
	metrics.GetOrRegisterGaugeCollection(`Inventory.DailyTurn.Upsert.Attempt`, nil).Add(1)
	mSuccess := metrics.GetOrRegisterGaugeCollection(`Inventory.DailyTurn.Upsert.Success`, nil)
	mUpsertErr := metrics.GetOrRegisterGaugeCollection(`Inventory.DailyTurn.Upsert.Error`, nil)
	mUpsertLatency := metrics.GetOrRegisterTimer(`Inventory.DailyTurn.Upsert.Latency`, nil)

	obj, err := json.Marshal(history)
	if err != nil {
		return err
	}

	upsertStmt := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s) 
									 ON CONFLICT (( %s  ->> %s )) 
									 DO UPDATE SET %s = %s.%s || %s; `,
		pq.QuoteIdentifier(historyTable),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteLiteral(string(obj)),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteLiteral(productIdColumn),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteIdentifier(historyTable),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteLiteral(string(obj)),
	)

	upsertTimer := time.Now()
	_, err = dbs.Exec(upsertStmt)
	mUpsertLatency.Update(time.Since(upsertTimer))
	if err != nil {
		mUpsertErr.Add(1)
		return errors.Wrap(err, "db.dailyturnhistory.Upsert()")
	}

	mSuccess.Add(1)
	return nil
}

func computeDailyTurnRecord(dbs *sql.DB, productId string) error {
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
func ProcessIncomingASNList(dbs *sql.DB, asnList []tag.AdvanceShippingNotice) {
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
func CreateHistoryMap(dbs *sql.DB, tags []tag.Tag) map[string]History {
	historyMap := make(map[string]History)

	log.Debugf("Creating daily turn history map")

	for i := 0; i < len(tags); i++ {
		productId := tags[i].ProductID

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
func FindHistoryByProductId(dbs *sql.DB, productId string) (History, error) {

	// Metrics
	metrics.GetOrRegisterGaugeCollection(`Inventory.FindHistoryByProductId.Attempt`, nil).Add(1)
	mSuccess := metrics.GetOrRegisterGauge(`Inventory.FindHistoryByProductId.Success`, nil)
	mFindErr := metrics.GetOrRegisterGauge(`Inventory.FindHistoryByProductId.Find-Error`, nil)
	mFindLatency := metrics.GetOrRegisterTimer(`Inventory.FindHistoryByProductId.Find-Latency`, nil)
	mNotFound := metrics.GetOrRegisterGauge(`Inventory.FindHistoryByProductId.NotFound`, nil)

	var history History

	selectQuery := fmt.Sprintf(`SELECT %s FROM %s WHERE %s ->> %s = %s LIMIT 1`,
		pq.QuoteIdentifier(jsonb),
		pq.QuoteIdentifier(historyTable),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteLiteral(productIdColumn),
		pq.QuoteLiteral(productId),
	)

	retrieveTimer := time.Now()
	if err := dbs.QueryRow(selectQuery).Scan(&history); err != nil {

		if err == sql.ErrNoRows {
			mNotFound.Update(1)
			return History{}, nil
		}
		mFindErr.Update(1)
		return History{}, err
	}

	mFindLatency.Update(time.Since(retrieveTimer))

	mSuccess.Update(1)
	return history, nil
}

func FindPresentOrDepartedTagsSinceTimestamp(db *sql.DB, productId string, sinceTimestamp int64) (PresentOrDepartedResults, error) {
	// Metrics
	metrics.GetOrRegisterGaugeCollection(`Inventory.FindPresentOrDepartedTagsSinceTimestamp.Attempt`, nil).Add(1)
	mSuccess := metrics.GetOrRegisterGaugeCollection(`Inventory.FindPresentOrDepartedTagsSinceTimestamp.Success`, nil)
	mCountErr := metrics.GetOrRegisterGaugeCollection(`Inventory.FindPresentOrDepartedTagsSinceTimestamp.Find-Error`, nil)
	mFindLatency := metrics.GetOrRegisterTimer(`Inventory.FindPresentOrDepartedTagsSinceTimestamp.Find-Latency`, nil)

	result := PresentOrDepartedResults{}
	var count int
	retrieveTimer := time.Now()
	timestamp := fmt.Sprintf("%v", sinceTimestamp)

	// query for not departed i.e. present tags
	selectStmt := fmt.Sprintf("SELECT count(*) FROM %s WHERE %s ->> %s = %s" +
		"AND %s ->> %s != %s AND (%s ->> %s)::numeric > '0'",
		pq.QuoteIdentifier(tagsTable),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteLiteral(productIdColumn),
		pq.QuoteLiteral(productId),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteLiteral(eventColumn),
		pq.QuoteLiteral(departedEvent),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteLiteral(lastReadColumn),
	)

	row := db.QueryRow(selectStmt)
	err := row.Scan(&count)
	mFindLatency.Update(time.Since(retrieveTimer))
	if err != nil {
		mCountErr.Add(1)
		return PresentOrDepartedResults{}, err
	} else {
		result.PresentTags = count
		mSuccess.Add(1)
	}

	// query for departed tags
	selectStmt = fmt.Sprintf("SELECT count(*) FROM %s  WHERE %s ->> %s = %s"+
		"AND %s ->> %s = %s AND (%s ->> %s)::numeric > %s",
		pq.QuoteIdentifier(tagsTable),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteLiteral(productIdColumn),
		pq.QuoteLiteral(productId),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteLiteral(eventColumn),
		pq.QuoteLiteral(departedEvent),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteLiteral(lastReadColumn),
		pq.QuoteLiteral(timestamp),
	)

	row = db.QueryRow(selectStmt)
	err = row.Scan(&count)
	mFindLatency.Update(time.Since(retrieveTimer))
	if err != nil {
		mCountErr.Add(1)
		return PresentOrDepartedResults{}, err
	} else {
		result.DepartedTags = count
		mSuccess.Add(1)
		return result, nil
	}
}

// Value implements driver.Valuer interfaces
func (history History) Value() (driver.Value, error) {
	return json.Marshal(history)
}

// Scan implements sql.Scanner interfaces
func (history *History) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, history)
}
