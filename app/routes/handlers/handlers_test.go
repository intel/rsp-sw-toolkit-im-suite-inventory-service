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

package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/facility"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/tag"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/integrationtest"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/web"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

const (
	tagsTable = "tags"
)

type dbFunc func(dbs *sql.DB, t *testing.T) error
type validateFunc func(dbs *sql.DB, r *httptest.ResponseRecorder, t *testing.T) error
var dbHost integrationtest.DBHost

// tagResponse holds a more specific version of the generic tag.Response
// struct to avoid multi-level type assertion
type tagResponse struct {
	PagingType tag.PagingType `json:"paging"`
	Results    []tag.Tag      `json:"results"`
}

type inputTest struct {
	title    string
	setup    dbFunc
	input    []byte
	code     []int
	validate validateFunc
	destroy  dbFunc
	queryStr string
}

var isProbabilisticPluginFound bool

func TestMain(m *testing.M) {
	dbHost = integrationtest.InitHost("handlers_test")

	if err := loadConfidencePlugin(); err != nil {
		isProbabilisticPluginFound = false
		log.Printf("these tests lack confidence: %+v\n", err)
		os.Exit(m.Run())
	}

	os.Exit(m.Run())
}

func TestGetIndex(t *testing.T) {
	request, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Errorf("Unable to create new HTTP request %s", err.Error())
	}
	recorder := httptest.NewRecorder()
	inventory := Inventory{nil, 0, ""}
	handler := web.Handler(inventory.Index)
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected 200 response")
	}
	log.Print(recorder.Body.String())
	if recorder.Body.String() != "\"Inventory Service\"" {
		t.Errorf("Expected body to equal Inventory Service")
	}
}

// nolint :dupl
func TestGetTags(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(1 * time.Second)
		if request.URL.EscapedPath() != "/skus" {
			t.Errorf("Expected request to be '/skus', received %s",
				request.URL.EscapedPath())
		}
		if request.Method != "GET" {
			t.Errorf("Expected 'GET' request, received '%s", request.Method)
		}
		var jsonData []byte
		if request.URL.EscapedPath() == "/skus" {
			result := buildProductData(0.2, 0.75, 0.2, 0.1, "00111111")
			jsonData, _ = json.Marshal(result)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(jsonData)
	}))

	defer testServer.Close()

	masterDB := dbHost.CreateDB(t)
	defer masterDB.Close()

	request, err := http.NewRequest("GET", "/tags", nil)
	if err != nil {
		t.Errorf("Unable to create new HTTP request %s", err.Error())
	}

	recorder := httptest.NewRecorder()

	inventory := Inventory{masterDB, config.AppConfig.ResponseLimit, testServer.URL + "/skus"}

	handler := web.Handler(inventory.GetTags)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK &&
		recorder.Code != http.StatusNoContent {
		t.Errorf("Success expected: %d", recorder.Code)
	}

}

func TestGetOdata(t *testing.T) {

	var testCases = []inputTest{
		{
			queryStr: "/inventory/tags?$filter=startswith(epc,'3')&$count",
			code:     []int{200},
		},
		{
			queryStr: "/inventory/tags?$filter=startswith(epc,'3')&$inlinecount=allpages&$top=1",
			code:     []int{200},
		},
		{
			queryStr: "/inventory/tags?$filter=startswith(epc,'3')&$count&$inlinecount=allpages",
			code:     []int{400},
		},
	}

	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(1 * time.Second)
		if request.URL.EscapedPath() != "/skus" {
			t.Errorf("Expected request to be '/skus', received %s",
				request.URL.EscapedPath())
		}
		if request.Method != "GET" {
			t.Errorf("Expected 'GET' request, received '%s", request.Method)
		}
		var jsonData []byte
		if request.URL.EscapedPath() == "/skus" {
			result := buildProductData(0.2, 0.75, 0.2, 0.1, "00111111")
			jsonData, _ = json.Marshal(result)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(jsonData)
	}))

	defer testServer.Close()

	masterDB := dbHost.CreateDB(t)
	defer masterDB.Close()

	for _, item := range testCases {

		request, err := http.NewRequest("GET", item.queryStr, nil)
		if err != nil {
			t.Errorf("Unable to create new HTTP request %s", err.Error())
		}

		recorder := httptest.NewRecorder()

		inventory := Inventory{masterDB, config.AppConfig.ResponseLimit, testServer.URL + "/skus"}

		handler := web.Handler(inventory.GetTags)

		handler.ServeHTTP(recorder, request)

		if recorder.Code != item.code[0] {
			t.Errorf("Error, expected: %d Actual: %d", item.code[0], recorder.Code)
		}
	}
}

func insertFacilitiesHelper(t *testing.T, dbs *sql.DB) {
	var facilities []facility.Facility
	var testFacility facility.Facility
	testFacility.Name = "Test"

	facilities = append(facilities, testFacility)

	var coefficients facility.Coefficients
	// Random coefficient values
	coefficients.DailyInventoryPercentage = 0.1
	coefficients.ProbExitError = 0.1
	coefficients.ProbInStoreRead = 0.1
	coefficients.ProbUnreadToRead = 0.1

	if err := facility.Insert(dbs, &facilities, coefficients); err != nil {
		t.Errorf("error inserting facilities %s", err.Error())
	}
}

func TestGetSelectTags(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.EscapedPath() != "/skus" {
			t.Errorf("Expected request to be '/skus', received %s",
				request.URL.EscapedPath())
		}
		if request.Method != "GET" {
			t.Errorf("Expected 'GET' request, received '%s", request.Method)
		}
		var jsonData []byte
		if request.URL.EscapedPath() == "/skus" {
			result := buildProductData(0.2, 0.75, 0.2, 0.1, "00111111")
			jsonData, _ = json.Marshal(result)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(jsonData)
	}))

	defer testServer.Close()

	masterDB := dbHost.CreateDB(t)
	defer masterDB.Close()

	insertFacilitiesHelper(t, masterDB)
	epc := "100683590000000000001106"
	facilityID := "Test"
	lastRead := int64(1506638821662)
	locationHistory := []tag.LocationHistory{
		{
			Location:  "RSP-950b44",
			Timestamp: 1506638821662,
			Source:    "fixed",
		}}

	var selectTests = []inputTest{
		{
			title: "Select Tag with epc",
			setup: insertTag(tag.Tag{
				Epc:        epc,
				FacilityID: facilityID,
			}),
			code: []int{200},
			validate: validateAll([]validateFunc{
				validateSelectEpc(epc),
			}),
			queryStr: "?$select=epc",
			destroy:  deleteTag(epc),
		},
		{
			title: "Select tag with confidence",
			setup: insertTag(tag.Tag{
				Epc:             epc,
				FacilityID:      facilityID,
				LastRead:        lastRead,
				LocationHistory: locationHistory,
			}),
			code: []int{200},
			validate: validateAll([]validateFunc{
				validateSelectConfidence(),
			}),
			queryStr: "?$select=epc,confidence",
			destroy:  deleteTag(epc),
		},
		{
			title: "Count Return values",
			setup: insertTag(tag.Tag{
				Epc:             epc,
				FacilityID:      facilityID,
				LastRead:        lastRead,
				LocationHistory: locationHistory,
			}),
			code: []int{200},
			validate: validateAll([]validateFunc{
				validateSelectFields([]string{"epc", "confidence"}),
			}),
			queryStr: "?$select=epc,confidence",
			destroy:  deleteTag(epc),
		},
		{
			title: "Select Fields With Spaces",
			setup: insertTag(tag.Tag{
				Epc:             epc,
				FacilityID:      facilityID,
				LastRead:        lastRead,
				LocationHistory: locationHistory,
			}),
			code: []int{200},
			validate: validateAll([]validateFunc{
				validateSelectFields([]string{"epc", "confidence"}),
			}),
			queryStr: "?$select= epc, confidence",
			destroy:  deleteTag(epc),
		},
	}

	inventory := Inventory{masterDB, config.AppConfig.ResponseLimit, testServer.URL + "/skus"}

	handler := web.Handler(inventory.GetTags)
	testHandlerHelper(selectTests, "GET", handler, masterDB, t)
}

// nolint :dupl
func TestPostCurrentInventory(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(1 * time.Second)
		if request.URL.EscapedPath() != "/skus" {
			t.Errorf("Expected request to be '/skus', received %s",
				request.URL.EscapedPath())
		}
		if request.Method != "GET" {
			t.Errorf("Expected 'GET' request, received '%s", request.Method)
		}
		var jsonData []byte
		if request.URL.EscapedPath() == "/skus" {
			result := buildProductData(0.2, 0.75, 0.2, 0.1, "00111111")
			jsonData, _ = json.Marshal(result)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(jsonData)
	}))

	defer testServer.Close()
	masterDB := dbHost.CreateDB(t)
	defer masterDB.Close()

	var currentInventoryTests = []inputTest{
		{
			input: []byte(`{
			"qualified_state":"sold",
			"facility_id":"store001",
			"epc_state":"sold"
		  }`),
			code: []int{200},
		},
		{
			input: []byte(`{
			"facility_id":"store001"
		  }`),
			code: []int{200},
		},
		// Invalid input type for facility_id
		{
			input: []byte(`{ "facility_id":10 }`),
			code:  []int{400},
		},
		// Additional properties not allowed
		{
			input: []byte(`{ "test":10 }`),
			code:  []int{400},
		},
	}

	inventory := Inventory{masterDB, config.AppConfig.ResponseLimit, testServer.URL + "/skus"}

	handler := web.Handler(inventory.PostCurrentInventory)

	testHandlerHelper(currentInventoryTests, "POST", handler, masterDB, t)
}

//nolint :gocyclo
func testHandlerHelper(input []inputTest, requestType string, handler web.Handler, dbs *sql.DB, t *testing.T) {
	var failures []*httptest.ResponseRecorder

	for i, item := range input {
		if item.title == "" {
			item.title = fmt.Sprintf("Test Input %d", i)
		}

		if item.setup != nil {
			if err := item.setup(dbs, t); err != nil {
				t.Errorf("Unable to setup test function: %s", err.Error())
			}
		}
		var request *http.Request
		var err error
		if requestType != "GET" {
			request, err = http.NewRequest(requestType, "", bytes.NewBuffer(item.input))
		} else {
			request, err = http.NewRequest(requestType, item.queryStr, nil)
		}
		if err != nil {
			t.Errorf("Unable to create new HTTP request %s", err.Error())
		}

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		var validateErr error
		success := false
		for _, statusCode := range item.code {
			if statusCode == recorder.Code {
				if item.validate != nil {
					if err := item.validate(dbs, recorder, t); err != nil {
						validateErr = err
						success = false
						break
					}
				}
				success = true
				break
			}
		}

		if success {
			t.Logf("\r [PASS] %s :: %s", t.Name(), item.title)
		} else {
			// Mark as failed, but do not exit so the other tests can run
			t.Fail()
			failures = append(failures, recorder)
			t.Logf("\r [FAIL] %s :: %s ", t.Name(), item.title)
			if validateErr != nil {
				t.Logf("\r\t\tValidation error: %s, response body: %s",
					validateErr.Error(), recorder.Body.String())
			} else {
				t.Logf("\r\t\tStatus code didn't match, status code received: %d, response body: %s",
					recorder.Code, recorder.Body.String())
			}
		}

		if item.destroy != nil {
			item.destroy(dbs, t)
		}
	}

	s := "PASS"
	if t.Failed() {
		s = "FAIL"
	}
	t.Log("\r---------------------------------------------------------")
	t.Logf("\r| Results: %-45s|",
		fmt.Sprintf("%s (%d tests, %d passed, %d failed)",
			s, len(input), len(input)-len(failures), len(failures)))
	t.Log("\r---------------------------------------------------------")
}

// nolint :dupl
func TestSearchByGtinPositive(t *testing.T) {

	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(1 * time.Second)
		if request.URL.EscapedPath() != "/skus" {
			t.Errorf("Expected request to be '/skus', received %s",
				request.URL.EscapedPath())
		}
		if request.Method != "GET" {
			t.Errorf("Expected 'GET' request, received '%s", request.Method)
		}
		var jsonData []byte
		if request.URL.EscapedPath() == "/skus" {
			result := buildProductData(0.2, 0.75, 0.2, 0.1, "00111111")
			jsonData, _ = json.Marshal(result)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(jsonData)
	}))

	defer testServer.Close()

	masterDB := dbHost.CreateDB(t)
	defer masterDB.Close()

	var searchGtinTests = []inputTest{
		// Expected input with count_only = false
		{
			input: []byte(`{
					"facility_id":"store001",
					"gtin":"00012345678905",				
					"count_only":false,
					"size":500
				  }`),
			code: []int{200, 204},
		},
		// Expected input with count_only = true
		{
			input: []byte(`{
				"facility_id":"store001",
				"gtin":"00012345678905",				
				"count_only":true,
				"size":500
			  }`),
			code: []int{200, 204},
		},
	}

	inventory := Inventory{masterDB, config.AppConfig.ResponseLimit, testServer.URL + "/skus"}

	handler := web.Handler(inventory.GetSearchByGtin)

	testHandlerHelper(searchGtinTests, "POST", handler, masterDB, t)
}

// nolint :dupl
func TestSearchByGtinNegative(t *testing.T) {

	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(1 * time.Second)
		if request.URL.EscapedPath() != "/skus" {
			t.Errorf("Expected request to be '/skus', received %s",
				request.URL.EscapedPath())
		}
		if request.Method != "GET" {
			t.Errorf("Expected 'GET' request, received '%s", request.Method)
		}
		var jsonData []byte
		if request.URL.EscapedPath() == "/skus" {
			result := buildProductData(0.2, 0.75, 0.2, 0.1, "00111111")
			jsonData, _ = json.Marshal(result)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(jsonData)
	}))

	defer testServer.Close()

	masterDB := dbHost.CreateDB(t)
	defer masterDB.Close()

	var searchGtinTests = []inputTest{
		// No facility id
		{
			input: []byte(`{
				"size":500,
				"gtin":"00012345678905"
			  }`),
			code: []int{400},
		},
		// Empty request body
		{
			input: []byte(``),
			code:  []int{400},
		},
		// Non-json body
		{
			input: []byte(`blah`),
			code:  []int{400},
		},
		// Invalid input type for facility_id
		{
			input: []byte(`{ "facility_id":10 }`),
			code:  []int{400},
		},
		// No gtin field
		{
			input: []byte(`{ "facility_id":"store1" }`),
			code:  []int{400},
		},
		// Invalid gtin size
		{
			input: []byte(`{
				"size":500,
				"gtin":"123",
				"facility_id":"store1"
			  }`),
			code: []int{400},
		},
	}

	inventory := Inventory{masterDB, config.AppConfig.ResponseLimit, testServer.URL + "/skus"}

	handler := web.Handler(inventory.GetSearchByGtin)

	testHandlerHelper(searchGtinTests, "POST", handler, masterDB, t)

}

func insertTag(t tag.Tag) dbFunc {
	return func(dbs *sql.DB, _ *testing.T) error {
		return tag.Replace(dbs, []tag.Tag{t})
	}
}

func deleteTag(epc string) dbFunc {
	return func(dbs *sql.DB, _ *testing.T) error {
		return tag.Delete(dbs, epc)
	}
}

func deleteAllTags() dbFunc {

	return func(dbs *sql.DB, _ *testing.T) error {
		selectQuery := fmt.Sprintf(`DELETE FROM %s`,
			pq.QuoteIdentifier(tagsTable),
		)
		_, err := dbs.Exec(selectQuery)
		return err
	}
}

func getTagCount(dbs *sql.DB) (int, error) {
	var count int

	row := dbs.QueryRow("SELECT count(*) FROM " + tagsTable)
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func validateAll(fs []validateFunc) validateFunc {
	return func(dbs *sql.DB, r *httptest.ResponseRecorder, t *testing.T) error {
		for _, f := range fs {
			if err := f(dbs, r, t); err != nil {
				return err
			}
		}
		return nil
	}
}

//nolint:unparam
func validateSelectEpc(epc string) validateFunc {
	return func(_ *sql.DB, r *httptest.ResponseRecorder, _ *testing.T) error {
		var js tagResponse
		if err := json.Unmarshal([]byte(r.Body.Bytes()), &js); err != nil {
			return errors.Wrap(err, "Unable to parse results as json!")
		}

		var found bool
		for _, tag := range js.Results {
			if tag.Epc == epc {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("expected epc to be %s, but not found", epc)
		}
		return nil
	}
}

func validateSelectConfidence() validateFunc {
	return func(_ *sql.DB, r *httptest.ResponseRecorder, _ *testing.T) error {
		var js tagResponse
		if err := json.Unmarshal([]byte(r.Body.Bytes()), &js); err != nil {
			return errors.Wrap(err, "Unable to parse results as json!")
		}
		for _, tag := range js.Results {
			if tag.Confidence < 0.0 && tag.Confidence > 1.0 {
				return fmt.Errorf("confidence is invalid. Must be 0-1. Got %v", tag.Confidence)
			}
		}
		return nil
	}
}
func validateSelectFields(fields []string) validateFunc {
	return func(_ *sql.DB, r *httptest.ResponseRecorder, _ *testing.T) error {
		var js tagResponse
		if err := json.Unmarshal([]byte(r.Body.Bytes()), &js); err != nil {
			return errors.Wrap(err, "Unable to parse results as json!")
		}
		mapSelectFields := make(map[string]bool, len(fields))
		for _, field := range fields {
			mapSelectFields[strings.TrimSpace(field)] = true
		}

		for _, tag := range js.Results {
			tag := MapOfTags(&tag, mapSelectFields)
			for _, fieldValue := range tag {
				switch fieldValue.(type) {
				case string:
					if fieldValue.(string) == "" {
						return fmt.Errorf("unepxected empty string field value")
					}
				case float64:
					if fieldValue.(float64) < 0 {
						return fmt.Errorf(" field value")
					}
				}
			}
		}

		return nil
	}
}

func validateResultSize(size int) validateFunc {
	return func(_ *sql.DB, r *httptest.ResponseRecorder, _ *testing.T) error {
		var js tagResponse
		if err := json.Unmarshal([]byte(r.Body.Bytes()), &js); err != nil {
			return errors.Wrap(err, "Unable to parse results as json!")
		}

		if len(js.Results) != size {
			return fmt.Errorf("invalid result size. Expected: %d, but got: %d", size, len(js.Results))
		}
		return nil
	}
}

func validateTagCount(count int) validateFunc {
	return func(dbs *sql.DB, _ *httptest.ResponseRecorder, _ *testing.T) error {
		n, err := getTagCount(dbs)
		if err != nil {
			return err
		}
		if n != count {
			return fmt.Errorf("invalid tag count -- expected: %d, got: %d", count, n)
		}
		return nil
	}
}

// nolint :dupl
func validateQualifiedStateUpdate(epc string, qualifiedState string) validateFunc {
	return func(dbs *sql.DB, _ *httptest.ResponseRecorder, _ *testing.T) error {
		tagInDb, err := tag.FindByEpc(dbs, epc)
		if err != nil {
			return err
		}
		if tagInDb.QualifiedState != qualifiedState {
			return fmt.Errorf("invalid Qualified State -- expected: %s, got: %s", qualifiedState, tagInDb.QualifiedState)
		}
		return nil
	}
}

// nolint :dupl
func validateEpcContextSet(epc string, epcContext string) validateFunc {
	return func(dbs *sql.DB, _ *httptest.ResponseRecorder, _ *testing.T) error {
		tagInDb, err := tag.FindByEpc(dbs, epc)
		if err != nil {
			return err
		}
		if tagInDb.EpcContext != epcContext {
			return fmt.Errorf("invalid Epc context -- expected: %s, got: %s", epcContext, tagInDb.EpcContext)
		}
		return nil
	}
}

// nolint :dupl
func validateEpcContextDelete(epc string) validateFunc {
	return func(dbs *sql.DB, _ *httptest.ResponseRecorder, _ *testing.T) error {
		tagInDb, err := tag.FindByEpc(dbs, epc)
		if err != nil {
			return err
		}
		if tagInDb.EpcContext != "" {
			return fmt.Errorf("invalid Epc context -- expected: %s, got: %s", "", tagInDb.EpcContext)
		}
		return nil
	}
}

func TestUpdateQualifiedState(t *testing.T) {
	masterDB := dbHost.CreateDB(t)
	defer masterDB.Close()

	epc := "100683590000000000001106"
	facility := "test-facility"
	qualifiedState := "sold"

	// nolint :dupl
	var qualifiedStateTests = []inputTest{
		{
			title: "Set success",
			setup: insertTag(tag.Tag{
				Epc:            epc,
				FacilityID:     facility,
				QualifiedState: "unknown",
			}),
			input: []byte(fmt.Sprintf(`{"epc": "%s", "facility_id": "%s", "qualified_state": "%s"}`,
				epc, facility, qualifiedState)),
			code: []int{200},
			validate: validateAll([]validateFunc{
				validateQualifiedStateUpdate(epc, qualifiedState),
			}),
			destroy: deleteTag(epc),
		},
		{
			title: "Tag doesn't exist",
			input: []byte(fmt.Sprintf(`{"epc": "%s", "facility_id": "%s", "qualified_state": "%s"}`,
				epc, facility, "hello")),
			code: []int{404},
		},
		{
			title: "No facility_id",
			input: []byte(fmt.Sprintf(`{"data": [{"epc": "%s"}]}`, epc)),
			code:  []int{400},
		},
		{
			title: "Bad request, wrong input type",
			input: []byte(fmt.Sprintf(`{"epc": "%s", "facility_id": "%s", "qualified_state": %d}`,
				epc, facility, 123)),
			code: []int{400},
		},
		{
			title: "Empty request body",
			input: []byte(``),
			code:  []int{400},
		},
	}

	inventory := Inventory{masterDB, config.AppConfig.ResponseLimit, ""}

	handler := web.Handler(inventory.UpdateQualifiedState)

	testHandlerHelper(qualifiedStateTests, "PUT", handler, masterDB, t)

}

// nolint :dupl
func TestGetFacilities(t *testing.T) {
	masterDB := dbHost.CreateDB(t)
	defer masterDB.Close()

	request, err := http.NewRequest("GET", "/facilities", nil)
	if err != nil {
		t.Errorf("Unable to create new HTTP request %s", err.Error())
	}

	recorder := httptest.NewRecorder()

	inventory := Inventory{masterDB, config.AppConfig.ResponseLimit, ""}

	handler := web.Handler(inventory.GetFacilities)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK &&
		recorder.Code != http.StatusNoContent {
		t.Errorf("Success expected: %d", recorder.Code)
	}

}

func TestGetHandheldEvents(t *testing.T) {
	masterDB := dbHost.CreateDB(t)
	defer masterDB.Close()

	request, err := http.NewRequest("GET", "/handheldevents", nil)
	if err != nil {
		t.Errorf("Unable to create new HTTP request %s", err.Error())
	}

	recorder := httptest.NewRecorder()

	inventory := Inventory{masterDB, config.AppConfig.ResponseLimit, ""}

	handler := web.Handler(inventory.GetHandheldEvents)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK &&
		recorder.Code != http.StatusNoContent {
		t.Errorf("Success expected: %d", recorder.Code)
	}
}

func TestMapRequestToOdata(t *testing.T) {
	var requestBody = tag.RequestBody{
		QualifiedState: "sold",
		FacilityID:     "store001",
		EpcState:       "sold",
		ProductID:      "12345678978945",
		Time:           1483228800000,
		StartTime:      1482624000000,
		EndTime:        1483228800000,
		Confidence:     0.75,
		CountOnly:      false,
	}

	requestBody.CountOnly = false

	actualMap := make(map[string][]string)
	filterArray := []string{"facility_id eq 'store001' and qualified_state eq 'sold' and epc_state eq 'sold'" +
		" and confidence ge 0.75 and gtin eq '12345678978945' and last_read ge 1482624000000" +
		" and last_read le 1483228800000 and last_read le 1483228800000"}
	expectedMap := map[string][]string{
		"$filter": filterArray,
	}
	actualMap = mapRequestToOdata(actualMap, &requestBody)
	if !reflect.DeepEqual(actualMap, expectedMap) {
		t.Errorf("Actual map is %v but got %v", actualMap, expectedMap)
	}

	requestBody.CountOnly = true

	expectedMap["$inlinecount"] = []string{"allpages"}
	actualMap = make(map[string][]string)
	actualMap = mapRequestToOdata(actualMap, &requestBody)
	if !reflect.DeepEqual(actualMap, expectedMap) {
		t.Errorf("Actual map is %v but got %v", actualMap, expectedMap)
	}
}

func TestMapEpcRequestToOdata(t *testing.T) {
	var requestBody = tag.RequestBody{
		FacilityID: "store001",
		Epc:        "0123456",
	}
	compareMaps(t, []string{"facility_id eq 'store001' and epc eq '0123456'"}, &requestBody)

	requestBody.Epc = "*0123456"
	compareMaps(t, []string{"facility_id eq 'store001' and endswith(epc, '0123456')"}, &requestBody)

	requestBody.Epc = "0123456*"
	compareMaps(t, []string{"facility_id eq 'store001' and startswith(epc, '0123456')"}, &requestBody)

	requestBody.Epc = "0123*456"
	compareMaps(t, []string{"facility_id eq 'store001' and startswith(epc, '0123') and endswith(epc, '456')"},
		&requestBody)
}

func compareMaps(t *testing.T, filterStrings []string, requestBody *tag.RequestBody) {
	expectedMap := map[string][]string{
		"$filter": filterStrings,
	}
	actualMap := make(map[string][]string)
	actualMap = mapRequestToOdata(actualMap, requestBody)
	if !reflect.DeepEqual(actualMap, expectedMap) {
		t.Errorf("Actual map is %v but got %v", actualMap, expectedMap)
	}
}

func TestSearchByEpcPositive(t *testing.T) {

	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(1 * time.Second)
		if request.URL.EscapedPath() != "/skus" {
			t.Errorf("Expected request to be '/skus', received %s",
				request.URL.EscapedPath())
		}
		if request.Method != "GET" {
			t.Errorf("Expected 'GET' request, received '%s", request.Method)
		}
		var jsonData []byte
		if request.URL.EscapedPath() == "/skus" {
			result := buildProductData(0.2, 0.75, 0.2, 0.1, "00111111")
			jsonData, _ = json.Marshal(result)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(jsonData)
	}))

	defer testServer.Close()

	masterDB := dbHost.CreateDB(t)
	defer masterDB.Close()

	var searchEpcTests = []inputTest{
		{
			input: []byte(`{
						"facility_id":"store001",
						"epc":"00012345678905",
						"size":500
						}`),
			code: []int{200, 204},
		},
		{
			input: []byte(`{
						"facility_id":"store001",
						"epc":"000123*45678905",
						"size":500
						}`),
			code: []int{200, 204},
		},
		{
			input: []byte(`{
						"facility_id":"store001",
						"epc":"*00012345678905",
						"size":500
						}`),
			code: []int{200, 204},
		},
		{
			input: []byte(`{
						"facility_id":"store001",
						"epc":"00012345678905*",
						"size":500
						}`),
			code: []int{200, 204},
		},
	}

	inventory := Inventory{masterDB, config.AppConfig.ResponseLimit, testServer.URL + "/skus"}

	handler := web.Handler(inventory.GetSearchByEpc)

	testHandlerHelper(searchEpcTests, "POST", handler, masterDB, t)
}

// nolint :dupl
func TestSearchByEpcNegative(t *testing.T) {

	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(1 * time.Second)
		if request.URL.EscapedPath() != "/skus" {
			t.Errorf("Expected request to be '/skus', received %s",
				request.URL.EscapedPath())
		}
		if request.Method != "GET" {
			t.Errorf("Expected 'GET' request, received '%s", request.Method)
		}
		var jsonData []byte
		if request.URL.EscapedPath() == "/skus" {
			result := buildProductData(0.2, 0.75, 0.2, 0.1, "00111111")
			jsonData, _ = json.Marshal(result)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(jsonData)
	}))

	defer testServer.Close()

	masterDB := dbHost.CreateDB(t)
	defer masterDB.Close()

	var searchEpcTests = []inputTest{
		// No facility id
		{
			input: []byte(`{
					"size":500,
					"epc":"0001*"
					}`),
			code: []int{400},
		},
		// Empty request body
		{
			input: []byte(``),
			code:  []int{400},
		},
		// Invalid input type for facility_id
		{
			input: []byte(`{ "facility_id":10 }`),
			code:  []int{400},
		},
		// No EPC field
		{
			input: []byte(`{ "facility_id":"store1" }`),
			code:  []int{400},
		},
		// Invalid characters in EPC field
		{
			input: []byte(`{
					"size":500,
					"epc":"0GHI-$",
					"facility_id":"store1"
					}`),
			code: []int{400},
		},
		// More than one '*'
		{
			input: []byte(`{
					"size":500,
					"epc":"*01234*",
					"facility_id":"store1"
					}`),
			code: []int{400},
		},
	}

	inventory := Inventory{masterDB, config.AppConfig.ResponseLimit, testServer.URL + "/skus"}

	handler := web.Handler(inventory.GetSearchByEpc)

	testHandlerHelper(searchEpcTests, "POST", handler, masterDB, t)
}

func TestUpdateCoefficientsPositive(t *testing.T) {

	var searchGtinTests = []inputTest{
		{
			input: []byte(`{
					"facility_id":"Tavern",
					"dailyinventorypercentage":0.30,
					"probunreadtoread":0.30,
					"probinstoreread":0.30,
					"probexiterror": 0.30
				  }`),
			code: []int{200, 404},
		},
	}

	masterDB := dbHost.CreateDB(t)
	defer masterDB.Close()

	inventory := Inventory{masterDB, config.AppConfig.ResponseLimit, ""}
	handler := web.Handler(inventory.UpdateCoefficients)

	testHandlerHelper(searchGtinTests, "PUT", handler, masterDB, t)

}

func TestUpdateCoefficientsNegative(t *testing.T) {

	var searchGtinTests = []inputTest{
		// No facility_id
		{
			input: []byte(`{
						"dailyinventorypercentage":0.30,
						"probunreadtoread":0.30,
						"probinstoreread":0.30,
						"probexiterror": 0.30
					  }`),
			code: []int{400},
		},
		{
			// No number types
			input: []byte(`{
						"facility_id":"Tavern",
						"dailyinventorypercentage":"0.30",
						"probunreadtoread":"0.30",
						"probinstoreread":"0.30",
						"probexiterror": "0.30"
					  }`),
			code: []int{400},
		},
		{
			// No coefficients
			input: []byte(`{
						"facility_id":"Tavern"
					  }`),
			code: []int{400},
		},
	}

	masterDB := dbHost.CreateDB(t)
	defer masterDB.Close()

	inventory := Inventory{masterDB, config.AppConfig.ResponseLimit, ""}
	handler := web.Handler(inventory.UpdateCoefficients)

	testHandlerHelper(searchGtinTests, "PUT", handler, masterDB, t)

}
func TestSetEpcContext(t *testing.T) {
	masterDB := dbHost.CreateDB(t)
	defer masterDB.Close()

	epc := "100683590000000000001106"
	facility := "test-facility"
	epcContext := "hello-world"

	// nolint :dupl
	var epcContextTests = []inputTest{
		{
			title: "Set success",
			setup: insertTag(tag.Tag{
				Epc:        epc,
				FacilityID: facility,
				EpcContext: "old-text",
			}),
			input: []byte(fmt.Sprintf(`{"epc": "%s", "facility_id": "%s", "epc_context": "%s"}`,
				epc, facility, epcContext)),
			code: []int{200},
			validate: validateAll([]validateFunc{
				validateEpcContextSet(epc, epcContext),
			}),
			destroy: deleteTag(epc),
		},
		{
			title: "Tag doesn't exist",
			input: []byte(fmt.Sprintf(`{"epc": "%s", "facility_id": "%s", "epc_context": "%s"}`,
				epc, facility, "hello")),
			code: []int{404},
		},
		{
			title: "No facility_id",
			input: []byte(fmt.Sprintf(`{"data": [{"epc": "%s"}]}`, epc)),
			code:  []int{400},
		},
	}

	inventory := Inventory{masterDB, config.AppConfig.ResponseLimit, ""}

	handler := web.Handler(inventory.SetEpcContext)

	testHandlerHelper(epcContextTests, "PUT", handler, masterDB, t)

}

func TestDeleteEpcContext(t *testing.T) {
	masterDB := dbHost.CreateDB(t)
	defer masterDB.Close()

	epc := "100683590000000000001106"
	facility := "test-facility"
	epcContext := "test context"

	var epcContextTests = []inputTest{
		{
			title: "Delete success",
			setup: insertTag(tag.Tag{
				Epc:        epc,
				FacilityID: facility,
				EpcContext: epcContext,
			}),
			input: []byte(fmt.Sprintf(`{"epc": "%s", "facility_id": "%s"}`,
				epc, facility)),
			code: []int{200},
			validate: validateAll([]validateFunc{
				validateEpcContextDelete(epc),
			}),
			destroy: deleteTag(epc),
		},
	}

	inventory := Inventory{masterDB, config.AppConfig.ResponseLimit, ""}

	handler := web.Handler(inventory.DeleteEpcContext)

	testHandlerHelper(epcContextTests, "DELETE", handler, masterDB, t)

}

func TestDeleteAllTags(t *testing.T) {
	masterDB := dbHost.CreateDB(t)
	defer masterDB.Close()

	epc := "100683590000000000001106"
	facility := "test-facility"
	epcContext := "test context"

	var deleteAllTagTests = []inputTest{
		{
			title: "Delete All Tags success",
			setup: insertTag(tag.Tag{
				Epc:        epc,
				FacilityID: facility,
				EpcContext: epcContext,
			}),
			input: []byte(fmt.Sprintf(`{"epc": "%s", "facility_id": "%s"}`,
				epc, facility)),
			code: []int{204},
			validate: validateAll([]validateFunc{
				validateTagCount(0),
			}),
			destroy: deleteAllTags(),
		},
	}

	inventory := Inventory{masterDB, config.AppConfig.ResponseLimit, ""}

	handler := web.Handler(inventory.DeleteAllTags)

	testHandlerHelper(deleteAllTagTests, "DELETE", handler, masterDB, t)
}
