/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/expect"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/cloudconnector/event"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/heartbeat"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/tag"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/integrationtest"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/statemodel"
	"os"

	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/jsonrpc"
	"github.impcloud.net/RSP-Inventory-Suite/tagcode/epc"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const tagsTable = "tags"
const jsonb = "data"
const epcColumn = "epc"

/* If useful this will be used for POSTGRESQL too in future */
var dbHost integrationtest.DBHost

func TestMain(m *testing.M) {
	dbHost = integrationtest.InitHost("main_test")
	exitCode := m.Run()
	dbHost.Close()
	os.Exit(exitCode)
}

// POC only implementation
func TestMarkDepartedIfUnseen(t *testing.T) {
	timestamp := 1500583263000 // Thursday, July 20, 2017 1:41:03 PM
	minutesBeforeAgeOut := 10

	tagEvent := jsonrpc.TagEvent{
		EventType:  "cycle_count",
		FacilityID: "testFacility",
		Timestamp:  int64(timestamp),
	}

	ageOuts := map[string]int{
		"testFacility": minutesBeforeAgeOut,
	}

	// calculate the future time after which we should consider this departed
	expectedFuture := int64(timestamp + (minutesBeforeAgeOut * 60 * 1000))

	markDepartedIfUnseen(&tagEvent, ageOuts, expectedFuture+1)
	if tagEvent.EventType != "departed" {
		t.Errorf("Expected tag to be changed to departed, but it's type is %s", tagEvent.EventType)
	}
}

func TestNotYetDeparted(t *testing.T) {
	timestamp := 1500583263000 // Thursday, July 20, 2017 1:41:03 PM
	minutesBeforeAgeOut := 10

	tagEvent := jsonrpc.TagEvent{
		EventType:  "cycle_count",
		FacilityID: "testFacility",
		Timestamp:  int64(timestamp),
	}

	ageOuts := map[string]int{
		"testFacility": minutesBeforeAgeOut,
	}

	// calculate the future time after which we should consider this departed
	expectedFuture := int64(timestamp + (minutesBeforeAgeOut * 60 * 1000))

	markDepartedIfUnseen(&tagEvent, ageOuts, expectedFuture-1)
	if tagEvent.EventType != "cycle_count" {
		t.Errorf("Expected tag to be stay as cycle_count, but it's type is %s", tagEvent.EventType)
	}
}

func TestNotACycleCount(t *testing.T) {
	timestamp := 1500583263000 // Thursday, July 20, 2017 1:41:03 PM
	minutesBeforeAgeOut := 10

	tagEvent := jsonrpc.TagEvent{
		EventType:  "arrival",
		FacilityID: "testFacility",
		Timestamp:  int64(timestamp),
	}

	ageOuts := map[string]int{
		"testFacility": minutesBeforeAgeOut,
	}

	// calculate the future time after which we should consider this departed
	// it won't become departed, though, because the event is not a cycle_count
	expectedFuture := int64(timestamp + (minutesBeforeAgeOut * 60 * 1000))

	markDepartedIfUnseen(&tagEvent, ageOuts, expectedFuture+1)
	if tagEvent.EventType != "arrival" {
		t.Errorf("Expected tag to be stay as arrival, but it's type is %s", tagEvent.EventType)
	}
}

func TestUnknownFacility(t *testing.T) {
	timestamp := 1500583263000 // Thursday, July 20, 2017 1:41:03 PM
	minutesBeforeAgeOut := 10

	tagEvent := jsonrpc.TagEvent{
		EventType:  "cycle_count",
		FacilityID: "testFacility",
		Timestamp:  int64(timestamp),
	}

	ageOuts := map[string]int{
		"someOtherFacility": minutesBeforeAgeOut,
	}

	// calculate the future time after which we should consider this departed
	// it won't become departed, though, because the facility isn't in the ageOuts config
	expectedFuture := int64(timestamp + (minutesBeforeAgeOut * 60 * 1000))

	markDepartedIfUnseen(&tagEvent, ageOuts, expectedFuture+1)
	if tagEvent.EventType != "cycle_count" {
		t.Errorf("Expected tag to be stay as cycle_count, but it's type is %s", tagEvent.EventType)
	}
}

//nolint:dupl
func TestFilter(t *testing.T) {
	testTag := jsonrpc.TagEvent{
		EpcCode: "302103201",
	}

	filters := []string{"123", "302", "456"}

	expected := true

	if result := statemodel.IsTagWhitelisted(testTag.EpcCode, filters); result != expected {
		t.Errorf("Filtering failed. Expected %v. Actual %v.", expected, result)
	}
}

//nolint:dupl
func TestFilterNotPresent(t *testing.T) {
	testTag := jsonrpc.TagEvent{
		EpcCode: "402103201",
	}

	filters := []string{"123", "302", "456"}
	expected := false

	if result := statemodel.IsTagWhitelisted(testTag.EpcCode, filters); result != expected {
		t.Errorf("Filtering failed. Expected %v. Actual %v.", expected, result)
	}
}

func TestFilterNoTags(t *testing.T) {
	testTag := jsonrpc.TagEvent{
		EpcCode: "402103201",
	}

	var filters []string
	expected := false

	if result := statemodel.IsTagWhitelisted(testTag.EpcCode, filters); result != expected {
		t.Errorf("Filtering failed. Expected %v. Actual %v.", expected, result)
	}
}

func TestDataProcessHandheld(t *testing.T) {

	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer func() {
			if _, err := ioutil.ReadAll(request.Body); err != nil {
				t.Error(err)
			}
		}()

		switch request.URL.EscapedPath() {
		case "/skus":
			jsonData := getMappingSkuSample()
			writer.Header().Set("Content-Type", "application/json")
			if _, err := writer.Write(jsonData); err != nil {
				t.Fatal(err)
			}
		case "/callwebhook":
			t.Log("webhook called")
		default:
			t.Fatalf("unexpected request for %s", request.URL.EscapedPath())
		}
	}))
	defer testServer.Close()

	config.AppConfig.RulesUrl = ""
	config.AppConfig.CloudConnectorRetrySeconds = 1

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	JSONSample := getJSONSampleHandheld(t)
	skuMapping := NewSkuMapping(testServer.URL + "/skus")
	config.AppConfig.CloudConnectorUrl = testServer.URL

	// insert data as handheld
	if err := skuMapping.processTagData(newInventoryApp(testDB.DB), JSONSample, "handheld", nil); err != nil {
		t.Errorf("error processing data %+v", err)
	}
}

func TestDataProcessFixedAllRulesTriggered(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.EscapedPath() != "/triggerrules" &&
			request.URL.EscapedPath() != "/skus" &&
			request.URL.EscapedPath() != "/callwebhook" {
			t.Fatalf("unexpected request path: %s", request.URL.EscapedPath())
		}

		var jsonData []byte
		if request.URL.EscapedPath() == "/triggerrules" {
			if request.Method != "POST" {
				t.Errorf("Expected 'POST' request, received '%s", request.Method)
			}

			queryString := request.URL.Query()
			ruleTypes := queryString["ruletype"]

			if len(ruleTypes) != 0 {
				t.Error("Expected trigger all rules")
			}

			var data interface{}

			jsonData, _ = json.Marshal(tag.Response{Results: data})
		}
		if request.URL.EscapedPath() == "/skus" {
			jsonData = getMappingSkuSample()
		}

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(jsonData)
	}))

	defer testServer.Close()

	config.AppConfig.TriggerRulesEndpoint = "/triggerrules"
	config.AppConfig.RulesUrl = testServer.URL
	config.AppConfig.CloudConnectorApiGatewayEndpoint = "/callwebhook"
	config.AppConfig.CloudConnectorUrl = testServer.URL
	config.AppConfig.TriggerRulesOnFixedTags = true

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	JSONSample := getJSONDepartedSample(t)
	skuMapping := NewSkuMapping(testServer.URL + "/skus")
	// insert data as fixed
	if err := skuMapping.processTagData(newInventoryApp(testDB.DB), JSONSample, "fixed", nil); err != nil {
		t.Errorf("error processing data %s", err.Error())
	}

}

func getJSONDepartedSample(t *testing.T) *jsonrpc.InventoryEvent {
	return createInventoryEvent(t, `{	
				 "controller_id": "rsp-controller",
				 "total_event_segments": 1,
				 "event_segment_number": 1,
				 "data": [
							 {
								 "epc_code": "30143639F84191AD22900204",
								 "epc_encode_format": "tbd",
								 "event_type": "departed",
								 "facility_id": "Tavern",
								 "location": "RSP-95bd71",
								 "tid": null,
								 "timestamp": 1501863300375
							 }
						 ],
				 "sent_on": 1501872400247
   }`)
}

func TestDataProcessFixedNoOoSRulesTriggered(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.EscapedPath() != "/triggerrules" && request.URL.EscapedPath() != "/skus" && request.URL.EscapedPath() != "/callwebhook" {
			t.Errorf("Expected request to '/triggerrules', received %s", request.URL.EscapedPath())
		}
		var jsonData []byte
		if request.URL.EscapedPath() == "/triggerrules" {
			if request.Method != "POST" {
				t.Errorf("Expected 'POST' request, received '%s", request.Method)
			}

			queryString := request.URL.Query()
			ruleTypes := queryString["ruletype"]

			if len(ruleTypes) == 0 {
				t.Error("Expected to NOT trigger all rules")
			}

			if len(ruleTypes) != 1 || ruleTypes[0] != tag.StateChangeEvent {
				t.Error("Expected to only trigger State Change rules")
			}

			var data interface{}

			jsonData, _ = json.Marshal(tag.Response{Results: data})
		}
		if request.URL.EscapedPath() == "/skus" {
			jsonData = getMappingSkuSample()
		}
		if request.URL.EscapedPath() == "/callwebhook" {

			var data event.TagEventPayload
			body, err := ioutil.ReadAll(request.Body)
			if err != nil {
				t.Errorf(err.Error())
			}

			if err := json.Unmarshal(body, &data); err != nil {
				t.Errorf(err.Error())
			}

			for _, tag := range data.Body.TagEvent {
				if tag.Event != statemodel.DepartedEpcState {
					t.Errorf("Event type was not departed")
				}
			}
		}

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(jsonData)
	}))

	defer testServer.Close()

	config.AppConfig.TriggerRulesEndpoint = "/triggerrules"
	config.AppConfig.RulesUrl = testServer.URL
	config.AppConfig.TriggerRulesOnFixedTags = false

	config.AppConfig.CloudConnectorApiGatewayEndpoint = "/callwebhook"
	config.AppConfig.CloudConnectorUrl = testServer.URL

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	JSONSample := getJSONDepartedSample(t)
	skuMapping := NewSkuMapping(testServer.URL + "/skus")
	// insert data as fixed
	if err := skuMapping.processTagData(newInventoryApp(testDB.DB), JSONSample, "fixed", nil); err != nil {
		t.Errorf("error processing data %s", err.Error())
	}

}

func TestTagExistingArrivalReceiveCycleCountUpstreamCycleCount(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.EscapedPath() != "/triggerrules" && request.URL.EscapedPath() != "/skus" && request.URL.EscapedPath() != "/callwebhook" {
			t.Errorf("Expected request to '/triggerrules', received %s", request.URL.EscapedPath())
		}
		var jsonData []byte
		if request.URL.EscapedPath() == "/triggerrules" {
			if request.Method != "POST" {
				t.Errorf("Expected 'POST' request, received '%s", request.Method)
			}

			queryString := request.URL.Query()
			ruleTypes := queryString["ruletype"]

			if len(ruleTypes) == 0 {
				t.Error("Expected to NOT trigger all rules")
			}

			if len(ruleTypes) != 1 || ruleTypes[0] != tag.StateChangeEvent {
				t.Error("Expected to only trigger State Change rules")
			}

			var data interface{}

			jsonData, _ = json.Marshal(tag.Response{Results: data})
		}
		if request.URL.EscapedPath() == "/skus" {
			jsonData = getMappingSkuSample()
		}
		if request.URL.EscapedPath() == "/callwebhook" {

			var data event.TagEventPayload
			body, err := ioutil.ReadAll(request.Body)
			if err != nil {
				t.Errorf(err.Error())
			}

			if err := json.Unmarshal(body, &data); err != nil {
				t.Errorf(err.Error())
			}

			for _, tag := range data.Body.TagEvent {
				if tag.Event != statemodel.CycleCountEvent {
					t.Errorf("Event type was %s and not cycle_count", tag.Event)
				}
			}
		}

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(jsonData)
	}))

	defer testServer.Close()

	config.AppConfig.TriggerRulesEndpoint = "/triggerrules"
	config.AppConfig.RulesUrl = testServer.URL
	config.AppConfig.TriggerRulesOnFixedTags = false

	config.AppConfig.CloudConnectorApiGatewayEndpoint = "/callwebhook"
	config.AppConfig.CloudConnectorUrl = testServer.URL

	tagArray := make([]tag.Tag, 2)

	var tag0 tag.Tag
	tag0.Epc = "303400C0E43FF48000000002"
	sgtin, err := epc.DecodeSGTINString(tag0.Epc)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	uri := sgtin.URI()
	tag0.URI = uri
	tag0.Tid = t.Name() + "0"
	tag0.Source = "fixed"
	tag0.Event = statemodel.ArrivalEvent
	tag0.EpcState = statemodel.PresentEpcState
	tagArray[0] = tag0

	var tag1 tag.Tag
	tag1.Epc = "30143639F8419145DB601597"
	tag0.Epc = "303400C0E43FF48000000002"
	sgtin, err = epc.DecodeSGTINString(tag1.Epc)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	uri = sgtin.URI()
	tag0.URI = uri
	tag1.Tid = t.Name() + "1"
	tag1.Source = "fixed"
	tag1.Event = statemodel.ArrivalEvent
	tag1.EpcState = statemodel.PresentEpcState
	tagArray[1] = tag1

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	err = tag.Replace(testDB.DB, tagArray)
	if err != nil {
		t.Error("Unable to replace tags", err.Error())
	}

	JSONSample := createInventoryEvent(t, `{			 
				 "controller_id": "rsp-controller",
				 "total_event_segments": 1,
				 "event_segment_number": 1,
				 "data": [
							 {
								 "epc_code": "303400C0E43FF48000000002",
								 "epc_encode_format": "tbd",
								 "event_type": "cycle_count",
								 "facility_id": "Tavern",
								 "location": "RSP-95bd71",
								 "tid": null,
								 "timestamp": 1501863300375
							 },
							 {
								 "epc_code": "30143639F8419145DB601597",
								 "epc_encode_format": "tbd",
								 "event_type": "cycle_count",
								 "facility_id": "Tavern",
								 "location": "RSP-95bd71",
								 "tid": null,
								 "timestamp": 1501864735850
							 }
						 ],
				 "sent_on": 1501872400247
   }`)

	skuMapping := NewSkuMapping(testServer.URL + "/skus")
	// insert data as fixed
	if err := skuMapping.processTagData(newInventoryApp(testDB.DB), JSONSample, "fixed", nil); err != nil {
		t.Errorf("error processing data %s", err.Error())
	}

}

// TestTagExistingMovedReceiveCycleCountUpstreamCycleCount tests that current
// tags existed and were not changed thus the event type cycle_count
func TestTagExistingMovedReceiveCycleCountUpstreamCycleCount(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.EscapedPath() != "/triggerrules" && request.URL.EscapedPath() != "/skus" && request.URL.EscapedPath() != "/callwebhook" {
			t.Errorf("Expected request to '/triggerrules', received %s", request.URL.EscapedPath())
		}
		var jsonData []byte
		if request.URL.EscapedPath() == "/triggerrules" {
			if request.Method != "POST" {
				t.Errorf("Expected 'POST' request, received '%s", request.Method)
			}

			queryString := request.URL.Query()
			ruleTypes := queryString["ruletype"]

			if len(ruleTypes) == 0 {
				t.Error("Expected to NOT trigger all rules")
			}

			if len(ruleTypes) != 1 || ruleTypes[0] != tag.StateChangeEvent {
				t.Error("Expected to only trigger State Change rules")
			}

			var data interface{}

			jsonData, _ = json.Marshal(tag.Response{Results: data})
		}
		if request.URL.EscapedPath() == "/skus" {
			jsonData = getMappingSkuSample()
		}
		if request.URL.EscapedPath() == "/callwebhook" {

			var data event.TagEventPayload
			body, err := ioutil.ReadAll(request.Body)
			if err != nil {
				t.Errorf(err.Error())
			}

			if err := json.Unmarshal(body, &data); err != nil {
				t.Errorf(err.Error())
			}

			for _, tag := range data.Body.TagEvent {
				if tag.Event != statemodel.CycleCountEvent {
					t.Errorf("Event type was %s and not cycle_count", tag.Event)
				}
			}
		}

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(jsonData)
	}))

	defer testServer.Close()

	config.AppConfig.TriggerRulesEndpoint = "/triggerrules"
	config.AppConfig.RulesUrl = testServer.URL
	config.AppConfig.TriggerRulesOnFixedTags = false

	config.AppConfig.CloudConnectorApiGatewayEndpoint = "/callwebhook"
	config.AppConfig.CloudConnectorUrl = testServer.URL

	tagArray := make([]tag.Tag, 4)

	var tag0 tag.Tag
	tag0.Epc = "30143639F8419145DB601567"
	sgtin, err := epc.DecodeSGTINString(tag0.Epc)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	tag0.URI = sgtin.URI()
	tag0.Tid = t.Name() + "0"
	tag0.Source = "fixed"
	tag0.Event = statemodel.MovedEvent
	tag0.EpcState = statemodel.PresentEpcState
	tagArray[0] = tag0

	var tag1 tag.Tag
	tag1.Epc = "30343639F8419145DB601443"
	sgtin, err = epc.DecodeSGTINString(tag1.Epc)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	tag1.URI = sgtin.URI()
	tag1.Tid = t.Name() + "1"
	tag1.Source = "fixed"
	tag1.Event = statemodel.MovedEvent
	tag1.EpcState = statemodel.PresentEpcState
	tagArray[1] = tag1

	var tag2 tag.Tag
	tag2.Epc = "3014032F440010C5407BA3FB"
	sgtin, err = epc.DecodeSGTINString(tag2.Epc)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	tag2.URI = sgtin.URI()
	tag2.Tid = t.Name() + "0"
	tag2.Source = "fixed"
	tag2.Event = statemodel.MovedEvent
	tag2.EpcState = statemodel.PresentEpcState
	tagArray[2] = tag2

	var tag3 tag.Tag
	tag3.Epc = "30143639F8419145DB601543"
	sgtin, err = epc.DecodeSGTINString(tag3.Epc)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	tag3.URI = sgtin.URI()
	tag3.Tid = t.Name() + "1"
	tag3.Source = "fixed"
	tag3.Event = statemodel.MovedEvent
	tag3.EpcState = statemodel.PresentEpcState
	tagArray[3] = tag3

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	err = tag.Replace(testDB.DB, tagArray)
	if err != nil {
		t.Error("Unable to replace tags", err.Error())
	}

	JSONSample1 := createInventoryEvent(t, `{			 
				 "controller_id": "rsp-controller",
				 "event_segment_number": 1,
				 "total_event_segments": 2,
				 "data": [
							 {
								 "epc_code": "30143639F8419145DB601567",
								 "epc_encode_format": "tbd",
								 "event_type": "cycle_count",
								 "facility_id": "Tavern",
								 "location": "RSP-95bd71",
								 "tid": null,
								 "timestamp": 1501863300375
							 },
							 {
								 "epc_code": "30143639F8419145DB601567",
								 "epc_encode_format": "tbd",
								 "event_type": "cycle_count",
								 "facility_id": "Tavern",
								 "location": "RSP-95bd71",
								 "tid": null,
								 "timestamp": 1501864735850
							 }
						 ],
				 "sent_on": 1501872400247
  }`)
	JSONSample2 := createInventoryEvent(t, `{			 
				 "controller_id": "rsp-controller",
				 "event_segment_number": 2,
				 "total_event_segments": 2,
				 "data": [
							 {
								 "epc_code": "3014032F440010C5407BA3FB",
								 "epc_encode_format": "tbd",
								 "event_type": "cycle_count",
								 "facility_id": "Tavern",
								 "location": "RSP-95bd71",
								 "tid": null,
								 "timestamp": 1501863300375
							 },
							 {
								 "epc_code": "30143639F8419145DB601543",
								 "epc_encode_format": "tbd",
								 "event_type": "cycle_count",
								 "facility_id": "Tavern",
								 "location": "RSP-95bd71",
								 "tid": null,
								 "timestamp": 1501864735850
							 }
						 ],
				 "sent_on": 1501872400247
  }`)
	skuMapping := NewSkuMapping(testServer.URL + "/skus")
	// insert data as fixed
	if err := skuMapping.processTagData(newInventoryApp(testDB.DB), JSONSample1, "fixed", nil); err != nil {
		t.Errorf("error processing data %s", err.Error())
	}
	if err := skuMapping.processTagData(newInventoryApp(testDB.DB), JSONSample2, "fixed", nil); err != nil {
		t.Errorf("error processing data %s", err.Error())
	}

}

// TestTagExistingDepartedReceiveCycleCountUpstreamArrival tests that current tags that have departed events will be
// changed to Arrival when a cycle count is recieved with those EPC tags
func TestTagExistingDepartedReceiveCycleCountUpstreamArrival(t *testing.T) {
	rulesTriggered := make(chan bool, 1)

	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Log("request for", request.URL)

		defer func() {
			if _, err := ioutil.ReadAll(request.Body); err != nil {
				t.Error("error reading body", err)
			}
		}()

		var err error
		var jsonData []byte
		switch request.URL.Path {
		case "/triggerrules":
			if request.Method != "POST" {
				t.Errorf("Expected 'POST' request, received '%s", request.Method)
			}

			queryString := request.URL.Query()
			ruleTypes := queryString["ruletype"]

			if ruleTypes == nil {
				t.Error("missing ruletype from query string")
			} else if len(ruleTypes) == 0 {
				t.Error("Expected to NOT trigger all rules")
			} else if len(ruleTypes) != 1 || ruleTypes[0] != tag.StateChangeEvent {
				t.Error("Expected to only trigger State Change rules")
			} else {
				rulesTriggered <- true
			}

			var data interface{}
			jsonData, err = json.Marshal(tag.Response{Results: data})
			if err != nil {
				t.Error("marshaling failed", err)
			}
		case "/skus":
			jsonData = getMappingSkuSample()
		case "/callwebhook":
			var data event.TagEventPayload
			body, err := ioutil.ReadAll(request.Body)
			if err != nil {
				t.Fatal(err)
			}

			if err = json.Unmarshal(body, &data); err != nil {
				t.Error(err)
			}

			for tagIdx, tagEvent := range data.Body.TagEvent {
				if tagEvent.Event != statemodel.ArrivalEvent {
					t.Errorf("Event type for tag %d was not arrival", tagIdx)
				}
			}
		default:
			t.Error("unexpected request", request.URL.EscapedPath())
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		if _, err = writer.Write(jsonData); err != nil {
			t.Error(err)
		}
	}))

	defer testServer.Close()

	config.AppConfig.TriggerRulesEndpoint = "/triggerrules"
	config.AppConfig.RulesUrl = testServer.URL
	config.AppConfig.TriggerRulesOnFixedTags = false
	config.AppConfig.CloudConnectorApiGatewayEndpoint = "/callwebhook"
	config.AppConfig.CloudConnectorUrl = testServer.URL

	tagArray := make([]tag.Tag, 2)

	var tag0 tag.Tag
	tag0.Epc = "30143639F8419145DB601529"
	sgtin, err := epc.DecodeSGTINString(tag0.Epc)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	tag0.URI = sgtin.URI()
	tag0.Tid = t.Name() + "0"
	tag0.Source = "fixed"
	tag0.Event = statemodel.DepartedEvent
	tag0.EpcState = statemodel.DepartedEpcState
	tagArray[0] = tag0

	var tag1 tag.Tag
	tag1.Epc = "30143639F8419145DB601565"
	sgtin, err = epc.DecodeSGTINString(tag1.Epc)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	tag1.URI = sgtin.URI()
	tag1.Tid = t.Name() + "1"
	tag1.Source = "fixed"
	tag1.Event = statemodel.DepartedEvent
	tag1.EpcState = statemodel.DepartedEpcState
	tagArray[1] = tag1

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	err = tag.Replace(testDB.DB, tagArray)
	if err != nil {
		t.Errorf("Unable to replace tags: %+v", err)
	}

	JSONSample := createInventoryEvent(t, `{			 
				 "controller_id": "rsp-controller",
         "total_event_segments": 1,
         "event_segment_number": 1,
				 "data": [
							 {
								 "epc_code": "30143639F8419145DB601529",
								 "epc_encode_format": "tbd",
								 "event_type": "cycle_count",
								 "facility_id": "Tavern",
								 "location": "RSP-95bd71",
								 "tid": null,
								 "timestamp": 1501863300375
							 },
							 {
								 "epc_code": "30143639F8419145DB601565",
								 "epc_encode_format": "tbd",
								 "event_type": "cycle_count",
								 "facility_id": "Tavern",
								 "location": "RSP-95bd71",
								 "tid": null,
								 "timestamp": 1501864735850
							 }
						 ],
				 "sent_on": 1501872400247
   }`)

	skuMapping := NewSkuMapping(testServer.URL + "/skus")
	// insert data as fixed
	if err := skuMapping.processTagData(newInventoryApp(testDB.DB), JSONSample, "fixed", nil); err != nil {
		t.Errorf("error processing data %+v", err)
	}

	select {
	case <-rulesTriggered:
	case <-time.After(time.Second):
		t.Error("the rules were never triggered")
	}
}

// TestTagDoesNotExistReceiveCycleCountUpstreamArrival tests that current tags
// in the cycle count are new to the database and the event is changed to Arrival.
func TestTagDoesNotExistReceiveCycleCountUpstreamArrival(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	rulesTriggered := make(chan bool, 1)

	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Log("request for", request.URL)

		defer func() {
			if _, err := ioutil.ReadAll(request.Body); err != nil {
				t.Error("error reading body", err)
			}
		}()

		var err error
		var jsonData []byte

		switch request.URL.Path {
		case "/triggerrules":
			if request.Method != "POST" {
				t.Errorf("Expected 'POST' request, received '%s", request.Method)
			}

			queryString := request.URL.Query()
			ruleTypes := queryString["ruletype"]

			if ruleTypes == nil {
				t.Error("missing ruletype from query string")
			} else if len(ruleTypes) == 0 {
				t.Error("Expected to NOT trigger all rules")
			} else if len(ruleTypes) != 1 || ruleTypes[0] != tag.StateChangeEvent {
				t.Error("Expected to only trigger State Change rules")
			} else {
				rulesTriggered <- true
			}

			var data interface{}
			jsonData, err = json.Marshal(tag.Response{Results: data})
			if err != nil {
				t.Error(err)
			}
		case "/skus":
			jsonData = getMappingSkuSample()
		case "/callwebhook":
			var data event.TagEventPayload
			body, err := ioutil.ReadAll(request.Body)
			if err != nil {
				t.Errorf(err.Error())
			}

			if err := json.Unmarshal(body, &data); err != nil {
				t.Errorf(err.Error())
			}

			for tagIdx, tagEvent := range data.Body.TagEvent {
				log.Info(tagEvent)
				if tagEvent.Event != statemodel.ArrivalEvent {
					t.Errorf("When a cycle count event is recieved from the "+
						"RSP Controller AND the tag doesn't exist in the database, the event type "+
						"should be Arrival, but for tag %d, it was %s: %+v.",
						tagIdx, tagEvent.Event, tagEvent)
				}
			}
		default:
			t.Error("unexpected request", request.URL.EscapedPath())
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		if _, err = writer.Write(jsonData); err != nil {
			t.Error(err)
		}
	}))

	defer testServer.Close()

	config.AppConfig.TriggerRulesEndpoint = "/triggerrules"
	config.AppConfig.RulesUrl = testServer.URL
	config.AppConfig.CloudConnectorApiGatewayEndpoint = "/callwebhook"
	config.AppConfig.CloudConnectorUrl = testServer.URL
	config.AppConfig.TriggerRulesOnFixedTags = false

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	JSONSample := getJSONCycleCountSample(t)
	skuMapping := NewSkuMapping(testServer.URL + "/skus")
	// insert data as fixed
	if err := skuMapping.processTagData(newInventoryApp(testDB.DB), JSONSample, "fixed", nil); err != nil {
		t.Errorf("error processing data %+v", err)
	}

	select {
	case <-rulesTriggered:
	case <-time.After(time.Second):
		t.Error("the rules were never triggered")
	}
}

func TestDataProcessFixedWhitelisted(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.EscapedPath() != "/triggerrules" && request.URL.EscapedPath() != "/skus" && request.URL.EscapedPath() != "/callwebhook" {
			t.Errorf("Expected request to '/triggerrules', received %s", request.URL.EscapedPath())
		}
		var jsonData []byte
		if request.URL.EscapedPath() == "/triggerrules" {
			if request.Method != "POST" {
				t.Errorf("Expected 'POST' request, received '%s", request.Method)
			}

			queryString := request.URL.Query()
			ruleTypes := queryString["ruletype"]

			if len(ruleTypes) == 0 {
				t.Error("Expected to NOT trigger all rules")
			}

			if len(ruleTypes) != 1 || ruleTypes[0] != tag.StateChangeEvent {
				t.Error("Expected to only trigger State Change rules")
			}

			var data interface{}

			jsonData, _ = json.Marshal(tag.Response{Results: data})
		}
		if request.URL.EscapedPath() == "/skus" {
			jsonData = getMappingSkuSample()
		}

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(jsonData)
	}))

	defer testServer.Close()

	config.AppConfig.TriggerRulesEndpoint = "/triggerrules"
	config.AppConfig.CloudConnectorApiGatewayEndpoint = "/callwebhook"
	config.AppConfig.CloudConnectorUrl = testServer.URL
	config.AppConfig.RulesUrl = testServer.URL
	config.AppConfig.TriggerRulesOnFixedTags = false

	// Filter through only those starting with "30"
	config.AppConfig.EpcFilters = []string{"30"}

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	JSONSample := createInventoryEvent(t, `{			 
				 "controller_id": "rsp-controller",
				 "data": [
							 {
								 "epc_code": "30243639F84191AD22900266",
								 "epc_encode_format": "tbd",
								 "event_type": "cycle_count",
								 "facility_id": "Tavern",
								 "location": "RSP-95bd71",
								 "tid": null,
								 "timestamp": 1501863300375
							 },
							 {
								 "epc_code": "0F0000000000AA00000014D2",
								 "epc_encode_format": "tbd",
								 "event_type": "cycle_count",
								 "facility_id": "Tavern",
								 "location": "RSP-95bd71",
								 "tid": null,
								 "timestamp": 1501864735850
							 }
						 ],
				 "sent_on": 1501872400247
   }`)
	skuMapping := NewSkuMapping(testServer.URL + "/skus")
	// insert data as fixed
	if err := skuMapping.processTagData(newInventoryApp(testDB.DB), JSONSample, "fixed", nil); err != nil {
		t.Errorf("error processing data %s", err.Error())
	}

	getNotWhitelistedTag, err := tag.FindByEpc(testDB.DB, "0F0000000000AA00000014D2")
	if err != nil {
		t.Fatalf("Error retrieving tag from database: %s", err.Error())
	}

	// it should be empty because it was not whitelisted
	if !getNotWhitelistedTag.IsEmpty() {
		t.Errorf("Tag was not whitelisted.  Should not be in database")
	}

	getWhitelistedTag, err := tag.FindByEpc(testDB.DB, "30243639F84191AD22900266")
	if err != nil {
		t.Fatalf("Error retrieving tag from database: %s", err.Error())
	}

	// it should be empty because it was not whitelisted
	if getWhitelistedTag.IsEmpty() {
		t.Errorf("Tag was whitelisted.  Should be in database")
	}
}

func TestProcessHeartBeat(t *testing.T) {

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	JSONSample := createHeartbeat(t, `{
		   "controller_id": "rsp-controller",
		   "device_id": "rsp-controller",
		   "facilities": [
			 "Tavern"
		   ],
		   "facility_groups_cfg": "auto-0802233641",
		   "mesh_id": null,
		   "mesh_node_id": null,
		   "personality_groups_cfg": null,
		   "schedule_cfg": "UNKNOWN",
		   "schedule_groups_cfg": null,
		   "sent_on": 1503700192960		 
	   }`)

	if err := heartbeat.ProcessHeartbeat(JSONSample, testDB.DB); err != nil {
		t.Errorf("error processing hearbeat data %s", err.Error())
	}
}

func TestProcessShippingNotice(t *testing.T) {

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()
	clearAllData(t, testDB.DB)

	config.AppConfig.EpcFilters = []string{"303", "301"}
	JSONShippingNotice := []byte(`	
		[
			{
				"asnId": "AS876422",
				"eventTime": "2018-03-12T12: 34: 56.789Z",
				"siteId": "0105",
				"items": [
					{
						"itemId": "large lamp",
						"itemGtin": "00888446671424",
						"itemEpcs": [
							"30343639F84191AD22900204",
							"30143639F84191AD22900205",
							"30143639F84191AD22900206"
						]
					}
				]
			}
		]  
	`)

	// make sure the tag doesn't currently exist
	gotTag, err := tag.FindByEpc(testDB.DB, "30343639F84191AD22900204")
	if err != nil {
		t.Errorf("Error retrieving tag from database: %+v", err)
	} else {
		// the tag should be empty
		if !gotTag.IsEmpty() {
			t.Errorf("tag should be empty, but was: %+v", gotTag)
		}
		// new tags shouldn't default to being shipping notices
		if gotTag.IsShippingNoticeEntry() {
			t.Errorf("tag shouldn't yet have a Advance Shipping Notice: %+v", gotTag)
		}
	}

	// process the ASN
	if err = processShippingNotice(JSONShippingNotice, testDB.DB, nil); err != nil {
		t.Errorf("error processing data: %+v", err)
	}

	//now get the tag again; this time, it should exist
	gotTag, err = tag.FindByEpc(testDB.DB, "30343639F84191AD22900204")
	if err != nil {
		t.Fatalf("Error retrieving tag from database: %+v", err)
	}

	// it should not be empty, but it should only exist as an ASN
	if gotTag.IsEmpty() || !gotTag.IsShippingNoticeEntry() {
		t.Errorf("After processing ASN, new tag is not marked result of ASN: %+v", gotTag)
	}

	var asn tag.ASNContext
	if err := json.Unmarshal([]byte(gotTag.EpcContext), &asn); err != nil {
		t.Errorf("unable to unmarshal ASN in EPC context %+v", err)
	}
	checkASNContext(t, &asn)
}

func TestProcessShippingNoticeWhitelistedEPC(t *testing.T) {
	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	jsonShippingNotice := []byte(`
		[
			{
				"asnId": "AS876422",
				"eventTime": "2018-03-12T12: 34: 56.789Z",
				"siteId": "0105",
				"items": [
					{
						"itemId": "large lamp",
						"itemGtin": "00888446671424",
						"itemEpcs": [
							"3034257BF400B7800004CB2F"
						]
					}
				]
			}
		]		
	`)

	// Filter through only those starting with "30"
	config.AppConfig.EpcFilters = []string{"30"}

	// make sure the tag doesn't currently exist
	gotTag, err := tag.FindByEpc(testDB.DB, "3034257BF400B7800004CB2F")
	if err != nil {
		t.Errorf("Error retrieving tag from database: %s", err.Error())
	} else {
		// the tag should be empty
		if !gotTag.IsEmpty() {
			t.Errorf("tag should be empty, but was: %v", gotTag)
		}
		// new tags shouldn't default to being shipping notices
		if gotTag.IsShippingNoticeEntry() {
			t.Errorf("tag shouldn't yet have a Advance Shipping Notice: %v", gotTag)
		}
	}

	// process the ASN
	if err = processShippingNotice(jsonShippingNotice, testDB.DB, nil); err != nil {
		t.Errorf("error processing data %s", err.Error())
	}

	clearAllData(t, testDB.DB)

	// now get the tag again; this time, it should exist
	gotTag, err = tag.FindByEpc(testDB.DB, "3034257BF400B7800004CB2F")
	if err != nil {
		t.Fatalf("Error retrieving tag from database: %s", err.Error())
	}
}

func TestProcessShippingNoticeExistingTag(t *testing.T) {
	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	jsonShippingNotice := []byte(`
		[
			{
				"asnId": "AS876422",
				"eventTime": "2018-03-12T12: 34: 56.789Z",
				"siteId": "0105",
				"items": [
					{
						"itemId": "large lamp",
						"itemGtin": "00888446671424",
						"itemEpcs": [
						  "3034257BF400B7800004CB2F"
						]
					}
				]
			}
		]
	`)

	// insert a known tag
	existingTag := getTagData()[0]
	existingTag.Epc = "3034257BF400B7800004CB2F"
	w := expect.WrapT(t).StopOnMismatch().As(existingTag)
	w.ShouldNotBeEqual(existingTag.LastRead, 0)
	w.ShouldSucceed(insert(testDB.DB, existingTag))
	w.ShouldSucceed(processShippingNotice(jsonShippingNotice, testDB.DB, nil))

	gotTag := w.ShouldHaveResult(tag.FindByEpc(testDB.DB, existingTag.Epc)).(tag.Tag)
	w = w.As(gotTag)
	w.ShouldBeFalse(gotTag.IsEmpty())
	w.ShouldBeFalse(gotTag.IsShippingNoticeEntry())
	w.ShouldBeEqual(gotTag.LastRead, existingTag.LastRead)

	clearAllData(t, testDB.DB)

	var asn tag.ASNContext
	w.As(gotTag.EpcContext).ShouldSucceed(json.Unmarshal([]byte(gotTag.EpcContext), &asn))
	checkASNContext(t, &asn)
}

func TestCallDeleteTagCollection(t *testing.T) {
	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	if err := callDeleteTagCollection(testDB.DB); err != nil {
		t.Fatalf("error on calling delete tag collection %s", err.Error())
	}
}

func getTagData() []tag.Tag {
	return []tag.Tag{
		{
			FacilityID:      "Tavern",
			LastRead:        helper.UnixMilli(time.Now().AddDate(0, 0, -1)),
			Epc:             "30143639F84191AD22900204",
			EpcEncodeFormat: "tbd",
			Event:           "cycle_count",
			LocationHistory: []tag.LocationHistory{
				{
					Location:  "RSP-950b44",
					Timestamp: 1506638821662,
					Source:    "fixed",
				}},
			Tid: "",
		},
	}
}

func getJSONCycleCountSample(t *testing.T) *jsonrpc.InventoryEvent {
	return createInventoryEvent(t, `{			 
				 "controller_id": "rsp-controller",
				 "total_event_segments": 1,
				 "event_segment_number": 1,
				 "data": [
							 {
								 "epc_code": "301430A55C0AC40000000008",
								 "epc_encode_format": "tbd",
								 "event_type": "cycle_count",
								 "facility_id": "Tavern",
								 "location": "RSP-95bd71",
								 "tid": null,
								 "timestamp": 1501863300375
							 },
							 {
								 "epc_code": "0F00000000000C00000014D2",
								 "epc_encode_format": "tbd",
								 "event_type": "cycle_count",
								 "facility_id": "Tavern",
								 "location": "RSP-95bd71",
								 "tid": null,
								 "timestamp": 1501864735850
							 }
						 ],
				 "sent_on": 1501872400247
   }`)
}

func createHeartbeat(t *testing.T, data string) *jsonrpc.Heartbeat {
	data = wrapJsonrpcParams("heartbeat", data)
	reading := &models.Reading{Value: data}
	js := new(jsonrpc.Heartbeat)
	if err := jsonrpc.Decode(reading.Value, js, nil); err != nil {
		t.Error(errors.Wrap(err, data))
	}
	return js
}

func checkASNContext(t *testing.T, asn *tag.ASNContext) {
	w := expect.WrapT(t).As(asn)

	switch asn.ASNID {
	case "AS876422":
		w.ShouldBeEqual(asn.EventTime, "2018-03-12T12: 34: 56.789Z")
		w.ShouldBeEqual(asn.SiteID, "0105")
	case "AS876423":
		w.ShouldBeEqual(asn.EventTime, "2019-03-12T12: 59: 56.789Z")
		w.ShouldBeEqual(asn.SiteID, "0106")
	default:
		w.Errorf("Wrong ASNID: %s", asn.ASNID)
		return
	}

	w.ShouldBeEqual(asn.ItemGTIN, "00888446671424")
	w.ShouldBeEqual(asn.ItemID, "large lamp")
}

// nolint :dupl
func insert(db *sql.DB, tag tag.Tag) error {
	obj, err := json.Marshal(tag)
	if err != nil {
		return err
	}

	upsertStmt := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s) 
									 ON CONFLICT (( %s  ->>  %s )) 
									 DO UPDATE SET %s = %s.%s || %s;`,
		pq.QuoteIdentifier(tagsTable),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteLiteral(string(obj)),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteLiteral(epcColumn),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteIdentifier(tagsTable),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteLiteral(string(obj)),
	)

	_, err = db.Exec(upsertStmt)
	if err != nil {
		return err
	}
	return nil
}

// nolint :dupl
func clearAllData(t *testing.T, db *sql.DB) {

	selectQuery := fmt.Sprintf(`DELETE FROM %s`,
		pq.QuoteIdentifier(tagsTable),
	)

	_, err := db.Exec(selectQuery)
	if err != nil {
		t.Errorf("Unable to delete data from %s table: %s", tagsTable, err)
	}
}

func wrapJsonrpcParams(method string, params string) string {
	sb := strings.Builder{}
	sb.WriteString(`{"jsonrpc":"2.0","method":"`)
	sb.WriteString(method)
	sb.WriteString(`","params":`)
	sb.WriteString(params)
	sb.WriteString(`}`)
	return sb.String()
}

func createInventoryEvent(t *testing.T, data string) *jsonrpc.InventoryEvent {
	data = wrapJsonrpcParams("inventory_event", data)
	reading := &models.Reading{Value: data}
	js := new(jsonrpc.InventoryEvent)
	if err := jsonrpc.Decode(reading.Value, js, nil); err != nil {
		t.Error(errors.Wrap(err, data))
	}
	return js
}

// controller_id is empty for handheld data
func getJSONSampleHandheld(t *testing.T) *jsonrpc.InventoryEvent {
	return createInventoryEvent(t, `{			 
				 "controller_id": "",
				 "data": [
							 {
								 "epc_code": "30143639F84191AD22900104",
								 "epc_encode_format": "tbd",
								 "event_type": "cycle_count",
								 "facility_id": "Tavern",
								 "location": "RSP-95bd71",
								 "tid": null,
								 "timestamp": 1501863300375
							 },
							 {
								 "epc_code": "30143639F84191AD66100107",
								 "epc_encode_format": "tbd",
								 "event_type": "cycle_count",
								 "facility_id": "Tavern",
								 "location": "RSP-95bd71",
								 "tid": null,
								 "timestamp": 1501864735850
							 }
						 ],
				 "sent_on": 1501872400247
   }`)
}

func getMappingSkuSample() []byte {
	return []byte(`{
  "results": [
    {
      "_id": "5bd105f16136e8cf3f152ea7",
      "sku": "12345679",
      "productList": [
        {
          "becomingReadable": 0.0456,
          "beingRead": 0.0123,
          "dailyTurn": 0.0121,
          "exitError": 0.0789,
          "metadata": {
            "color": "red",
            "size": "M"
          },
          "productId": "00400013635631"
        }
      ]
    },
    {
      "_id": "6aa105f16136e8cf3f152ea7",
      "sku": "70727815015607",
      "productList": [
        {
          "becomingReadable": 0.0456,
          "beingRead": 0.0123,
          "dailyTurn": 0.0121,
          "exitError": 0.0789,
          "metadata": {
            "color": "red",
            "size": "M"
          },
          "productId": "70727815015607"
        }
      ]
    }
  ]
}`)
}
