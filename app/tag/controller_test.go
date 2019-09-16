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
	"crypto/rand"
	"fmt"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/encodingscheme"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/integrationtest"
	"math/big"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"

	db "github.impcloud.net/RSP-Inventory-Suite/go-dbWrapper"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/web"
)

var testEpc = "3014186A343E214000000009"

var dbHost integrationtest.DBHost

func TestMain(m *testing.M) {
	dbHost = integrationtest.InitHost("tag_test")
	os.Exit(m.Run())
}

//nolint:dupl
func TestNoDataRetrieve(t *testing.T) {

	masterDb := dbHost.CreateDB(t)
	defer masterDb.Close()

	copySession := masterDb.CopySession()
	defer copySession.Close()

	clearAllData(t, copySession)

	testURL, err := url.Parse("http://localhost/test?$top=10&$select=name,age")
	if err != nil {
		t.Error("failed to parse test url")
	}

	_, _, _, err = Retrieve(copySession, testURL.Query(), config.AppConfig.ResponseLimit)
	if err != nil {
		t.Error("Unable to retrieve tags")
	}
}

func TestWithDataRetrieve(t *testing.T) {

	masterDb := dbHost.CreateDB(t)
	defer masterDb.Close()

	copySession := masterDb.CopySession()
	defer copySession.Close()

	clearAllData(t, copySession)

	insertSample(t, copySession)

	testURL, err := url.Parse("http://localhost/test?$top=10")
	if err != nil {
		t.Error("failed to parse test url")
	}

	tags, _, paging, err := Retrieve(copySession, testURL.Query(), config.AppConfig.ResponseLimit)

	if err != nil {
		t.Error("Unable to retrieve tags")
	}

	if paging.Cursor == "" {
		t.Error("Cursor is empty")
	}

	tagSlice := reflect.ValueOf(tags)

	if tagSlice.Len() <= 0 {
		t.Error("Unable to retrieve tags")
	}
}

func TestCursor(t *testing.T) {

	masterDb := dbHost.CreateDB(t)
	defer masterDb.Close()

	copySession := masterDb.CopySession()
	defer copySession.Close()

	clearAllData(t, copySession)
	insertSample(t, copySession)

	testURL, err := url.Parse("http://localhost/test?$top=10")
	if err != nil {
		t.Error("failed to parse test url")
	}

	tags, _, pagingfirst, err := Retrieve(copySession, testURL.Query(), config.AppConfig.ResponseLimit)

	if err != nil {
		t.Error("Unable to retrieve tags")
	}

	if pagingfirst.Cursor == "" {
		t.Error("Cursor is empty")
	}

	cFirst := pagingfirst.Cursor
	tagSlice := reflect.ValueOf(tags)

	if tagSlice.Len() <= 0 {
		t.Error("Unable to retrieve tags")
	}

	// Initiating second http request to check if first sceond cursor or not same
	insertSampleCustom(t, copySession, "cursor")

	cursorTestURL, err := url.Parse("http://localhost/test?$filter=_id gt '" + url.QueryEscape(cFirst) + "'&$top=10")
	if err != nil {
		t.Error("failed to parse test url")
	}

	ctags, _, pagingnext, err := Retrieve(copySession, cursorTestURL.Query(), config.AppConfig.ResponseLimit)

	if err != nil {
		t.Error("Unable to retrieve tags")
		fmt.Println(err.Error())
	}

	if pagingnext.Cursor == "" {
		t.Error("Cursor is empty")
	}

	cSecond := pagingnext.Cursor
	tagSlice2 := reflect.ValueOf(ctags)

	if tagSlice2.Len() <= 0 {
		t.Error("Unable to retrieve tags")
	}

	if cFirst == cSecond {
		t.Error("paging failed ")
	}

}

//nolint:dupl
func TestRetrieveCount(t *testing.T) {
	testCases := []string{
		"http://localhost/test?$count",
		"http://localhost/test?$count&$filter=startswith(epc,'3')",
	}

	dbSession := dbHost.CreateDB(t)
	defer dbSession.Close()

	for _, item := range testCases {
		testURL, err := url.Parse(item)
		if err != nil {
			t.Error("failed to parse test url")
		}

		retrieveCountTest(t, testURL, dbSession)
	}
}

func retrieveCountTest(t *testing.T, testURL *url.URL, session *db.DB) {
	results, count, _, err := Retrieve(session, testURL.Query(), config.AppConfig.ResponseLimit)
	if results != nil {
		t.Error("expecting results to be nil")
	}
	if count == nil {
		t.Error("expecting CountType result")
	}
	if err != nil {
		t.Error("Unable to retrieve total count")
	}
}

func TestRetrieveInlineCount(t *testing.T) {

	testURL, err := url.Parse("http://localhost/test?$filter=test eq 1&$inlinecount=allpages")
	if err != nil {
		t.Error("failed to parse test url")
	}

	masterDB := dbHost.CreateDB(t)
	defer masterDB.Close()

	results, count, _, err := Retrieve(masterDB, testURL.Query(), config.AppConfig.ResponseLimit)

	if results == nil {
		t.Error("expecting results to not be nil")
	}

	if count == nil {
		t.Error("expecting inlinecount result")
	}

	if err != nil {
		t.Error("Unable to retrieve", err.Error())
	}
}

func TestNoDataRetrieveOne(t *testing.T) {

	masterDb := dbHost.CreateDB(t)
	defer masterDb.Close()

	copySession := masterDb.CopySession()
	defer copySession.Close()

	clearAllData(t, copySession)

	myMap := make(map[string][]string)
	myMap["$select"] = append(myMap["$select"], "name")

	noTagFound, err := RetrieveOne(copySession, myMap)
	if err != nil {
		t.Error("Unable to retrieve tags")
	}

	if noTagFound.IsTagReadByRspController() {
		t.Error("Did not return empty tag")
	}
}

func TestWithDataRetrieveOne(t *testing.T) {

	masterDb := dbHost.CreateDB(t)
	defer masterDb.Close()

	copySession := masterDb.CopySession()
	defer copySession.Close()

	insertSample(t, copySession)

	myMap := make(map[string][]string)
	myMap["$select"] = append(myMap["$select"], "epc")

	gotTag, err := RetrieveOne(copySession, myMap)
	if err != nil {
		t.Error("Unable to retrieve tags")
	}

	if !gotTag.IsTagReadByRspController() {
		t.Error("Unable to retrieve tags")
	}
}

func TestRetrieveOdataAllWithOdataQuery(t *testing.T) {

	masterDb := dbHost.CreateDB(t)
	defer masterDb.Close()

	copySession := masterDb.CopySession()
	defer copySession.Close()

	tagArray := make([]Tag, 2)

	var tag0 Tag
	tag0.Epc = "303401D6A415B5C000000002"
	tag0.FacilityID = "facility1"
	tagArray[0] = tag0

	var tag1 Tag
	tag1.Epc = "303401D6A415B5C000000001"
	tag0.FacilityID = "facility2"
	tagArray[1] = tag1

	err := Replace(copySession, &tagArray)
	if err != nil {
		t.Error("Unable to insert tags", err.Error())
	}

	odataMap := make(map[string][]string)
	odataMap["$filter"] = append(odataMap["$filter"], "facility_id eq facility1")

	tags, err := RetrieveOdataAll(copySession, odataMap)
	if err != nil {
		t.Error("Error in retrieving tags based on odata query")
	} else if len(tags) != 1 {
		t.Error("Expected one tag to be retrieved based on query")
	}
}

func TestRetrieveOdataAllNoOdataQuery(t *testing.T) {

	masterDb := dbHost.CreateDB(t)
	defer masterDb.Close()

	copySession := masterDb.CopySession()
	defer copySession.Close()

	numOfSamples := 600
	tagSlice := make([]Tag, numOfSamples)
	epcSlice := generateSequentialEpcs("3014", 0, int64(numOfSamples))

	for i := 0; i < numOfSamples; i++ {
		var tag Tag
		tag.Epc = epcSlice[i]
		tag.Source = "fixed"
		tag.Event = "arrived"
		tagSlice[i] = tag
	}

	err := Replace(copySession, &tagSlice)
	if err != nil {
		t.Errorf("Unable to insert tags in bulk: %s", err.Error())
	}

	odataMap := make(map[string][]string)
	tags, err := RetrieveOdataAll(copySession, odataMap)
	if err != nil  {
		t.Error("Error in retrieving tags")
	} else if len(tags) != numOfSamples {
		t.Error("Number of tags in database and number of tags retrieved do not match")
	}
}

func TestInsert(t *testing.T) {
	masterDb := dbHost.CreateDB(t)
	defer masterDb.Close()

	dbs := masterDb.CopySession()
	defer dbs.Close()
	insertSample(t, dbs)
}

func TestDataReplace(t *testing.T) {

	masterDb := dbHost.CreateDB(t)
	defer masterDb.Close()

	copySession := masterDb.CopySession()
	defer copySession.Close()

	tagArray := make([]Tag, 2)

	var tag0 Tag
	tag0.Epc = testEpc
	tag0.Tid = t.Name() + "0"
	tag0.Source = "fixed"
	tag0.Event = "arrived"
	tagArray[0] = tag0

	var tag1 Tag
	tag1.Epc = "303401D6A415B5C000000001"
	tag1.Tid = t.Name() + "1"
	tag1.Source = "handheld"
	tag1.Event = "arrived"
	tagArray[1] = tag1

	err := Replace(copySession, &tagArray)
	if err != nil {
		t.Error("Unable to replace tags", err.Error())
	}
}

func TestRetrieveSizeLimitWithTop(t *testing.T) {

	var sizeLimit = 1

	// Trying to return more than 1 result
	testURL, err := url.Parse("http://localhost/test?$inlinecount=allpages&$top=2")
	if err != nil {
		t.Error("Failed to parse test URL")
	}

	masterDb := dbHost.CreateDB(t)
	defer masterDb.Close()

	copySession := masterDb.CopySession()
	defer copySession.Close()

	numOfSamples := 10

	tagSlice := make([]Tag, numOfSamples)

	epcSlice := generateSequentialEpcs("3014", 0, int64(numOfSamples))

	for i := 0; i < numOfSamples; i++ {
		var tag Tag
		tag.Epc = epcSlice[i]
		tag.Source = "fixed"
		tag.Event = "arrived"
		tagSlice[i] = tag
	}

	if replaceErr := Replace(copySession, &tagSlice); replaceErr != nil {
		t.Errorf("Unable to replace tags: %s", replaceErr.Error())
	}

	results, count, _, err := Retrieve(copySession, testURL.Query(), sizeLimit)
	if err != nil {
		t.Errorf("Retrieve failed with error %v", err.Error())
	}

	resultSlice := reflect.ValueOf(results)

	if resultSlice.Len() > sizeLimit {
		t.Errorf("Error retrieving results with size limit. Expected: %d , received: %d", sizeLimit, count.Count)
	}

}

func TestRetrieveSizeLimitInvalidTop(t *testing.T) {

	var sizeLimit = 1

	// Trying to return more than 1 result
	testURL, err := url.Parse("http://localhost/test?$inlinecount=allpages&$top=string")
	if err != nil {
		t.Error("Failed to parse test URL")
	}

	masterDb := dbHost.CreateDB(t)
	defer masterDb.Close()

	copySession := masterDb.CopySession()
	defer copySession.Close()

	_, _, _, err = Retrieve(copySession, testURL.Query(), sizeLimit)
	if err == nil {
		t.Errorf("Expecting an error for invalid $top value")
	}

}

func TestDataReplace_Bulk(t *testing.T) {

	masterDb := dbHost.CreateDB(t)
	defer masterDb.Close()

	copySession := masterDb.CopySession()
	defer copySession.Close()

	numOfSamples := 600

	tagSlice := make([]Tag, numOfSamples)

	epcSlice := generateSequentialEpcs("3014", 0, int64(numOfSamples))

	for i := 0; i < numOfSamples; i++ {
		var tag Tag
		tag.Epc = epcSlice[i]
		tag.Source = "fixed"
		tag.Event = "arrived"
		tagSlice[i] = tag
	}

	err := Replace(copySession, &tagSlice)
	if err != nil {
		t.Errorf("Unable to replace tags: %s", err.Error())
	}

	// randomly pick one to test to save testing time
	indBig, randErr := rand.Int(rand.Reader, big.NewInt(int64(numOfSamples)))
	var testIndex int
	if randErr != nil {
		testIndex = 0
	} else {
		testIndex = int(indBig.Int64())
	}
	epcToTest := epcSlice[testIndex]
	gotTag, err := FindByEpc(copySession, epcToTest)

	if err != nil {
		t.Error("Unable to retrieve tags")
	}

	if !gotTag.IsTagReadByRspController() {
		t.Error("Unable to retrieve tags")
	}
}

// nolint :dupl
func TestDelete(t *testing.T) {

	masterDb := dbHost.CreateDB(t)
	defer masterDb.Close()

	dbs := masterDb.CopySession()

	// have to insert something before we can delete it
	insertSample(t, dbs)

	if err := Delete(dbs, t.Name()); err != nil {
		if err == web.ErrNotFound {
			t.Fatal("Tag Not found, nothing to delete")
		}
		t.Error("Unable to delete tag")
	}
}

// nolint :dupl
func TestDeleteTagCollection(t *testing.T) {

	masterDb := dbHost.CreateDB(t)
	defer masterDb.Close()

	dbs := masterDb.CopySession()

	// have to insert something before we can delete it
	insertSample(t, dbs)

	if err := DeleteTagCollection(dbs); err != nil {
		if err == web.ErrNotFound {
			t.Fatal("Tag Not found, nothing to delete")
		}
		t.Error("Unable to delete tag")
	}
}

//nolint:dupl
func TestDelete_nonExistItem(t *testing.T) {
	masterDb := dbHost.CreateDB(t)
	defer masterDb.Close()
	dbs := masterDb.CopySession()

	// we will try to delete random gibberish

	if err := Delete(dbs, "emptyId"); err != nil {
		if err == web.ErrNotFound {
			// because we didn't find it, it should succeed
			t.Log("Tag NOT FOUND, this is the expected result")
		} else {
			t.Error("Expected to not be able to delete")
		}
	}
}

func TestUpdate(t *testing.T) {
	masterDb := dbHost.CreateDB(t)
	defer masterDb.Close()

	dbs := masterDb.CopySession()
	defer dbs.Close()

	epc := "30143639F8419105417AED6F"
	facilityID := "TestFacility"
	// insert sample data
	var tagArray = []Tag{
		{
			Epc:        epc,
			FacilityID: facilityID,
		},
	}

	err := Replace(dbs, &tagArray)
	if err != nil {
		t.Error("Unable to insert tag", err.Error())
	}

	// maps needed for update
	selectorMap := make(map[string]interface{})
	selectorMap["epc"] = epc
	selectorMap["facility_id"] = facilityID
	objectMap := make(map[string]interface{})
	objectMap["qualified_state"] = "sold"

	err = Update(dbs, selectorMap, objectMap)
	if err != nil {
		t.Error("Unable to update the tag", err.Error())
	}

	// verify that update was successful
	tag, err := FindByEpc(dbs, epc)
	if err != nil {
		t.Errorf("Error trying to find tag by epc %s", err.Error())
	} else if !tag.IsTagReadByRspController() {
		if tag.QualifiedState != "sold" {
			t.Fatal("Qualified_state update failed")
		}
	}

	// clean data to do negative tests
	clearAllData(t, dbs)
	err = Update(dbs, selectorMap, objectMap)
	if err == nil {
		t.Error("Tag not found error not caught")
	}
}

func TestUpdateTTLIndex(t *testing.T) {
	masterDb := dbHost.CreateDB(t)
	defer masterDb.Close()

	dbs := masterDb.CopySession()
	defer dbs.Close()

	ttlIndex := "ttl"
	purgingSeconds := 1800

	// Add index before updating
	if err := addIndex(t, dbs, ttlIndex); err != nil {
		t.Errorf("Error addIndex(): %s", err.Error())
	}

	if err := UpdateTTLIndexForTags(dbs, purgingSeconds); err != nil {
		t.Errorf("Error UpdateTTLIndexForTags(): %s", err.Error())
	}

	execFunc := func(collection *mgo.Collection) error {
		indexes, err := collection.Indexes()
		if err != nil {
			return err
		}
		// check if ttl index was updated
		for i, v := range indexes {
			if strings.Contains(v.Name, ttlIndex) {
				updatedExpireTime := int(v.ExpireAfter / time.Second)
				if updatedExpireTime != purgingSeconds {
					t.Error("Update of ttl index failed expected", purgingSeconds, "got", updatedExpireTime)
				}
			} else if i == (len(indexes) - 1) {
				t.Error("ttl index not found")
			}

		}
		return nil
	}

	if err := dbs.Execute(tagCollection, execFunc); err != nil {
		t.Error("UpdateTTLIndexForTags test failed", err.Error())
	}

	// Clear data and negative testing
	if err := dropIndex(t, dbs, ttlIndex); err != nil {
		t.Errorf("Error dropIndex(): %s", err.Error())
	}

	err := UpdateTTLIndexForTags(dbs, purgingSeconds)
	if err == nil {
		t.Error("Update should have failed as ttl index does not exist", err.Error())
	}
}

func addIndex(t *testing.T, mydb *db.DB, indexName string) error {

	index := mgo.Index{
		Key:         []string{indexName},
		Unique:      false,
		DropDups:    false,
		Background:  false,
		ExpireAfter: time.Duration(60) * time.Second,
	}

	execFunc := func(collection *mgo.Collection) error {
		return collection.EnsureIndex(index)
	}

	if err := mydb.Execute(tagCollection, execFunc); err != nil {
		t.Error("Add index failed", err.Error())
	}

	return nil
}

func dropIndex(t *testing.T, mydb *db.DB, index string) error {

	execFunc := func(collection *mgo.Collection) error {
		return collection.DropIndex(index)
	}

	if err := mydb.Execute(tagCollection, execFunc); err != nil {
		t.Error("Drop index failed", err.Error())
	}

	return nil
}

func TestFindByEpc_found(t *testing.T) {
	masterDb := dbHost.CreateDB(t)
	defer masterDb.Close()

	dbs := masterDb.CopySession()
	defer dbs.Close()

	epc := t.Name()
	insertSampleCustom(t, dbs, epc)

	tag, err := FindByEpc(dbs, epc)
	if err != nil {
		t.Errorf("Error trying to find tag by epc %s", err.Error())
	} else if !tag.IsTagReadByRspController() {
		t.Errorf("Expected to find a tag with epc: %s", epc)
	} else if tag.Epc != epc {
		t.Error("Expected found tag epc to be equal to the input epc")
	}

	err = Delete(dbs, epc)
	if err != nil {
		t.Error("Error on delete")
	}
}

func TestFindByEpc_notFound(t *testing.T) {
	masterDb := dbHost.CreateDB(t)
	defer masterDb.Close()

	dbs := masterDb.CopySession()
	defer dbs.Close()

	epc := t.Name()
	tag, err := FindByEpc(dbs, epc)
	if err != nil {
		t.Errorf("Error trying to find tag by epc %s", err.Error())
	} else if tag.IsTagReadByRspController() {
		t.Errorf("Expected to NOT find a tag with epc: %s", epc)
	}
}

func TestCalculateGtin(t *testing.T) {
	config.AppConfig.TagDecoders = []encodingscheme.TagDecoder{encodingscheme.SGTIN96Decoder()}
	validEpc := "303402662C3A5F904C19939D"
	gtin, _ := DecodeTagData(validEpc)
	if gtin == UndefinedProductID {
		t.Errorf("Error trying to calculate valid epc %s", validEpc)
	}
}

func TestCalculateInvalidGtin(t *testing.T) {
	setSGTINOnlyDecoderConfig()
	epcSlice := generateSequentialEpcs("0014", 0, 1)
	gtin, _ := DecodeTagData(epcSlice[0])
	if gtin != encodingInvalid {
		t.Errorf("Unexpected result calculating invalid epc %s, expected %s", epcSlice[0], UndefinedProductID)
	}
}

func setSGTINOnlyDecoderConfig() {
	config.AppConfig.TagDecoders = []encodingscheme.TagDecoder{
		encodingscheme.SGTIN96Decoder(),
	}
}

func setMixedDecoderConfig(t *testing.T) {
	decoder, err := encodingscheme.NewProprietary(
		"test.com", "2019-01-01",
		"header.serialNumber.productID", "8.48.40")
	if err != nil {
		t.Fatal(err)
	}
	config.AppConfig.TagDecoders = []encodingscheme.TagDecoder{
		encodingscheme.SGTIN96Decoder(),
		decoder,
	}
}

func TestCalculateProductCode(t *testing.T) {
	setMixedDecoderConfig(t)
	validEpc := "0F00000000000C00000014D2"
	expectedWrin := "14D2"
	productID, _ := DecodeTagData(validEpc)
	if productID == UndefinedProductID {
		t.Errorf("Error trying to calculate valid epc %s", validEpc)
	}
	if productID != expectedWrin {
		t.Errorf("Error trying to calculate valid epc %s", validEpc)
	}
}

func TestCalculateSGTINTagUrn(t *testing.T) {
	setSGTINOnlyDecoderConfig()
	_, uri := DecodeTagData("3034257BF400B7800004CB2F")
	expectedURI := encodingscheme.EPCPureURIPrefix + "0614141.000734.314159"
	if uri != expectedURI {
		t.Errorf("Error trying to calculate uri.  Got %s, expected %s", uri, expectedURI)
	}
}

func TestCalculateProprietaryTagUrn(t *testing.T) {
	setMixedDecoderConfig(t)
	expectedURI := "tag:test.com,2019-01-01:15.12.5330"
	_, uri := DecodeTagData("0F00000000000C00000014D2")
	if uri != expectedURI {
		t.Errorf("Error trying to calculate uri.  Got %s, expected %s", uri, expectedURI)
	}
}

func insertSample(t *testing.T, mydb *db.DB) {
	insertSampleCustom(t, mydb, t.Name())
}

func insertSampleCustom(t *testing.T, mydb *db.DB, sampleID string) {
	var tag Tag

	tag.Epc = sampleID

	if err := insert(mydb, tag); err != nil {
		t.Error("Unable to insert tag")
	}
}

// nolint :dupl
func clearAllData(t *testing.T, mydb *db.DB) {
	execFunc := func(collection *mgo.Collection) error {
		_, err := collection.RemoveAll(bson.M{})
		return err
	}

	if err := mydb.Execute(tagCollection, execFunc); err != nil {
		t.Error("Unable to delete collection")
	}
}

// nolint :dupl
func insert(dbs *db.DB, tag Tag) error {

	execFunc := func(collection *mgo.Collection) (*mgo.ChangeInfo, error) {
		return collection.Upsert(bson.M{"epc": tag.Epc}, &tag)
	}

	const tagCollection = "tags"
	if _, err := dbs.ExecuteWithChangeInfo(tagCollection, execFunc); err != nil {
		return errors.Wrap(err, "db.tag.upsert()")
	}

	return nil
}

//nolint:unparam
func generateSequentialEpcs(header string, offset int64, limit int64) []string {
	digits := 24 - len(header)
	epcs := make([]string, limit)
	for i := int64(0); i < limit; i++ {
		epcs[i] = strings.ToUpper(fmt.Sprintf("%s%0"+strconv.Itoa(digits)+"s", header, strconv.FormatInt(offset+i, 16)))
	}
	return epcs
}
