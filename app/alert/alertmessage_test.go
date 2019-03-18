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
package alert

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
)

func TestMain(m *testing.M) {
	if err := config.InitConfig(); err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func TestGenerateDeleteTagAlertMessagePayload(t *testing.T) {
	alertMessage := new(MessagePayload)
	payloadBytes, genErr := alertMessage.generateDeleteTagCollectionDoneMessage()
	if genErr != nil {
		t.Fatal("failed to generate alert message payload")
	}
	if len(payloadBytes) == 0 {
		t.Fatal("alert message payload bytes is empty")
	}
}

func TestSendAlertMessageDeleteCompletionOk(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("Expected 'POST' request, received '%s'", request.Method)
		}

		switch reqPath := request.URL.EscapedPath(); reqPath {
		case "/alertmessage":
			data := "rfid-alert/alertmessage ok"
			jsonData, _ := json.Marshal(data)
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write(jsonData)

		default:
			t.Errorf("Expected rfid-laert/alertmessage API endpoint, received '%s'", reqPath)
		}

	}))

	defer testServer.Close()

	config.AppConfig.RfidAlertMessageEndpoint = "/alertmessage"
	config.AppConfig.RfidAlertURL = testServer.URL

	alertMessage := new(MessagePayload)
	if err := alertMessage.SendDeleteTagCompletionAlertMessage(); err != nil {
		t.Fatalf("error sendDeleteTagCompletionAlertMessage %s", err.Error())
	}
}

func TestSendAlertMessageDeleteCompletionServerError(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("Expected 'POST' request, received '%s'", request.Method)
		}

		switch reqPath := request.URL.EscapedPath(); reqPath {
		case "/alertmessage":
			writer.WriteHeader(http.StatusInternalServerError)
		default:
			t.Errorf("Expected rfid-laert/alertmessage API endpoint, received '%s'", reqPath)
		}

	}))

	defer testServer.Close()

	config.AppConfig.RfidAlertMessageEndpoint = "/alertmessage"
	config.AppConfig.RfidAlertURL = testServer.URL

	alertMessage := new(MessagePayload)
	if err := alertMessage.SendDeleteTagCompletionAlertMessage(); err == nil {
		t.Fatal("expecting internal server error sendDeleteTagCompletionAlertMessage")
	}
}

func TestGenerateSendEventFailedAlertMessagePayload(t *testing.T) {
	alertMessage := new(MessagePayload)
	cloudConnectorPostURL := config.AppConfig.CloudConnectorUrl + "/events"
	payloadBytes, genErr := alertMessage.generateSendEventFailedAlertMessage(cloudConnectorPostURL)
	if genErr != nil {
		t.Fatal("failed to generate alert message payload")
	}
	if len(payloadBytes) == 0 {
		t.Fatal("alert message payload bytes is empty")
	}

	var alertMsgPayload MessagePayload
	if err := json.Unmarshal(payloadBytes, &alertMsgPayload); err != nil {
		t.Fatalf("incorrect payload bytes generated: %s", string(payloadBytes))
	}
	if alertMsgPayload.Value.AlertNumber != SendEventFailed {
		t.Fatalf("expecting alert number to be %d but found %d", SendEventFailed, alertMsgPayload.Value.AlertNumber)
	}
	if alertMsgPayload.Value.Severity != "critical" {
		t.Fatalf("expecting critical severity but found %s", alertMsgPayload.Value.Severity)
	}

	optionalField := alertMsgPayload.Value.Optional.(string)
	if !strings.Contains(optionalField, "cloudConnectorPostURL:") || !strings.Contains(optionalField, cloudConnectorPostURL) {
		t.Fatalf("expecting optional fields to have cloudConnectorPostURL: %s but found %s", cloudConnectorPostURL, optionalField)
	}
}

func TestSendEventFailedAlertMessageOk(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("Expected 'POST' request, received '%s'", request.Method)
		}

		switch reqPath := request.URL.EscapedPath(); reqPath {
		case "/alertmessage":
			data := "rfid-alert/alertmessage ok"
			jsonData, _ := json.Marshal(data)
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write(jsonData)

		default:
			t.Errorf("Expected rfid-laert/alertmessage API endpoint, received '%s'", reqPath)
		}

	}))

	defer testServer.Close()

	config.AppConfig.RfidAlertMessageEndpoint = "/alertmessage"
	config.AppConfig.RfidAlertURL = testServer.URL

	alertMessage := new(MessagePayload)
	cloudConnectorPostURL := config.AppConfig.CloudConnectorUrl + "/events"
	if err := alertMessage.SendEventPostFailedAlertMessage(cloudConnectorPostURL); err != nil {
		t.Fatalf("error SendEventPostFailedAlertMessage %s", err.Error())
	}
}
