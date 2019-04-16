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
	"bytes"
	"encoding/json"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
)

// RRSAlert deals with RRS type of alerts
type RRSAlert []byte

// NewRRSAlert instantiates RRSAlert object for processing RRS type of alerts
func NewRRSAlert(jsonBytes []byte) RRSAlert {
	return RRSAlert(jsonBytes)
}

// ProcessAlert is to process the RRS Alert JSON payload
func (rrsAlert RRSAlert) ProcessAlert() error {
	mRRSAlertReceived := metrics.GetOrRegisterGaugeCollection("Inventory.ProcessRRSAlert.RRSAlertReceived", nil)
	mRRSAlertDetails := metrics.GetOrRegisterGaugeCollection("Inventory.ProcessRRSAlert.RRSAlertDetails", nil)

	mRRSAlertReceived.Add(1) // Have to let metrics system know an alert occurred even if we can't provide details below

	var data map[string]interface{}

	decoder := json.NewDecoder(bytes.NewBuffer(rrsAlert))
	if err := decoder.Decode(&data); err != nil {
		return errors.Wrap(err, "unable to Decode data")
	}

	deviceID, ok := data["device_id"].(string)
	if !ok { //nolint:golint
		return errors.New("Missing device_id Field")
	}

	alertNumber, ok := data["alert_number"].(float64)
	if !ok { //nolint:golint
		return errors.New("Missing alert_number Field")
	}

	tag := metrics.Tag{
		Name:  "DeviceId",
		Value: deviceID,
	}
	// Use tag version to flag the Alert with more details
	mRRSAlertDetails.AddWithTag(int64(alertNumber), tag)

	return nil
}

// IsInventoryUnloadAlert parses out the payload JSON bytes and check if the alert number is for INENTORY_UNLOAD,
// which is 260, or not. Return true if it is; false otherwise
func (rrsAlert RRSAlert) IsInventoryUnloadAlert() bool {
	// TODO: this is silly; we don't need to deseralize the message twice
	// TODO: just unmarshal it once, into a type that has the relevant info
	log.Infof("alert:\n%s", string(rrsAlert))

	var data map[string]interface{}

	if err := json.Unmarshal(rrsAlert, &data); err != nil {
		return false
	}

	alertNumFromJSON, ok := data["alert_number"].(float64)
	if !ok {
		return false
	}

	alertNumber := int(alertNumFromJSON)

	return alertNumber == InventoryUnload
}
