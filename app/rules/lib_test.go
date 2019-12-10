/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package rules

import (
	"encoding/json"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/tag"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTriggerRulesWithData(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.EscapedPath() != "/triggerrules" {
			t.Errorf("Expected request to '/triggerrules', received %s", request.URL.EscapedPath())
		}
		if request.Method != "POST" {
			t.Errorf("Expected 'POST' request, received '%s", request.Method)
		}
		var data interface{}
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			t.Errorf(err.Error())
		}

		if err := json.Unmarshal(body, &data); err != nil {
			t.Errorf(err.Error())
		}

		if data.(string) != "testTriggerRules" {
			t.Errorf("Expected string data input")
		}

		jsonData, _ := json.Marshal(tag.Response{Results: data})
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(jsonData)
	}))

	defer testServer.Close()

	err := TriggerRules(testServer.URL+"/triggerrules", "testTriggerRules")
	if err != nil {
		t.Error(err)
	}
}

func TestTriggerRulesWithoutData(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.EscapedPath() != "/triggerrules" {
			t.Errorf("Expected request to '/triggerrules', received %s", request.URL.EscapedPath())
		}
		if request.Method != "POST" {
			t.Errorf("Expected 'POST' request, received '%s", request.Method)
		}
		var data interface{}
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			t.Errorf(err.Error())
		}

		if err := json.Unmarshal(body, &data); err != nil {
			t.Errorf(err.Error())
		}

		if data != nil {
			t.Errorf("Expected data to be nil")
		}

		jsonData, _ := json.Marshal(tag.Response{Results: data})
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(jsonData)
	}))

	defer testServer.Close()

	err := TriggerRules(testServer.URL+"/triggerrules", nil)
	if err != nil {
		t.Error(err)
	}
}

func TestTriggerRules_BadRequest(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusBadRequest)
	}))

	defer testServer.Close()

	err := TriggerRules(testServer.URL+"/triggerrules", nil)

	if err == nil {
		t.Errorf("Expecting StatusCode 400, but error was nil")
	}
}

func TestTriggerRules_TimeOut(t *testing.T) {
	orig := config.AppConfig.EndpointConnectionTimedOutSeconds
	config.AppConfig.EndpointConnectionTimedOutSeconds = 1
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(time.Duration(config.AppConfig.EndpointConnectionTimedOutSeconds+1) * time.Second)
	}))

	defer testServer.Close()

	err := TriggerRules(testServer.URL+"/triggerrules", nil)

	config.AppConfig.EndpointConnectionTimedOutSeconds = orig
	if err == nil {
		t.Errorf("Expecting timeout, but error was nil")
	}
}
