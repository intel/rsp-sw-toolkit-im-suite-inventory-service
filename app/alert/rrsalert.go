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
	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/pkg/errors"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
	"strings"
)

type RRSAlert struct {
	Alert           // embed
	DeviceId string `json:"device_id"`
}

// ProcessAlert is to process the RRS Alert JSON payload
func ProcessAlert(reading *models.Reading) (RRSAlert, error) {
	mAlertReceived := metrics.GetOrRegisterGaugeCollection("Inventory.ProcessAlert.AlertReceived", nil)
	mAlertDetails := metrics.GetOrRegisterGaugeCollection("Inventory.ProcessAlert.AlertDetails", nil)

	mAlertReceived.Add(1) // Have to let metrics system know an alert occurred even if we can't provide details below

	alert := RRSAlert{}

	decoder := json.NewDecoder(strings.NewReader(reading.Value))
	if err := decoder.Decode(&alert); err != nil {
		return RRSAlert{}, errors.Wrap(err, "unable to Decode data")
	}

	if alert.DeviceId == "" {
		return RRSAlert{}, errors.New("Missing device_id Field")
	}
	if alert.Number == 0 {
		return RRSAlert{}, errors.New("Missing alert_number Field")
	}

	tag := metrics.Tag{
		Name:  "DeviceId",
		Value: alert.DeviceId,
	}
	// Use tag version to flag the Alert with more details
	mAlertDetails.AddWithTag(int64(alert.Number), tag)

	return alert, nil
}

// IsInventoryUnloadAlert parses out the payload JSON bytes and check if the alert number is for INENTORY_UNLOAD,
// which is 260, or not. Return true if it is; false otherwise
func (alert Alert) IsInventoryUnloadAlert() bool {
	return alert.Number == InventoryUnload
}
