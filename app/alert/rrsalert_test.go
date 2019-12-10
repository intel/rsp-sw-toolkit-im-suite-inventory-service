/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */
package alert

import (
	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/pkg/errors"
	"testing"
)

func TestProcessRRSAlert(t *testing.T) {
	tests := []struct {
		name        string
		expectError bool
		failMessage string
		data        string
	}{
		{
			name:        "basicSuccess",
			expectError: false,
			failMessage: "error processing alert data",
			data: `
			{		
				"device_id": "RSP123",
		  		"facilities": [
					"Tavern"
		  		],
		  		"alert_number": 156,
		  		"alert_description": "Test Alert",
		  		"severity": "high",
		  		"sent_on": 1503700192960
			}`,
		},
		{
			name:        "missingValue",
			expectError: true,
			failMessage: "expecting error since missing value field in JSON ",
			data: `
			{
				"macaddress": "02:42:ac:1d:00:04",
				"application": "rsp_collector",
				"providerId": -1,
				"dateTime": "2017-08-25T22:29:23.816Z",
				"type": "urn:x-intel:context:retailsensingplatform:alerts"
	  		}`,
		},
		{
			name:        "missingDeviceID",
			expectError: true,
			failMessage: "expecting error since missing device_id field in JSON ",
			data: `
			{
		  		"facilities": [
					"Tavern"
		  		],
		  		"alert_number": 156,
		  		"alert_description": "Test Alert",
		  		"severity": "high",
		  		"sent_on": 1503700192960
	  		}`,
		},
		{
			name:        "missingAlertNumber",
			expectError: true,
			failMessage: "expecting error since missing alert_number field in JSON ",
			data: `
			{
				"device_id": "RSP123",
		  		"facilities": [
					"Tavern"
		  		],
		  		"alert_description": "Test Alert",
		  		"severity": "high",
		  		"sent_on": 1503700192960		
	  		}`,
		},
		{
			name:        "junkInput",
			expectError: true,
			failMessage: "expecting error since input JSON is corrupted",
			data:        "junk data",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ProcessAlert(&models.Reading{Value: test.data})

			if test.expectError && err == nil {
				t.Error(test.failMessage)
			} else if !test.expectError && err != nil {
				t.Error(errors.Wrap(err, test.failMessage))
			}
		})
	}
}

func TestIsInventoryUnloadAlert(t *testing.T) {
	tests := []struct {
		name                    string
		expectIsInventoryUnload bool
		failMessage             string
		data                    string
	}{
		{
			name:                    "basicSuccess",
			expectIsInventoryUnload: true,
			failMessage:             "expected json to parse and be decoded as inventory_unload event",
			data: `{
				"device_id": "RSP123",
		  		"facilities": [
					"Tavern"
		  		],
		  		"alert_number": 260,
		  		"alert_description": "Reset baseline Alert",
		  		"severity": "critical",
		  		"sent_on": 1503700192960
			}`,
		},
		{
			name:                    "notInventoryUnload1",
			expectIsInventoryUnload: false,
			failMessage:             "expected json to parse and and not be an inventory_unload event",
			data: `{
		  		"device_id": "RSP123",
		  		"facilities": [
					"Tavern"
		  		],
		  		"alert_number": 241,
		  		"alert_description": "RSP Controller shutdown Alert",
		  		"severity": "critical",
		  		"sent_on": 1503700192960
			}`,
		},
		{
			name:                    "notInventoryUnload2",
			expectIsInventoryUnload: false,
			failMessage:             "expected json to parse and not be an inventory_unload event",
			data: `
			{		
				"device_id": "RSP123",
		  		"facilities": [
					"Tavern"
		  		],
		  		"alert_number": 156,
		  		"alert_description": "Test Alert",
		  		"severity": "high",
		  		"sent_on": 1503700192960
			}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rrsAlert, err := ProcessAlert(&models.Reading{Value: test.data})
			if err != nil {
				t.Error(errors.Wrap(err, test.failMessage))
			}

			if rrsAlert.IsInventoryUnloadAlert() != test.expectIsInventoryUnload {
				t.Error(test.failMessage)
			}
		})
	}
}
