/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package tag

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/app/config"
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/pkg/encodingscheme"
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/pkg/integrationtest"
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/pkg/web"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"math/big"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

var testEpc = "3014186A343E214000000009"

var dbHost integrationtest.DBHost

func TestMain(m *testing.M) {
	dbHost = integrationtest.InitHost("tag_test")
	exitCode := m.Run()
	dbHost.Close()
	os.Exit(exitCode)
}

// nolint :dupl
func TestDelete(t *testing.T) {
	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	// have to insert something before we can delete it
	insertSample(t, testDB.DB)

	selectQuery := fmt.Sprintf(`DELETE FROM %s`,
		pq.QuoteIdentifier(tagsTable),
	)

	_, err := testDB.DB.Exec(selectQuery)
	if err != nil {
		if err == web.ErrNotFound {
			t.Fatal("Tag Not found, nothing to delete")
		}
		t.Error("Unable to delete tag", err)
	}
}

//nolint:dupl
func TestNoDataRetrieve(t *testing.T) {

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	clearAllData(t, testDB.DB)

	testURL, err := url.Parse("http://localhost/test?$top=10&$select=name,age")
	if err != nil {
		t.Error("failed to parse test url")
	}

	_, _, err = Retrieve(testDB.DB, testURL.Query(), config.AppConfig.ResponseLimit)
	if err != nil {
		t.Error("Unable to retrieve tags")
	}
}

func TestWithDataRetrieve(t *testing.T) {

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	clearAllData(t, testDB.DB)

	insertSample(t, testDB.DB)

	testURL, err := url.Parse("http://localhost/test?$top=10")
	if err != nil {
		t.Error("failed to parse test url")
	}

	tags, _, err := Retrieve(testDB.DB, testURL.Query(), config.AppConfig.ResponseLimit)

	if err != nil {
		t.Error("Unable to retrieve tags")
	}

	//if paging.Cursor == "" {
	//	t.Error("Cursor is empty")
	//}

	tagSlice := reflect.ValueOf(tags)

	if tagSlice.Len() <= 0 {
		t.Error("Unable to retrieve tags")
	}
}

/*func TestCursor(t *testing.T) {

	testDB := dbTestSetup(t)
	defer testDB.Close()

	clearAllData(t, testDB.DB)
	insertSample(t, testDB.DB)

	testURL, err := url.Parse("http://localhost/test?$top=10")
	if err != nil {
		t.Error("failed to parse test url")
	}

	tags, _, pagingfirst, err := Retrieve(testDB.DB, testURL.Query(), config.AppConfig.ResponseLimit)

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
	insertSampleCustom(t, testDB.DB, "cursor")

	cursorTestURL, err := url.Parse("http://localhost/test?$filter=_id gt '" + url.QueryEscape(cFirst) + "'&$top=10")
	if err != nil {
		t.Error("failed to parse test url")
	}

	ctags, _, pagingnext, err := Retrieve(testDB.DB, cursorTestURL.Query(), config.AppConfig.ResponseLimit)

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

}*/

//nolint:dupl
func TestRetrieveCount(t *testing.T) {
	testCases := []string{
		"http://localhost/test?$count",
		"http://localhost/test?$count&$filter=startswith(epc,'3')",
	}

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	for _, item := range testCases {
		testURL, err := url.Parse(item)
		if err != nil {
			t.Error("failed to parse test url")
		}

		retrieveCountTest(t, testURL, testDB.DB)
	}
}

func retrieveCountTest(t *testing.T, testURL *url.URL, session *sql.DB) {
	results, count, err := Retrieve(session, testURL.Query(), config.AppConfig.ResponseLimit)
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

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	results, count, err := Retrieve(testDB.DB, testURL.Query(), config.AppConfig.ResponseLimit)

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

func TestRetrieveOdataAllWithOdataQuery(t *testing.T) {

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	tagArray := make([]Tag, 2)

	var tag0 Tag
	tag0.Epc = "303401D6A415B5C000000002"
	tag0.FacilityID = "facility1"
	tag0.URI = "tag1.test"
	tagArray[0] = tag0

	var tag1 Tag
	tag1.Epc = "303401D6A415B5C000000001"
	tag1.FacilityID = "facility2"
	tag1.URI = "tag2.test"
	tagArray[1] = tag1

	err := Replace(testDB.DB, tagArray)
	if err != nil {
		t.Error("Unable to insert tags", err.Error())
	}

	odataMap := make(map[string][]string)
	odataMap["$filter"] = append(odataMap["$filter"], "facility_id eq facility1")

	//tags, err := RetrieveOdataAll(testDB.DB, odataMap)
	tags, err := RetrieveOdataAll(testDB.DB, odataMap)
	if err != nil {
		t.Error("Error in retrieving tags based on odata query")
	} else {
		tagSlice, err := unmarshallTagsInterface(tags)
		if err != nil {
			t.Error("Error in unmarshalling tag interface")
		}
		if len(tagSlice) != 1 {
			t.Error("Expected one tag to be retrieved based on query", len(tagSlice))
		}
	}
	clearAllData(t, testDB.DB)
}

func unmarshallTagsInterface(tags interface{}) ([]Tag, error) {

	tagsBytes, err := json.Marshal(tags)
	if err != nil {
		return nil, errors.Wrap(err, "marshaling []interface{} to []bytes")
	}

	var tagSlice []Tag
	if err := json.Unmarshal(tagsBytes, &tagSlice); err != nil {
		return nil, errors.Wrap(err, "unmarshaling []bytes to []Tags")
	}

	return tagSlice, nil
}

func TestRetrieveOdataAllNoOdataQuery(t *testing.T) {

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	clearAllData(t, testDB.DB)

	numOfSamples := 600
	tagSlice := make([]Tag, numOfSamples)
	epcSlice := generateSequentialEpcs("3014", 0, int64(numOfSamples))

	for i := 0; i < numOfSamples; i++ {
		var tag Tag
		tag.Epc = epcSlice[i]
		tag.Source = "fixed"
		tag.Event = "arrived"
		tag.URI = "test" + "." + epcSlice[i]
		tagSlice[i] = tag
	}

	err := Replace(testDB.DB, tagSlice)
	if err != nil {
		t.Errorf("Unable to insert tags in bulk: %s", err.Error())
	}

	odataMap := make(map[string][]string)

	tags, err := RetrieveOdataAll(testDB.DB, odataMap)
	if err != nil {
		t.Error("Error in retrieving tags")
	} else {
		tagSlice, err := unmarshallTagsInterface(tags)
		if err != nil {
			t.Error("Error in unmarshalling tag interface")
		}
		if len(tagSlice) != numOfSamples {
			t.Error("Number of tags in database and number of tags retrieved do not match")
		}
	}
	clearAllData(t, testDB.DB)
}

func TestInsert(t *testing.T) {
	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	insertSample(t, testDB.DB)
}

func TestDataReplace(t *testing.T) {

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	tagArray := make([]Tag, 2)

	var tag0 Tag
	tag0.Epc = testEpc
	tag0.URI = "tag1.test"
	tag0.Tid = t.Name() + "0"
	tag0.Source = "fixed"
	tag0.Event = "arrived"
	tagArray[0] = tag0

	var tag1 Tag
	tag1.Epc = "303401D6A415B5C000000001"
	tag1.URI = "tag2.test"
	tag1.Tid = t.Name() + "1"
	tag1.Source = "handheld"
	tag1.Event = "arrived"
	tagArray[1] = tag1

	err := Replace(testDB.DB, tagArray)
	if err != nil {
		t.Error("Unable to replace tags", err.Error())
	}
	clearAllData(t, testDB.DB)
}

func TestRetrieveSizeLimitWithTop(t *testing.T) {

	var sizeLimit = 1

	// Trying to return more than 1 result
	testURL, err := url.Parse("http://localhost/test?$inlinecount=allpages&$top=2")
	if err != nil {
		t.Error("Failed to parse test URL")
	}

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	numOfSamples := 10

	tagSlice := make([]Tag, numOfSamples)

	epcSlice := generateSequentialEpcs("3014", 0, int64(numOfSamples))

	for i := 0; i < numOfSamples; i++ {
		var tag Tag
		tag.Epc = epcSlice[i]
		tag.URI = "test" + "." + epcSlice[i]
		tag.Source = "fixed"
		tag.Event = "arrived"
		tagSlice[i] = tag
	}

	if replaceErr := Replace(testDB.DB, tagSlice); replaceErr != nil {
		t.Errorf("Unable to replace tags: %s", replaceErr.Error())
	}

	results, count, err := Retrieve(testDB.DB, testURL.Query(), sizeLimit)
	if err != nil {
		t.Errorf("Retrieve failed with error %v", err.Error())
	}

	resultSlice := reflect.ValueOf(results)

	if resultSlice.Len() > sizeLimit {
		t.Errorf("Error retrieving results with size limit. Expected: %d , received: %d", sizeLimit, count.Count)
	}
	clearAllData(t, testDB.DB)
}

func TestRetrieveSizeLimitInvalidTop(t *testing.T) {

	var sizeLimit = 1

	// Trying to return more than 1 result
	testURL, err := url.Parse("http://localhost/test?$inlinecount=allpages&$top=string")
	if err != nil {
		t.Error("Failed to parse test URL")
	}

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	_, _, err = Retrieve(testDB.DB, testURL.Query(), sizeLimit)
	if err == nil {
		t.Errorf("Expecting an error for invalid $top value")
	}

}

func TestDataReplace_Bulk(t *testing.T) {

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	numOfSamples := 600

	tagSlice := make([]Tag, numOfSamples)

	epcSlice := generateSequentialEpcs("3014", 0, int64(numOfSamples))

	for i := 0; i < numOfSamples; i++ {
		var tag Tag
		tag.Epc = epcSlice[i]
		tag.URI = "test" + "." + epcSlice[i]
		tag.Source = "fixed"
		tag.Event = "arrived"
		tagSlice[i] = tag
	}

	err := Replace(testDB.DB, tagSlice)
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
	gotTag, err := FindByEpc(testDB.DB, epcToTest)

	if err != nil {
		t.Error("Unable to retrieve tags")
	}

	if !gotTag.IsTagReadByRspController() {
		t.Error("Unable to retrieve tags")
	}
	clearAllData(t, testDB.DB)
}

// nolint :dupl
func TestDeleteTagCollection(t *testing.T) {

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	// have to insert something before we can delete it
	insertSample(t, testDB.DB)

	if err := DeleteTagCollection(testDB.DB); err != nil {
		if err == web.ErrNotFound {
			t.Fatal("Tag Not found, nothing to delete")
		}
		t.Error("Unable to delete tag")
	}
}

//nolint:dupl
func TestDelete_nonExistItem(t *testing.T) {
	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	// we will try to delete random gibberish
	if err := Delete(testDB.DB, "emptyId"); err != nil {
		if err == web.ErrNotFound {
			// because we didn't find it, it should succeed
			t.Log("Tag NOT FOUND, this is the expected result")
		} else {
			t.Error("Expected to not be able to delete")
		}
	}
}

func TestUpdate(t *testing.T) {
	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	epc := "30143639F8419105417AED6F"
	facilityID := "TestFacility"
	// insert sample data
	var tagArray = []Tag{
		{
			Epc:        epc,
			FacilityID: facilityID,
		},
	}

	err := Replace(testDB.DB, tagArray)
	if err != nil {
		t.Error("Unable to insert tag", err.Error())
	}

	objectMap := make(map[string]string)
	objectMap["qualified_state"] = "sold"

	err = Update(testDB.DB, epc, facilityID, objectMap)
	if err != nil {
		t.Error("Unable to update the tag", err.Error())
	}

	// verify that update was successful
	tag, err := FindByEpc(testDB.DB, epc)
	if err != nil {
		t.Errorf("Error trying to find tag by epc %s", err.Error())
	} else if !tag.IsTagReadByRspController() {
		if tag.QualifiedState != "sold" {
			t.Fatal("Qualified_state update failed")
		}
	}

	//clean data
	clearAllData(t, testDB.DB)
	err = Update(testDB.DB, epc, facilityID, objectMap)
	if err == nil {
		t.Error("Tag not found error not caught")
	}
}

func TestFindByEpc_found(t *testing.T) {
	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	epc := t.Name()
	insertSampleCustom(t, testDB.DB, epc)

	tag, err := FindByEpc(testDB.DB, epc)
	if err != nil {
		t.Errorf("Error trying to find tag by epc %s", err.Error())
	} else if !tag.IsTagReadByRspController() {
		t.Errorf("Expected to find a tag with epc: %s", epc)
	} else if tag.Epc != epc {
		t.Error("Expected found tag epc to be equal to the input epc")
	}

	selectQuery := fmt.Sprintf(`DELETE FROM %s`,
		pq.QuoteIdentifier(tagsTable),
	)

	_, err = testDB.DB.Exec(selectQuery)
	if err != nil {
		if err == web.ErrNotFound {
			t.Fatal("Tag Not found, nothing to delete")
		}
		t.Error("Unable to delete tag", err)
	}
	clearAllData(t, testDB.DB)
}

func TestFindByEpc_notFound(t *testing.T) {
	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	epc := t.Name()
	tag, err := FindByEpc(testDB.DB, epc)
	if err != nil {
		t.Errorf("Error trying to find tag by epc %s", err.Error())
	} else if tag.IsTagReadByRspController() {
		t.Errorf("Expected to NOT find a tag with epc: %s", epc)
	}
}

func TestCalculateGtin(t *testing.T) {
	config.AppConfig.TagDecoders = []encodingscheme.TagDecoder{encodingscheme.NewSGTINDecoder(true)}
	validEpc := "303402662C3A5F904C19939D"
	gtin, _, err := DecodeTagData(validEpc)
	if gtin == UndefinedProductID {
		t.Errorf("Error trying to calculate valid epc %s: %+v", validEpc, err)
	}
}

func TestCalculateInvalidGtin(t *testing.T) {
	setSGTINOnlyDecoderConfig()
	epcSlice := generateSequentialEpcs("0014", 0, 1)
	gtin, _, err := DecodeTagData(epcSlice[0])
	if gtin != encodingInvalid {
		t.Errorf("Unexpected result calculating invalid epc %s, expected %s, got %s. error val was: %+v",
			epcSlice[0], UndefinedProductID, gtin, err)
	}
}

func setSGTINOnlyDecoderConfig() {
	config.AppConfig.TagDecoders = []encodingscheme.TagDecoder{
		encodingscheme.NewSGTINDecoder(true),
	}
}

func setMixedDecoderConfig(t *testing.T) {
	decoder, err := encodingscheme.NewProprietary(
		"test.com", "2019-01-01", "8.48.40", 2)
	if err != nil {
		t.Fatal(err)
	}
	config.AppConfig.TagDecoders = []encodingscheme.TagDecoder{
		encodingscheme.NewSGTINDecoder(true),
		decoder,
	}
}

func TestCalculateProductCode(t *testing.T) {
	setMixedDecoderConfig(t)
	validEpc := "0F00000000000C00000014D2"
	expectedWrin := "00000014D2"
	productID, _, err := DecodeTagData(validEpc)
	if productID == UndefinedProductID {
		t.Errorf("Error trying to calculate valid epc %s: %+v", validEpc, err)
	}
	if productID != expectedWrin {
		t.Errorf("Error trying to calculate valid epc %s: wanted %s, got %s; err is: %+v",
			validEpc, expectedWrin, productID, err)
	}
}

func insertSample(t *testing.T, db *sql.DB) {
	insertSampleCustom(t, db, t.Name())
}

func insertSampleCustom(t *testing.T, db *sql.DB, sampleID string) {
	var tag Tag

	tag.Epc = sampleID

	if err := insert(db, tag); err != nil {
		t.Error("Unable to insert tag", err)
	}
}

//
//// nolint :dupl
func clearAllData(t *testing.T, db *sql.DB) {
	selectQuery := fmt.Sprintf(`DELETE FROM %s`,
		pq.QuoteIdentifier(tagsTable),
	)

	_, err := db.Exec(selectQuery)
	if err != nil {
		t.Errorf("Unable to delete data from %s table: %s", tagsTable, err)
	}
}

// nolint :dupl
func insert(db *sql.DB, tag Tag) error {

	obj, err := json.Marshal(tag)
	if err != nil {
		return err
	}

	upsertStmt := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s) 
									 ON CONFLICT (( %s  ->> 'epc' )) 
									 DO UPDATE SET %s = %s.%s || %s; `,
		pq.QuoteIdentifier(tagsTable),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteLiteral(string(obj)),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteIdentifier(tagsTable),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteLiteral(string(obj)),
	)

	_, err = db.Exec(upsertStmt)
	if err != nil {
		return errors.Wrap(err, "error in inserting tag")
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
