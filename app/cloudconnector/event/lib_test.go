/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package event

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/app/config"
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/app/tag"
	"github.com/intel/rsp-sw-toolkit-im-suite-utilities/helper"
)

func TestMain(m *testing.M) {
	if err := config.InitConfig(); err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func TestBuildEventPayload(t *testing.T) {
	header := http.Header{}
	header["Content-Type"] = []string{"application/json"}

	tags := getTagData()
	eventPayload := newEventPayload(tags, "test_id", 123456, 3, 1, header)
	if !reflect.DeepEqual(eventPayload.Headers, header) {
		t.Errorf("Error Building DataPayload")
	}

	if eventPayload.Body.ControllerId != "test_id" {
		t.Errorf("Error Building DataPayload")
	}
	if !eventPayload.Body.TagEvent[0].IsEqual(tags[0]) {
		t.Errorf("Error Building DataPayload")
	}
}

func TestTriggerCloudConnectorWithData(t *testing.T) {
	var tagData = getTagData()

	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(1 * time.Second)
		if request.URL.EscapedPath() != "/callwebhook" {
			t.Errorf("Expected request to be either '/callwebhook', received %s",
				request.URL.EscapedPath())
		}
		if request.Method != "POST" {
			t.Errorf("Expected 'POST' request, received '%s", request.Method)
		}
		var jsonData []byte
		if request.URL.EscapedPath() == "/aws-cloud/invoke" {
			var data []tag.Tag
			var ccPayload TagEventPayload
			body, err := ioutil.ReadAll(request.Body)
			if err != nil {
				t.Errorf(err.Error())
			}

			if err := json.Unmarshal(body, &ccPayload); err != nil {
				t.Errorf(err.Error())
			}

			if !ccPayload.Body.TagEvent[0].IsEqual(tagData[0]) {
				t.Errorf("Expected tag data input")
			}

			jsonData, _ = json.Marshal(data)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(jsonData)
	}))

	defer testServer.Close()

	err := TriggerCloudConnector("rsp-controller", 123456, 3, 1, tagData, testServer.URL+"/callwebhook")
	if err != nil {
		t.Error(err)
	}
}

func TestTriggerCloudConnectorWithoutData(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(1 * time.Second)
		if request.URL.EscapedPath() != "/callwebhook" {
			t.Errorf("Expected request to '/callwebhook', received %s",
				request.URL.EscapedPath())
		}
		if request.Method != "POST" {
			t.Errorf("Expected 'POST' request, received '%s", request.Method)
		}
		var jsonData []byte
		if request.URL.EscapedPath() == "/aws-cloud/invoke" {
			var data []tag.Tag
			var ccPayload TagEventPayload
			body, err := ioutil.ReadAll(request.Body)
			if err != nil {
				t.Errorf(err.Error())
			}

			if err := json.Unmarshal(body, &ccPayload); err != nil {
				t.Errorf(err.Error())
			}

			if len(ccPayload.Body.TagEvent) > 0 {
				t.Errorf("Expected tag data empty")
			}

			jsonData, _ = json.Marshal(data)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(jsonData)
	}))

	defer testServer.Close()

	err := TriggerCloudConnector("rsp-controller", 123456, 3, 1, nil, testServer.URL+"/callwebhook")
	if err != nil {
		t.Error(err)
	}
}

func TestCloudConnector_BadRequest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test with long timeout")
	}
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusBadRequest)
	}))

	defer testServer.Close()

	eOrig := config.AppConfig.EndpointConnectionTimedOutSeconds
	cOrig := config.AppConfig.CloudConnectorRetrySeconds
	config.AppConfig.EndpointConnectionTimedOutSeconds = 1
	config.AppConfig.CloudConnectorRetrySeconds = 0

	err := TriggerCloudConnector("controller_id", 123456, 3, 1, nil, testServer.URL+"/callwebhook")

	config.AppConfig.CloudConnectorRetrySeconds = cOrig
	config.AppConfig.EndpointConnectionTimedOutSeconds = eOrig

	if err == nil {
		t.Errorf("Expecting to get error for StatusCode 400 but found no error")
	}
}

func TestTriggerCloudConnector_TimedOut(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test with long timeout")
	}

	eOrig := config.AppConfig.EndpointConnectionTimedOutSeconds
	cOrig := config.AppConfig.CloudConnectorRetrySeconds
	config.AppConfig.EndpointConnectionTimedOutSeconds = 1
	config.AppConfig.CloudConnectorRetrySeconds = 0

	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(time.Duration(config.AppConfig.EndpointConnectionTimedOutSeconds+1) * time.Second)
	}))

	defer testServer.Close()

	err := TriggerCloudConnector("controller_id", 123456, 3, 1, nil, testServer.URL+"/callwebhook")

	config.AppConfig.CloudConnectorRetrySeconds = cOrig
	config.AppConfig.EndpointConnectionTimedOutSeconds = eOrig

	if err == nil {
		t.Errorf("Expecting to get error for timed-out but found no error")
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
