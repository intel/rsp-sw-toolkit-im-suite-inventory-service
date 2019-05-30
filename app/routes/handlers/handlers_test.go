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
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"plugin"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/encodingscheme"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/integrationtest"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
	db "github.impcloud.net/RSP-Inventory-Suite/go-dbWrapper"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/contraepc"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/facility"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/tag"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/web"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
)

const (
	tagCollection = "tags"
)

type dbFunc func(dbs *db.DB, t *testing.T) error
type validateFunc func(dbs *db.DB, r *httptest.ResponseRecorder, t *testing.T) error

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

var dbHost integrationtest.DBHost

var calculateConfidencePlugin func(float64, float64, float64, float64, int64, bool) float64
var isProbabilisticPluginFound bool

func TestMain(m *testing.M) {
	dbHost = integrationtest.InitHost("handlers_test")

	// Loading Inventory Probabilistic plugin
	confidencePlugin, err := plugin.Open("/plugin/inventory-probabilistic-algo")
	if err != nil {
		isProbabilisticPluginFound = false
		log.Print("Inventory Probabilistic algorithm plugin not found. Confidence value will be 0")
		os.Exit(m.Run())
	}
	// Find CalculateConfidence function
	symbol, err := confidencePlugin.Lookup("CalculateConfidence")
	if err != nil {
		log.Print("Unable to find calculate confidence function. Confidence value will be 0")
		os.Exit(m.Run())
	}

	var ok bool
	calculateConfidencePlugin, ok = symbol.(func(float64, float64, float64, float64, int64, bool) float64)
	if ok {
		isProbabilisticPluginFound = true
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

func insertFacilitiesHelper(t *testing.T, dbs *db.DB) {
	var facilities []facility.Facility
	var testFacility facility.Facility
	testFacility.Name = "Test"

	facilities = append(facilities, testFacility)

	var coefficientes facility.Coefficients
	// Random coefficient values
	coefficientes.DailyInventoryPercentage = 0.1
	coefficientes.ProbExitError = 0.1
	coefficientes.ProbInStoreRead = 0.1
	coefficientes.ProbUnreadToRead = 0.1

	if err := facility.Insert(dbs, &facilities, coefficientes); err != nil {
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

	copySession := masterDB.CopySession()
	insertFacilitiesHelper(t, copySession)
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

	defer copySession.Close()

	inventory := Inventory{masterDB, config.AppConfig.ResponseLimit, testServer.URL + "/skus"}

	handler := web.Handler(inventory.GetTags)
	testHandlerHelper(selectTests, "GET", handler, copySession, t)
}

// nolint :dupl
func TestGetCurrentInventoryPositive(t *testing.T) {

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
		// Expected input with count_only = false
		{
			input: []byte(`{
			"qualified_state":"sold",
			"facility_id":"store001",
			"epc_state":"sold",
			"starttime":1482624000000,
			"endtime":1483228800000,
			"size":2,
			"cursor":"59a754fa22e60174f5109efc",
			"count_only":false
		  }`),
			code: []int{200, 204},
		},
		// Expected input with count_only = true
		{
			input: []byte(`{
				"qualified_state":"sold",
				"facility_id":"store001",
				"epc_state":"sold",
				"starttime":1482624000000,
				"endtime":1483228800000,				
				"count_only":true
			  }`),
			code: []int{200, 204},
		},
	}

	copySession := masterDB.CopySession()
	defer copySession.Close()

	inventory := Inventory{copySession, config.AppConfig.ResponseLimit, testServer.URL + "/skus"}

	handler := web.Handler(inventory.GetCurrentInventory)

	testHandlerHelper(currentInventoryTests, "POST", handler, copySession, t)

}

func TestGetCurrentInventoryNegative(t *testing.T) {

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
		// No facility id
		{
			title: "No facility id test",
			input: []byte(`{
				"qualified_state":"sold",
				"epc_state":"sold"				
			  }`),
			code: []int{400},
		},
		// Empty request body
		{
			title: "Empty request body test",
			input: []byte(``),
			code:  []int{400},
		},
		// Invalid input type for facility_id
		{
			title: "Invalid input type for facility_id test",
			input: []byte(`{ "facility_id":10 }`),
			code:  []int{400},
		},
		// Invalid cursor
		{
			title: "Invalid cursor test",
			input: []byte(`{
					"qualified_state":"sold",
					"facility_id":"store001",
					"epc_state":"sold",
					"starttime":1482624000000,
					"endtime":1483228800000,
					"size":2,
					"cursor":"123",
					"count_only":false
				  }`),
			code: []int{400},
		},
	}

	copySession := masterDB.CopySession()
	defer copySession.Close()

	inventory := Inventory{copySession, config.AppConfig.ResponseLimit, testServer.URL + "/skus"}

	handler := web.Handler(inventory.GetCurrentInventory)

	testHandlerHelper(currentInventoryTests, "POST", handler, copySession, t)

}

// nolint :dupl
func TestGetMissingTagsPositive(t *testing.T) {

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

	var missingTagsTests = []inputTest{
		// Expected input with count_only = false
		{
			input: []byte(`{
				"facility_id":"store001",
				"time":1483228800000,				
				"count_only":false
			  }`),
			code: []int{200, 204},
		},
		// Expected input with count_only = true
		{
			input: []byte(`{
				"facility_id":"store001",
				"time":1483228800000,				
				"count_only":true
			  }`),
			code: []int{200, 204},
		},
	}

	copySession := masterDB.CopySession()
	defer copySession.Close()

	inventory := Inventory{copySession, config.AppConfig.ResponseLimit, testServer.URL + "/skus"}

	handler := web.Handler(inventory.GetMissingTags)

	testHandlerHelper(missingTagsTests, "POST", handler, copySession, t)

}

func TestGetMissingTagsNegative(t *testing.T) {

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

	var missingTagsTests = []inputTest{
		// No facility id
		{
			input: []byte(`{
				"size":500
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
		// No time field
		{
			input: []byte(`{ "facility_id":"store1" }`),
			code:  []int{400},
		},
	}

	copySession := masterDB.CopySession()
	defer copySession.Close()

	inventory := Inventory{copySession, config.AppConfig.ResponseLimit, testServer.URL + "/skus"}

	handler := web.Handler(inventory.GetMissingTags)

	testHandlerHelper(missingTagsTests, "POST", handler, copySession, t)

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

	copySession := masterDB.CopySession()
	defer copySession.Close()

	inventory := Inventory{copySession, config.AppConfig.ResponseLimit, testServer.URL + "/skus"}

	handler := web.Handler(inventory.GetSearchByGtin)

	testHandlerHelper(searchGtinTests, "POST", handler, copySession, t)
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

	copySession := masterDB.CopySession()
	defer copySession.Close()

	inventory := Inventory{copySession, config.AppConfig.ResponseLimit, testServer.URL + "/skus"}

	handler := web.Handler(inventory.GetSearchByGtin)

	testHandlerHelper(searchGtinTests, "POST", handler, copySession, t)

}

func insertTag(t tag.Tag) dbFunc {
	return func(dbs *db.DB, _ *testing.T) error {
		return tag.Replace(dbs, &[]tag.Tag{t})
	}
}

func deleteTag(epc string) dbFunc {
	return func(dbs *db.DB, _ *testing.T) error {
		return tag.Delete(dbs, epc)
	}
}

func deleteAllTags() dbFunc {
	return func(dbs *db.DB, _ *testing.T) error {
		execFunc := func(collection *mgo.Collection) error {
			_, err := collection.RemoveAll(bson.M{})
			return err
		}

		return dbs.Execute(tagCollection, execFunc)
	}
}

func getTagCount(dbs *db.DB) (int, error) {
	var count int
	execFunc := func(collection *mgo.Collection) error {
		n, err := collection.Count()
		count = n
		return err
	}

	err := dbs.Execute(tagCollection, execFunc)
	return count, err
}

func validateAll(fs []validateFunc) validateFunc {
	return func(dbs *db.DB, r *httptest.ResponseRecorder, t *testing.T) error {
		for _, f := range fs {
			if err := f(dbs, r, t); err != nil {
				return err
			}
		}
		return nil
	}
}

func validateContraEpcs() validateFunc {
	return func(_ *db.DB, r *httptest.ResponseRecorder, _ *testing.T) error {
		var js tagResponse
		if err := json.Unmarshal([]byte(r.Body.Bytes()), &js); err != nil {
			return errors.Wrap(err, "Unable to parse results as json!")
		}

		for _, contra := range js.Results {
			if !contraepc.IsContraEpc(contra) {
				return errors.New("result tag is not a contra-epc")
			}

			gtin, err := encodingscheme.GetGtin14(contra.Epc)
			if err != nil {
				return errors.Wrapf(err, "result gtin could not be computed for epc %s", contra.Epc)
			}
			if contra.ProductID != gtin {
				return fmt.Errorf("expected contra-epc gtin to be %s, but was %s", gtin, contra.ProductID)
			}
		}
		return nil
	}
}

//nolint:unparam
func validateSelectEpc(epc string) validateFunc {
	return func(_ *db.DB, r *httptest.ResponseRecorder, _ *testing.T) error {
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
	return func(_ *db.DB, r *httptest.ResponseRecorder, _ *testing.T) error {
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
	return func(_ *db.DB, r *httptest.ResponseRecorder, _ *testing.T) error {
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
						return fmt.Errorf("Unepxected empty string field value")
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
	return func(_ *db.DB, r *httptest.ResponseRecorder, _ *testing.T) error {
		var js tagResponse
		if err := json.Unmarshal([]byte(r.Body.Bytes()), &js); err != nil {
			return errors.Wrap(err, "Unable to parse results as json!")
		}

		if len(js.Results) != size {
			return fmt.Errorf("Invalid result size. Expected: %d, but got: %d", size, len(js.Results))
		}
		return nil
	}
}

func validateTagCount(count int) validateFunc {
	return func(dbs *db.DB, _ *httptest.ResponseRecorder, _ *testing.T) error {
		n, err := getTagCount(dbs)
		if err != nil {
			return err
		}
		if n != count {
			return fmt.Errorf("Invalid tag count -- expected: %d, got: %d", count, n)
		}
		return nil
	}
}

// nolint :dupl
func validateQualifiedStateUpdate(epc string, qualifiedState string) validateFunc {
	return func(dbs *db.DB, _ *httptest.ResponseRecorder, _ *testing.T) error {
		tagInDb, err := tag.FindByEpc(dbs, epc)
		if err != nil {
			return err
		}
		if tagInDb.QualifiedState != qualifiedState {
			return fmt.Errorf("Invalid Qualified State -- expected: %s, got: %s", qualifiedState, tagInDb.QualifiedState)
		}
		return nil
	}
}

// nolint :dupl
func validateEpcContextSet(epc string, epcContext string) validateFunc {
	return func(dbs *db.DB, _ *httptest.ResponseRecorder, _ *testing.T) error {
		tagInDb, err := tag.FindByEpc(dbs, epc)
		if err != nil {
			return err
		}
		if tagInDb.EpcContext != epcContext {
			return fmt.Errorf("Invalid Epc context -- expected: %s, got: %s", epcContext, tagInDb.EpcContext)
		}
		return nil
	}
}

// nolint :dupl
func validateEpcContextDelete(epc string) validateFunc {
	return func(dbs *db.DB, _ *httptest.ResponseRecorder, _ *testing.T) error {
		tagInDb, err := tag.FindByEpc(dbs, epc)
		if err != nil {
			return err
		}
		if tagInDb.EpcContext != "" {
			return fmt.Errorf("Invalid Epc context -- expected: %s, got: %s", "", tagInDb.EpcContext)
		}
		return nil
	}
}

//nolint :gocyclo
func testHandlerHelper(input []inputTest, requestType string, handler web.Handler, dbs *db.DB, t *testing.T) {
	failures := []*httptest.ResponseRecorder{}

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

	copySession := masterDB.CopySession()
	defer copySession.Close()

	inventory := Inventory{copySession, config.AppConfig.ResponseLimit, ""}

	handler := web.Handler(inventory.UpdateQualifiedState)

	testHandlerHelper(qualifiedStateTests, "PUT", handler, copySession, t)

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

	copySession := masterDB.CopySession()
	defer copySession.Close()

	inventory := Inventory{copySession, config.AppConfig.ResponseLimit, testServer.URL + "/skus"}

	handler := web.Handler(inventory.GetSearchByEpc)

	testHandlerHelper(searchEpcTests, "POST", handler, copySession, t)
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

	copySession := masterDB.CopySession()
	defer copySession.Close()

	inventory := Inventory{copySession, config.AppConfig.ResponseLimit, testServer.URL + "/skus"}

	handler := web.Handler(inventory.GetSearchByEpc)

	testHandlerHelper(searchEpcTests, "POST", handler, copySession, t)
}

func TestCreateContraEPC(t *testing.T) {
	masterDB := dbHost.CreateDB(t)
	defer masterDB.Close()

	copySession := masterDB.CopySession()
	defer copySession.Close()

	epc := "30143639F809D845407C9C5C"
	gtin := "00888446100818"
	gtin2 := "00052177002189"
	facility := "test-facility"

	epcRecord := fmt.Sprintf(`{"epc": "%s", "facility_id": "%s"}`, epc, facility)
	gtinRecord := fmt.Sprintf(`{"gtin": "%s", "facility_id": "%s"}`, gtin, facility)
	gtinRecord2 := fmt.Sprintf(`{"gtin": "%s", "facility_id": "%s"}`, gtin2, facility)

	// Create a large sample of 1000 input itmes
	var gtin1000recs [1000]string
	for i := 0; i < 1000; i++ {
		gtin1000recs[i] = gtinRecord
	}
	gtin1000 := strings.Join(gtin1000recs[:], ",")

	inputs := []inputTest{
		{
			title: "Empty request body",
			input: []byte(``),
			code:  []int{400},
		},
		{
			title: "No facility_id",
			input: []byte(fmt.Sprintf(`{"data": [{"epc": "%s"}]}`, epc)),
			code:  []int{400},
		},
		{
			title: "Missing gtin or epc",
			input: []byte(fmt.Sprintf(`{"data": [{"facility_id": "%s"}]}`, facility)),
			code:  []int{400},
		},
		{
			title: "Not allowed gtin AND epc",
			input: []byte(fmt.Sprintf(`{"data": [{"epc": "%s", "gtin": "%s", facility_id": "%s"}]}`,
				epc, gtin, facility)),
			code: []int{400},
		},
		{
			title: "Invalid gtin length",
			input: []byte(fmt.Sprintf(`{"data": [{"gtin": "1234", "facility_id": "%s"}]}`, facility)),
			code:  []int{400},
		},
		{
			title:   "Cannot add epc that already exists in database",
			setup:   insertTag(tag.Tag{Epc: epc}),
			input:   []byte(fmt.Sprintf(`{"data": [%s]}`, epcRecord)),
			code:    []int{400},
			destroy: deleteTag(epc),
		},
		{
			title:   "Create 1 contra-epc record by epc",
			input:   []byte(fmt.Sprintf(`{"data": [{"epc": "%s", "facility_id": "%s"}]}`, epc, facility)),
			code:    []int{200},
			destroy: deleteTag(epc),
		},
		{
			title: "Create 1 contra-epc record by gtin",
			setup: deleteAllTags(),
			input: []byte(fmt.Sprintf(`{"data": [%s]}`, gtinRecord)),
			code:  []int{200},
			validate: validateAll([]validateFunc{
				validateTagCount(1),
				validateResultSize(1),
				validateContraEpcs(),
			}),
			destroy: deleteAllTags(),
		},
		{
			title: "Create 6 contra-epc record with 2 unique gtins",
			setup: deleteAllTags(),
			input: []byte(fmt.Sprintf(`{"data": [%s, %s, %s, %s, %s, %s]}`,
				gtinRecord, gtinRecord, gtinRecord2, gtinRecord2, gtinRecord, gtinRecord2)),
			code: []int{200},
			validate: validateAll([]validateFunc{
				validateTagCount(6),
				validateResultSize(6),
				validateContraEpcs(),
			}),
			destroy: deleteAllTags(),
		},
		{
			title: "Create 1000 contra-epc records with same gtin (stress-test retries and randomness)",
			setup: deleteAllTags(),
			input: []byte(fmt.Sprintf(`{"data": [%s]}`, gtin1000)),
			code:  []int{200},
			validate: validateAll([]validateFunc{
				validateTagCount(1000),
				validateResultSize(1000),
				validateContraEpcs(),
			}),
			destroy: deleteAllTags(),
		},
		{
			title:   "Create 1001 contra-epcs: HTTP 400 Too many request items",
			setup:   deleteAllTags(),
			input:   []byte(fmt.Sprintf(`{"data": [%s, %s]}`, gtin1000, gtinRecord)),
			code:    []int{400},
			destroy: deleteAllTags(),
		},
		{
			title: "Create 3 contra-epc records mixing gtin and epc",
			setup: deleteAllTags(),
			input: []byte(fmt.Sprintf(`{"data": [%s, %s, %s]}`,
				gtinRecord, epcRecord, gtinRecord)),
			code: []int{200},
			validate: validateAll([]validateFunc{
				validateTagCount(3),
				validateResultSize(3),
				validateContraEpcs(),
			}),
			destroy: deleteAllTags(),
		},
		{
			title: "Do not allow 2 contra-epcs with the same epc",
			input: []byte(fmt.Sprintf(`{"data": [%s, %s]}`, epcRecord, epcRecord)),
			code:  []int{400},
		},
	}

	inventory := Inventory{copySession, config.AppConfig.ResponseLimit, ""}
	handler := web.Handler(inventory.CreateContraEPC)
	testHandlerHelper(inputs, "POST", handler, copySession, t)
}

func TestDeleteContraEPC(t *testing.T) {
	masterDB := dbHost.CreateDB(t)
	defer masterDB.Close()

	epc := "100683590000000000001106"
	facility := "test-facility"

	inputs := []inputTest{
		{
			title: "Empty request body",
			input: []byte(``),
			code:  []int{400},
		},
		{
			title: "No facility id",
			input: []byte(fmt.Sprintf(`{"epc": "%s"}`, epc)),
			code:  []int{400},
		},
		{
			title: "Invalid input type for facility_id",
			input: []byte(fmt.Sprintf(`{"epc": "%s", "facility_id": 10}`, epc)),
			code:  []int{400},
		},
		{
			title: "No epc",
			input: []byte(`{"facility_id": "test-facility" }`),
			code:  []int{400},
		},
		{
			title: "Invalid epc length",
			input: []byte(`{"epc": "123"}`),
			code:  []int{400},
		},
		{
			title: "Valid input, but does not exist",
			input: []byte(fmt.Sprintf(`{"epc": "%s", "facility_id": "%s"}`, epc, facility)),
			code:  []int{204},
		},
		{
			title:   "Unable to delete non-contra epc tags",
			setup:   insertTag(tag.Tag{Epc: epc}),
			input:   []byte(fmt.Sprintf(`{"epc": "%s", "facility_id": "%s"}`, epc, facility)),
			code:    []int{400},
			destroy: deleteTag(epc),
		},
		{
			title: "Delete success",
			setup: insertTag(tag.Tag{
				Epc: epc,
				LocationHistory: []tag.LocationHistory{
					{
						Location:  contraepc.Location,
						Timestamp: helper.UnixMilliNow(),
						Source:    contraepc.Source,
					},
				},
			}),
			input:   []byte(fmt.Sprintf(`{"epc": "%s", "facility_id": "%s"}`, epc, facility)),
			code:    []int{204},
			destroy: deleteTag(epc),
		},
	}

	inventory := Inventory{masterDB, config.AppConfig.ResponseLimit, ""}
	handler := web.Handler(inventory.DeleteContraEPC)
	testHandlerHelper(inputs, "POST", handler, masterDB, t)
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

	copySession := masterDB.CopySession()
	defer copySession.Close()

	inventory := Inventory{copySession, config.AppConfig.ResponseLimit, ""}

	handler := web.Handler(inventory.SetEpcContext)

	testHandlerHelper(epcContextTests, "PUT", handler, copySession, t)

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

	copySession := masterDB.CopySession()
	defer copySession.Close()

	inventory := Inventory{copySession, config.AppConfig.ResponseLimit, ""}

	handler := web.Handler(inventory.DeleteEpcContext)

	testHandlerHelper(epcContextTests, "DELETE", handler, copySession, t)

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

	copySession := masterDB.CopySession()
	defer copySession.Close()

	inventory := Inventory{copySession, config.AppConfig.ResponseLimit, ""}

	handler := web.Handler(inventory.DeleteAllTags)

	testHandlerHelper(deleteAllTagTests, "DELETE", handler, copySession, t)
}
