/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
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
