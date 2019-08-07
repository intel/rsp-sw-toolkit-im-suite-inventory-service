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

package event

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/alert"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/tag"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
)

const (
	cloudConnectorRetries = 5
)

// newEventPayload returns payload to send to cloud connector
func newEventPayload(tagData []tag.Tag, controllerId string, sentOn int64, totalEventSegments int, eventSegmentNumber int, header http.Header) EventPayload {

	eventPayload := EventPayload{
		Method:  http.MethodPost,
		Headers: header,
		IsAsync: false,
		URL:     config.AppConfig.EventDestination,
		Body: DataPayload{
			ControllerId:          controllerId,
			SentOn:             sentOn,
			TotalEventSegments: totalEventSegments,
			EventSegmentNumber: eventSegmentNumber,
			TagEvent:           tagData,
		},
	}

	// Unmarshal the auth data into the auth model for the cloud connector service to consume.
	if config.AppConfig.EventDestinationAuthEndpoint != "" &&
		config.AppConfig.EventDestinationAuthType != "" &&
		config.AppConfig.EventDestinationClientID != "" &&
		config.AppConfig.EventDestinationClientSecret != "" {
		// Encode the endpoint credentials as base64
		authDataString := config.AppConfig.EventDestinationClientID + ":" + config.AppConfig.EventDestinationClientSecret
		authData := "basic " + base64.StdEncoding.EncodeToString([]byte(authDataString))

		newAuth := Auth{
			Endpoint: config.AppConfig.EventDestinationAuthEndpoint,
			AuthType: config.AppConfig.EventDestinationAuthType,
			Data:     authData,
		}

		eventPayload.Auth = newAuth
	}

	return eventPayload
}

// TriggerCloudConnector sends payload it needs to go to external cloud, to the cloud connector
func TriggerCloudConnector(controllerId string, sentOn int64, totalEventSegments int, eventSegmentNumber int, tagData []tag.Tag, url string) error {
	log.Debugf("Making Cloud Connector call to: %s", url)
	// Metrics
	metrics.GetOrRegisterMeter(`InventoryService.triggerCloudConnector.Attempt`, nil).Mark(1)
	mTagEventSent := metrics.GetOrRegisterGaugeCollection(`InventoryService.triggerCloudConnector.Success`, nil)
	mGetErr := metrics.GetOrRegisterGauge(`InventoryService.triggerCloudConnector.triggerCloudConnector-Error`, nil)
	mSendEventFailed := metrics.GetOrRegisterGaugeCollection(`InventoryService.triggerCloudConnector.SendEventFailed`, nil)

	header := http.Header{}
	header["Content-Type"] = []string{"application/json"}
	eventPayload := newEventPayload(tagData, controllerId, sentOn, totalEventSegments, eventSegmentNumber, header)

	// Make the POST to authenticate
	eventPayloadBytes, err := json.Marshal(eventPayload)
	if err != nil {
		mGetErr.Update(1)
		return errors.Wrapf(err, "problem marshalling the data")
	}

	log.Debugf("Sending %d Bytes, %d Tags to Trigger Cloud Connector Event:\n", len(eventPayloadBytes), len(tagData))

	var cloudConnectorResponse *http.Response
	for attempt := 0; attempt < cloudConnectorRetries; attempt++ {
		log.Debugf("Attempt %d of %d", attempt+1, cloudConnectorRetries)
		cloudConnectorResponse, err = makePostCall(eventPayloadBytes, url)
		if err != nil {
			log.Warn("Retrying...")
			time.Sleep(time.Duration(config.AppConfig.CloudConnectorRetrySeconds) * time.Second)
		}
		if err == nil {
			break
		}
	}
	if err != nil {
		mGetErr.Update(1)
		// before return, need to post alert message to rfid-alert service
		// about processed event failed to send to the cloud connector
		go func() {
			sendEventFailedAlertMessage := new(alert.MessagePayload)
			mSendEventFailed.Add(1)
			if postErr := sendEventFailedAlertMessage.SendEventPostFailedAlertMessage(url); postErr != nil {
				log.WithFields(log.Fields{
					"Method": "TriggerCloudConnector",
					"Action": "SendEventPostFailedAlertMessage",
					"Error":  fmt.Errorf("postErr: %s", postErr.Error()),
				}).Error(postErr)
			}
		}()

		if cloudConnectorResponse != nil && cloudConnectorResponse.StatusCode != http.StatusOK {
			responseData, readErr := ioutil.ReadAll(cloudConnectorResponse.Body)
			if readErr != nil {
				mGetErr.Update(1)
				return errors.Wrapf(readErr, "unable to ReadALL response.Body for makePostCall")
			}
			return errors.Wrapf(err, "StatusCode %d , Response %s",
				cloudConnectorResponse.StatusCode, string(responseData))
		}

		return errors.Wrapf(err, "unable to make http POST request")
	}

	// when err == nil, the status code is http.StatusOk (200)
	log.Info("triggerCloudConnector success")
	mTagEventSent.Add(int64(len(tagData)))
	return nil
}

func makePostCall(dataBytes []byte, destination string) (*http.Response, error) {
	log.Debugf("Making POST call to: %s", destination)
	// Metrics
	metrics.GetOrRegisterMeter(`InventoryService.makePostCall.Attempt`, nil).Mark(1)
	mSuccess := metrics.GetOrRegisterGauge(`InventoryService.makePostCall.Success`, nil)
	mGetErr := metrics.GetOrRegisterGauge(`InventoryService.makePostCall.makePostCall-Error`, nil)
	mStatusErr := metrics.GetOrRegisterGauge(`InventoryService.makePostCall.requestStatusCode-Error`, nil)
	mGetLatency := metrics.GetOrRegisterTimer(`InventoryService.makePostCall.makePostCall-Latency`, nil)

	timeout := time.Duration(config.AppConfig.EndpointConnectionTimedOutSeconds) * time.Second
	client := &http.Client{
		Timeout: timeout,
	}
	request, err := http.NewRequest("POST", destination, bytes.NewBuffer(dataBytes))
	if err != nil {
		mGetErr.Update(1)
		log.WithFields(log.Fields{
			"Method":  "makePOSTCall",
			"Action":  "Make New HTTP POST request",
			"Error":   err.Error(),
			"Payload": string(dataBytes[:]),
		}).Error(err)
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	getTimer := time.Now()
	response, err := client.Do(request)
	if err != nil {
		mGetErr.Update(1)
		log.WithFields(log.Fields{
			"Method":  "makePOSTCall",
			"Action":  "Make New HTTP POST request",
			"Error":   err.Error(),
			"Payload": string(dataBytes[:]),
		}).Error(err)
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		mStatusErr.Update(1)
		errMsg := fmt.Errorf("Response code: %d from POST URL %s", response.StatusCode, destination)
		log.WithFields(log.Fields{
			"Method":  "makePOSTCall",
			"Action":  "Response code: " + strconv.Itoa(response.StatusCode),
			"Error":   errMsg,
			"Payload": string(dataBytes[:]),
		}).Error(err)
		return response, errMsg
	}
	mGetLatency.UpdateSince(getTimer)
	mSuccess.Update(1)

	return response, nil
}
