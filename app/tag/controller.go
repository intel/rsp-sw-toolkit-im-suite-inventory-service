/*
 * INTEL CONFIDENTIAL
 * Copyright (2017) Intel Corporation.
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

package tag

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	db "github.impcloud.net/RSP-Inventory-Suite/go-dbWrapper"
	odata "github.impcloud.net/RSP-Inventory-Suite/go-odata/mongo"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/web"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
)

const (
	tagCollection = "tags"
	// UndefinedProductID is the constant to set the product id when it cannot be decoded
	UndefinedProductID = "undefined"
	// UndefinedURI is the constant to set the uri when it cannot be decoded
	UndefinedURI = "undefined"
	// Write batch sizes must be between 1 and 1000. For safety, split it into 500 operations per call
	// Because upsert requires a pair of upserting instructions. Use mongoMaxOps = 1000
	mongoMaxOps = 1000
	// encodingInvalid is the constant to set when epc encoding cannot be decoded
	encodingInvalid = "encoding:invalid"
	// encodingUndefined is the constant to set when epc encoding cannot be decoded
	encodingUndefined = "encoding:undefined"

	numEpcBits   = 96
	numEpcDigits = numEpcBits / 4 // 4 bits per hex digit
)

// Retrieve retrieves tags from database based on Odata query and a size limit
//nolint:dupl
func Retrieve(dbs *db.DB, query url.Values, maxSize int) (interface{}, *CountType, *PagingType, error) {

	// Metrics
	metrics.GetOrRegisterGauge(`Inventory.Retrieve.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Inventory.Retrieve.Success`, nil)
	mFindErr := metrics.GetOrRegisterGauge("Inventory.Retrieve.Find-Error", nil)
	mInputErr := metrics.GetOrRegisterGauge("Inventory.Retrieve.Input-Error", nil)
	mFindLatency := metrics.GetOrRegisterTimer(`Inventory.Retrieve.Find-Latency`, nil)

	//var tagArray []Tag
	var object []interface{}
	var paging PagingType

	// If count is true, and only $count is set return total count of the collection
	if len(query["$count"]) > 0 && len(query) < 2 {

		return countHandler(dbs)
	}

	if len(query["$top"]) > 0 {

		topVal, err := strconv.Atoi(query["$top"][0])
		if err != nil {
			return nil, nil, nil, errors.Wrap(web.ErrValidation, "invalid $top value")
		}

		if topVal > maxSize {
			query["$top"][0] = strconv.Itoa(maxSize)
		}

	} else {
		query["$top"] = []string{strconv.Itoa(maxSize)} // Apply size limit to the odata query
	}

	// Else, run filter query and return slice of Tag
	execFunc := func(collection *mgo.Collection) error {
		return odata.ODataQuery(query, &object, collection)
	}

	retrieveTimer := time.Now()
	if err := dbs.Execute(tagCollection, execFunc); err != nil {
		if errors.Cause(err) == odata.ErrInvalidInput {
			mInputErr.Update(1)
			return nil, nil, nil, errors.Wrap(web.ErrInvalidInput, err.Error())
		}
		mFindErr.Update(1)
		return nil, nil, nil, errors.Wrap(err, "db.tags.find()")
	}
	mFindLatency.Update(time.Since(retrieveTimer))

	if len(object) > 0 {
		lastSlice := object[len(object)-1]
		sliceMap := lastSlice.(bson.M)
		objectID := sliceMap["_id"].(bson.ObjectId)
		paging.Cursor = objectID.Hex()
	}
	// Check if inlinecount is set
	isInlineCount := query["$inlinecount"]
	var countType *CountType

	if len(query["$count"]) > 0 || (len(isInlineCount) > 0 && isInlineCount[0] == "allpages") {
		return inlineCountHandler(dbs, isInlineCount, object)
	}

	var pagingType *PagingType
	if len(query["$top"]) > 0 {
		pagingType = &paging
	}

	mSuccess.Update(1)
	return object, countType, pagingType, nil
}

// RetrieveOdataAll retrieves all tags from database based on Odata query
func RetrieveOdataAll(dbs *db.DB, query url.Values) ([]Tag, error) {

	// Metrics
	metrics.GetOrRegisterGauge(`Inventory.RetrieveOdataNoLimit.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Inventory.RetrieveOdataNoLimit.Success`, nil)
	mFindErr := metrics.GetOrRegisterGauge("Inventory.RetrieveOdataNoLimit.Find-Error", nil)
	mFindLatency := metrics.GetOrRegisterTimer(`Inventory.RetrieveOdataNoLimit.Find-Latency`, nil)

	var object []Tag

	execFunc := func(collection *mgo.Collection) error {
		return odata.ODataQuery(query, &object, collection)
	}

	retrieveTimer := time.Now()
	if err := dbs.Execute(tagCollection, execFunc); err != nil {
		mFindErr.Update(1)
		return nil, errors.Wrap(err, "Error retrieving all tags based on odata query")
	}
	mFindLatency.Update(time.Since(retrieveTimer))

	if len(object) > 0 {
		mSuccess.Update(1)
		return object, nil
	}

	mSuccess.Update(1)
	return nil, nil
}

func countHandler(dbs *db.DB) (interface{}, *CountType, *PagingType, error) {

	mSuccess := metrics.GetOrRegisterGauge(`Inventory.Retrieve.Success`, nil)
	mCountErr := metrics.GetOrRegisterGauge("Location.Retrieve.Count-Error", nil)

	var count int
	var err error

	execFunc := func(collection *mgo.Collection) (int, error) {
		return odata.ODataCount(collection)
	}

	if count, err = dbs.ExecuteCount(tagCollection, execFunc); err != nil {
		mCountErr.Update(1)
		return nil, nil, nil, errors.Wrap(err, "db.inventory.Count()")
	}
	mSuccess.Update(1)
	return nil, &CountType{Count: &count}, nil, nil
}

func inlineCountHandler(dbs *db.DB, isInlineCount []string, object []interface{}) (interface{}, *CountType, *PagingType, error) {

	mSuccess := metrics.GetOrRegisterGauge(`Inventory.Retrieve.Success`, nil)
	mCountErr := metrics.GetOrRegisterGauge("Location.Retrieve.Count-Error", nil)

	var inlineCount int
	var err error

	// Get count from filtered data
	execInlineCount := func(collection *mgo.Collection) (int, error) {
		return odata.ODataInlineCount(collection)
	}

	if inlineCount, err = dbs.ExecuteCount(tagCollection, execInlineCount); err != nil {
		mCountErr.Update(1)
		return nil, nil, nil, errors.Wrap(err, "db.inventory.Count()")
	}

	// if $inlinecount is set, return results and inlinecount
	if len(isInlineCount) > 0 {

		if isInlineCount[0] == "allpages" {
			mSuccess.Update(1)
			return object, &CountType{Count: &inlineCount}, nil, nil
		}
	}

	// if $count is set with $filter, return only the count of the filtered results
	mSuccess.Update(1)
	return nil, &CountType{Count: &inlineCount}, nil, nil

}

// FindByEpc searches DB for tag based on the epc value
// Returns the tag if found or empty tag if it does not exist
func FindByEpc(dbs *db.DB, epc string) (Tag, error) {

	// Metrics
	metrics.GetOrRegisterGauge(`Inventory.FindByEpc.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Inventory.FindByEpc.Success`, nil)
	mFindErr := metrics.GetOrRegisterGauge("Inventory.FindByEpc.Find-Error", nil)
	mFindLatency := metrics.GetOrRegisterTimer(`Inventory.FindByEpc.Find-Latency`, nil)

	var tag Tag

	execFunc := func(collection *mgo.Collection) error {
		return collection.Find(bson.M{"epc": epc}).One(&tag)
	}
	retrieveTimer := time.Now()
	if err := dbs.Execute(tagCollection, execFunc); err != nil {
		// If the error was because item does not exist, return empty tag and no error
		if err == mgo.ErrNotFound {
			return Tag{}, nil
		}
		mFindErr.Update(1)
		return Tag{}, errors.Wrap(err, "db.tags.find()")
	}
	mFindLatency.Update(time.Since(retrieveTimer))

	mSuccess.Update(1)
	return tag, nil
}

// RetrieveOne retrieves One tag from database
func RetrieveOne(dbs *db.DB, query url.Values) (Tag, error) {

	// Metrics
	metrics.GetOrRegisterGauge(`Inventory.RetrieveOne.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Inventory.RetrieveOne.Success`, nil)
	mFindErr := metrics.GetOrRegisterGauge("Inventory.RetrieveOne.Find-Error", nil)
	mInputErr := metrics.GetOrRegisterGauge("Inventory.RetrieveOne.Input-Error", nil)
	mFindLatency := metrics.GetOrRegisterTimer(`Inventory.RetrieveOne.Find-Latency`, nil)

	var object []Tag

	// Else, run filter query and return slice of Tag
	execFunc := func(collection *mgo.Collection) error {
		return odata.ODataQuery(query, &object, collection)
	}

	retrieveTimer := time.Now()
	if err := dbs.Execute(tagCollection, execFunc); err != nil {
		if errors.Cause(err) == odata.ErrInvalidInput {
			mInputErr.Update(1)
			return Tag{}, errors.Wrap(web.ErrInvalidInput, err.Error())
		}
		mFindErr.Update(1)
		return Tag{}, errors.Wrap(err, "db.tags.find()")
	}
	mFindLatency.Update(time.Since(retrieveTimer))

	if len(object) > 0 {
		mSuccess.Update(1)
		return object[0], nil
	}

	mSuccess.Update(1)
	return Tag{}, nil
}

// RetrieveAll retrieves all tags from the database
func RetrieveAll(dbs *db.DB) ([]Tag, error) {

	// Metrics
	metrics.GetOrRegisterGauge(`Inventory.RetrieveAll.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Inventory.RetrieveAll.Success`, nil)
	mFindErr := metrics.GetOrRegisterGauge("Inventory.RetrieveAll.Find-Error", nil)
	mFindLatency := metrics.GetOrRegisterTimer(`Inventory.RetrieveAll.Find-Latency`, nil)

	var object []Tag

	execFunc := func(collection *mgo.Collection) error {
		return collection.Find(nil).All(&object)
	}

	retrieveTimer := time.Now()
	if err := dbs.Execute(tagCollection, execFunc); err != nil {
		mFindErr.Update(1)
		return nil, errors.Wrap(err, "Error in retrieving all the tags")
	}
	mFindLatency.Update(time.Since(retrieveTimer))

	if len(object) > 0 {
		mSuccess.Update(1)
		return object, nil
	}

	mSuccess.Update(1)
	return nil, nil
}

// Replace bulk upserts tags into database
func Replace(dbs *db.DB, tag *[]Tag) error {

	// Metrics
	metrics.GetOrRegisterGauge(`Inventory.Replace.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Inventory.Replace.Success`, nil)
	mValidationErr := metrics.GetOrRegisterGauge(`Inventory.Replace.Validation-Error`, nil)
	mBulkErr := metrics.GetOrRegisterGauge(`Inventory.Replace.Bulk-Error`, nil)

	if len(*tag) == 0 {
		return nil
	}

	//Create Bulk upsert interface input
	tags := make([]interface{}, len(*tag)*2)

	// Validate and prepare data into pairs of key,obj
	upsertIndex := 0
	for _, item := range *tag {
		// Validate empty epc_code
		if item.Epc == "" {
			mValidationErr.Update(1)
			return errors.Wrap(web.ErrValidation, "Unable to add new tag with empty EPC code")
		}

		// Upsert requires a pair of upserting instructions (select, obj)
		// e.g. ["key",obj,"key2",obj2,"key3", obj3]
		tags[upsertIndex] = bson.M{"epc": item.Epc}
		tags[upsertIndex+1] = bson.M{"$set": bson.M{
			"epc":              item.Epc,
			"uri":              item.URI,
			"productId":        item.ProductID,
			"filter_value":     item.FilterValue,
			"tid":              item.Tid,
			"encode_format":    item.EpcEncodeFormat,
			"facility_id":      item.FacilityID,
			"event":            item.Event,
			"arrived":          item.Arrived,
			"last_read":        item.LastRead,
			"source":           item.Source,
			"location_history": item.LocationHistory,
			"epc_state":        item.EpcState,
			"qualified_state":  item.QualifiedState,
			"epc_context":      item.EpcContext,
			"ttl":              item.TTL,
		}}

		upsertIndex += 2
	}

	if err := performUpsert(tags, dbs); err != nil {
		mBulkErr.Update(1)
		return err
	}

	mSuccess.Update(1)
	return nil
}

// Delete removes tag from database based on epc
// nolint :dupl
func Delete(dbs *db.DB, epc string) error {

	// Metrics
	metrics.GetOrRegisterGauge(`Inventory.Delete.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Inventory.Delete.Success`, nil)
	mDeleteErr := metrics.GetOrRegisterGauge(`Inventory.Delete.Delete-Error`, nil)
	mErrNotFound := metrics.GetOrRegisterGauge(`Inventory.Delete.NotFound-Error`, nil)
	mDeleteLatency := metrics.GetOrRegisterTimer(`Inventory.Delete.Delete-Latency`, nil)

	execFunc := func(collection *mgo.Collection) error {
		return collection.Remove(bson.M{"epc": epc})
	}

	deleteTimer := time.Now()
	if err := dbs.Execute(tagCollection, execFunc); err != nil {
		if err == mgo.ErrNotFound {
			mErrNotFound.Update(1)
			return web.ErrNotFound
		}
		mDeleteErr.Update(1)
		return errors.Wrap(err, "db.tag.Delete()")
	}
	mDeleteLatency.Update(time.Since(deleteTimer))

	mSuccess.Update(1)
	return nil
}

// DeleteTagCollection removes tag collection from database
// nolint :dupl
func DeleteTagCollection(dbs *db.DB) error {

	// Metrics
	metrics.GetOrRegisterGauge(`Inventory.DeleteTagCollection.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Inventory.DeleteTagCollection.Success`, nil)
	mDeleteAllErr := metrics.GetOrRegisterGauge(`Inventory.DeleteTagCollection.DeleteTagCollection-Error`, nil)
	mErrNotFound := metrics.GetOrRegisterGauge(`Inventory.DeleteTagCollection.NotFound-Error`, nil)
	mDeleteAllLatency := metrics.GetOrRegisterTimer(`Inventory.DeleteTagCollection.DeleteTagCollection-Latency`, nil)

	execFunc := func(collection *mgo.Collection) error {
		/* To remove all the documents of a given collection, call RemoveAll with an empty selector. */
		_, error := collection.RemoveAll(nil)
		return error
	}

	deleteTagCollectionTimer := time.Now()
	if err := dbs.Execute(tagCollection, execFunc); err != nil {
		if err == mgo.ErrNotFound {
			mErrNotFound.Update(1)
			return web.ErrNotFound
		}
		mDeleteAllErr.Update(1)
		return errors.Wrap(err, "db.tag.DeleteTagCollection()")
	}
	mDeleteAllLatency.Update(time.Since(deleteTagCollectionTimer))

	mSuccess.Update(1)
	return nil
}

// DecodeTagData extracts a ProductID and URI from tag data, according to the
// configured tag decoders. If none of the decoders can successfully decode the
// data, it returns `encodingInvalid` for both.
func DecodeTagData(tagData string) (productID string, URI string) {
	for idx, decoder := range config.AppConfig.TagDecoders {
		var err error
		if productID, URI, err = decoder.Decode(tagData); err == nil {
			return
		}
		log.Warnf("decoder %d (%s) unable to decode tag data: %s",
			idx+1, decoder.Type(), err)
		gaugeName := fmt.Sprintf("Inventory.DecodeTagData.%s", decoder.Type())
		metrics.GetOrRegisterGauge(gaugeName, nil).Update(1)
	}

	log.Error("unable to decode tag data with any of the configured decoders")
	metrics.GetOrRegisterGauge(`Inventory.DecodeTagData.CalculateProductCodeError`,
		nil).Update(1)
	return encodingInvalid, encodingInvalid
}

// performUpsert updates or inserts tags
func performUpsert(tags []interface{}, dbs *db.DB) error {

	// Metrics
	metrics.GetOrRegisterGauge(`Inventory.performUpsert.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Inventory.performUpsert.Success`, nil)
	mBulkErr := metrics.GetOrRegisterGauge("Inventory.performUpsert.Bulk-Upsert-Error", nil)
	mBulkLatency := metrics.GetOrRegisterTimer(`Inventory.performUpsert.Upsert-Latency`, nil)

	bulkFunc := func(collection *mgo.Collection) *mgo.Bulk {
		return collection.Bulk()
	}

	bulk := dbs.ExecuteBulk(tagCollection, bulkFunc)
	bulk.Unordered()

	upsertTimer := time.Now()
	// Upsert in batch of 500 due to mongodb 1000 max ops limitation
	// 1000 because is a pair of instructions. Thus, 500 items means 1000 size
	if len(tags) > mongoMaxOps {
		range1 := 0
		range2 := mongoMaxOps
		lastBatch := false

		for {

			// Queue batches of 1000 elements which translates to 500 operations
			if range2 < len(tags) {
				bulk.Upsert(tags[range1:range2]...)
			} else {
				// Last batch
				bulk.Upsert(tags[range1:]...)
				lastBatch = true
			}

			if _, err := bulk.Run(); err != nil {
				mBulkErr.Update(1)
				return errors.Wrap(err, "Unable to insert tags in database (db.bulk.upsert)")
			}

			// Flush any queued data
			// Reinitialize bulk after being flushed
			bulk = nil
			bulk = dbs.ExecuteBulk(tagCollection, bulkFunc)
			bulk.Unordered()

			// Break after last batch
			if lastBatch {
				break
			}
			range1 = range2
			range2 += mongoMaxOps
		}

	} else {
		bulk.Upsert(tags...)
		if _, err := bulk.Run(); err != nil {
			mBulkErr.Update(1)
			return errors.Wrap(err, "Unable to insert tags in database (db.bulk.upsert)")
		}
	}
	mBulkLatency.Update(time.Since(upsertTimer))
	mSuccess.Update(1)
	return nil
}

// Update updates a tag in the database
func Update(dbs *db.DB, selector map[string]interface{}, updateObject map[string]interface{}) error {

	// Metrics
	metrics.GetOrRegisterGauge(`Inventory.Update.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Inventory.Update.Success`, nil)
	mUpdateErr := metrics.GetOrRegisterGauge(`Inventory.Update.Update-Error`, nil)
	mErrNotFound := metrics.GetOrRegisterGauge(`Inventory.Update.NotFound-Error`, nil)
	mUpdateLatency := metrics.GetOrRegisterTimer(`Inventory.Update.Update-Latency`, nil)

	execFunc := func(collation *mgo.Collection) error {
		return collation.Update(selector, bson.M{"$set": updateObject})
	}

	updateTimer := time.Now()
	if err := dbs.Execute(tagCollection, execFunc); err != nil {
		if err == mgo.ErrNotFound {
			mErrNotFound.Update(1)
			return web.ErrNotFound
		}
		mUpdateErr.Update(1)
		return errors.Wrap(err, "db.tag.Update()")
	}
	mUpdateLatency.Update(time.Since(updateTimer))

	mSuccess.Update(1)
	return nil
}

// UpdateTTLIndexForTags updates the expireAfterSeconds value in ttl index
// nolint :dupl
func UpdateTTLIndexForTags(dbs *db.DB, purgingSeconds int) error {

	updateCommand := bson.D{{"collMod", tagCollection}, {"index", bson.D{{"keyPattern", bson.D{{"ttl", 1}}}, {"expireAfterSeconds", purgingSeconds}}}}
	var result interface{}

	execFunc := func(collection *mgo.Collection) error {
		return collection.Database.Run(updateCommand, result)
	}

	return dbs.Execute(tagCollection, execFunc)
}
