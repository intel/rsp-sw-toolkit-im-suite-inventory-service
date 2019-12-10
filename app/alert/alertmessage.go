/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package alert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
)

const (
	jsonApplication = "application/json;charset=utf-8"
)

// Alert value for cloud which does not include controller_id
type Alert struct {
	SentOn      int64       `json:"sent_on"`
	Number      int         `json:"alert_number"`
	Description string      `json:"alert_description"`
	Severity    string      `json:"severity"`
	Optional    interface{} `json:"optional"`
}

// MessagePayload is the json data to alertmessage endpoint of RFID-alert-service
type MessagePayload struct {
	Application string `json:"application"`
	Value       Alert  `json:"value"`
}

// generateDeleteTagCollectionDoneMessage is to generate the payload for completion of deleting tag collection in mongo db
// returns byte slice of the JSON MessagePayload
func (payload *MessagePayload) generateDeleteTagCollectionDoneMessage() ([]byte, error) {
	payload.Application = config.AppConfig.ServiceName
	payload.Value = Alert{
		SentOn:      helper.UnixMilliNow(),
		Number:      InventoryUnload,
		Description: "Deletion of inventory DB tag collection is done",
		Severity:    "info",
		Optional:    "",
	}

	alertMessageBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "Error on marshaling AlertMessage to []bytes")
	}
	return alertMessageBytes, nil
}

// SendDeleteTagCompletionAlertMessage sends alertmessage POST restful API call to RFID alert service
// for completion of deleting tag collection in mongo db
func (payload *MessagePayload) SendDeleteTagCompletionAlertMessage() error {
	payloadBytes, err := payload.generateDeleteTagCollectionDoneMessage()
	if err != nil {
		return err
	}

	postErr := postAlertMessageService(payloadBytes)
	log.Debug("SendDeleteTagCompletionAlertMessage posted")
	return postErr
}

// generateSendEventFailedAlertMessage is to generate the payload for alert on failing to send event to cloud connector
// returns byte slice of the JSON MessagePayload
func (payload *MessagePayload) generateSendEventFailedAlertMessage(cloudConnectorPostURL string) ([]byte, error) {
	payload.Application = config.AppConfig.ServiceName
	payload.Value = Alert{
		SentOn:      helper.UnixMilliNow(),
		Number:      SendEventFailed,
		Description: "Unable to send the processed event to the cloud connector",
		Severity:    "critical",
		Optional:    fmt.Sprintf("cloudConnectorPostURL: %s", cloudConnectorPostURL),
	}

	alertMessageBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "Error on marshaling AlertMessage to []bytes")
	}
	return alertMessageBytes, nil
}

// SendEventPostFailedAlertMessage sends alertmessage POST restful API call to RFID alert service
// for failures on posting events to cloud connector service
func (payload *MessagePayload) SendEventPostFailedAlertMessage(cloudConnectorPostURL string) error {
	payloadBytes, err := payload.generateSendEventFailedAlertMessage(cloudConnectorPostURL)
	if err != nil {
		return err
	}

	postErr := postAlertMessageService(payloadBytes)
	log.Debug("SendEventPostFailedAlertMessage posted")
	return postErr
}

func postAlertMessageService(payloadBytes []byte) error {
	// call the rfid alert endpoint to signal the deletion is done
	timeout := time.Duration(config.AppConfig.EndpointConnectionTimedOutSeconds) * time.Second
	client := &http.Client{
		Timeout: timeout,
	}

	alertAPI := config.AppConfig.RfidAlertURL + config.AppConfig.RfidAlertMessageEndpoint

	request, reqErr := http.NewRequest(http.MethodPost, alertAPI, bytes.NewBuffer(payloadBytes))
	if reqErr != nil {
		return reqErr
	}
	request.Header.Set("content-type", jsonApplication)
	response, postErr := client.Do(request)
	if postErr != nil {
		return postErr
	}

	if response.StatusCode != http.StatusOK {
		var respErrData string
		if response.Body != nil {
			respErrByes, respErr := ioutil.ReadAll(response.Body)
			if respErr != nil {
				return errors.Wrapf(respErr, "unable to readall response body")
			}
			respErrData = string(respErrByes)
		}
		return errors.Errorf("failed to post to rfid-alertAPI %s with status code %d  response error data %s",
			alertAPI, response.StatusCode, respErrData)
	}

	log.Debugf("post to alert service %s ok", alertAPI)

	defer func() {
		if err := response.Body.Close(); err != nil {
			log.Errorf("postAlertMessageService response body close error %s", err.Error())
		}
	}()

	return nil
}
