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

import "testing"

func TestProcessRRSAlertOk(t *testing.T) {
	JSONSample := []byte(`{
		"macaddress": "02:42:ac:1d:00:04",
		"application": "rsp_collector",
		"providerId": -1,
		"dateTime": "2017-08-25T22:29:23.816Z",
		"type": "urn:x-intel:context:retailsensingplatform:alerts",
		"value": {
		  "device_id": "RSP123",
		  "facilities": [
			"Tavern"
		  ],
		  "alert_number": 156,
		  "alert_description": "Test Alert",
		  "severity": "high",
		  "sent_on": 1503700192960
		}
	  }`)

	rrsAlert := NewRRSAlert(JSONSample)

	if err := rrsAlert.ProcessAlert(); err != nil {
		t.Fatalf("error processing alert data %s", err.Error())
	}

	if string(rrsAlert) != string(JSONSample) {
		t.Fatalf("alert payload not set correctly: %s", string(rrsAlert))
	}
}

func TestIsInventoryUnloadAlertOk(t *testing.T) {
	resetBaselineAlertJSON := []byte(`{
		"macaddress": "02:42:ac:1d:00:04",
		"application": "rsp_collector",
		"providerId": -1,
		"dateTime": "2017-08-25T22:29:23.816Z",
		"type": "urn:x-intel:context:retailsensingplatform:alerts",
		"value": {
		  "device_id": "RSP123",
		  "facilities": [
			"Tavern"
		  ],
		  "alert_number": 260,
		  "alert_description": "Reset baseline Alert",
		  "severity": "critical",
		  "sent_on": 1503700192960
		}
		}`)
	rrsAlert := NewRRSAlert(resetBaselineAlertJSON)
	resetBaselineAlert := rrsAlert.IsInventoryUnloadAlert()
	if !resetBaselineAlert {
		t.Fatal("expecting returning true but found false")
	}

	otherAlertJSON := []byte(`{
		"macaddress": "02:42:ac:1d:00:04",
		"application": "rsp_collector",
		"providerId": -1,
		"dateTime": "2017-08-25T22:29:23.816Z",
		"type": "urn:x-intel:context:retailsensingplatform:alerts",
		"value": {
		  "device_id": "RSP123",
		  "facilities": [
			"Tavern"
		  ],
		  "alert_number": 241,
		  "alert_description": "Gateway shutdown Alert",
		  "severity": "critical",
		  "sent_on": 1503700192960
		}
		}`)

	rrsAlert = NewRRSAlert(otherAlertJSON)
	resetBaselineAlert = rrsAlert.IsInventoryUnloadAlert()
	if resetBaselineAlert {
		t.Fatal("expecting returning false for reset baseline alert but found true")
	}
}

func TestProcessRRSAlertBadInputJson(t *testing.T) {
	missingValue := []byte(`{
		"macaddress": "02:42:ac:1d:00:04",
		"application": "rsp_collector",
		"providerId": -1,
		"dateTime": "2017-08-25T22:29:23.816Z",
		"type": "urn:x-intel:context:retailsensingplatform:alerts"
	  }`)

	rrsAlert := NewRRSAlert(missingValue)

	if err := rrsAlert.ProcessAlert(); err == nil {
		t.Fatal("expecting error since missing value field in JSON ")
	}

	missingDeviceID := []byte(`{
		"macaddress": "02:42:ac:1d:00:04",
		"application": "rsp_collector",
		"providerId": -1,
		"dateTime": "2017-08-25T22:29:23.816Z",
		"type": "urn:x-intel:context:retailsensingplatform:alerts",
		"value": {
		  "facilities": [
			"Tavern"
		  ],
		  "alert_number": 156,
		  "alert_description": "Test Alert",
		  "severity": "high",
		  "sent_on": 1503700192960
		}
	  }`)

	rrsAlert = NewRRSAlert(missingDeviceID)

	if err := rrsAlert.ProcessAlert(); err == nil {
		t.Fatal("expecting error since missing device_id field in JSON ")
	}

	missingAlertNumber := []byte(`{
		"macaddress": "02:42:ac:1d:00:04",
		"application": "rsp_collector",
		"providerId": -1,
		"dateTime": "2017-08-25T22:29:23.816Z",
		"type": "urn:x-intel:context:retailsensingplatform:alerts",
		"value": {
			"device_id": "RSP123",
		  "facilities": [
			"Tavern"
		  ],
		  "alert_description": "Test Alert",
		  "severity": "high",
		  "sent_on": 1503700192960
		}
	  }`)

	rrsAlert = NewRRSAlert(missingAlertNumber)

	if err := rrsAlert.ProcessAlert(); err == nil {
		t.Fatal("expecting error since missing alert_number field in JSON ")
	}

	junkInput := []byte("junk data")
	rrsAlert = NewRRSAlert(junkInput)

	if err := rrsAlert.ProcessAlert(); err == nil {
		t.Fatal("expecting error since input JSON is corrupted")
	}
}

func TestIsInventoryUnloadAlertBadInputs(t *testing.T) {
	missingValue := []byte(`{
		"macaddress": "02:42:ac:1d:00:04",
		"application": "rsp_collector",
		"providerId": -1,
		"dateTime": "2017-08-25T22:29:23.816Z",
		"type": "urn:x-intel:context:retailsensingplatform:alerts"
	  }`)
	rrsAlert := NewRRSAlert(missingValue)
	resetBaselineAlert := rrsAlert.IsInventoryUnloadAlert()
	if resetBaselineAlert {
		t.Fatal("expecting returning false since value field is missing")
	}

	missingAlertNumber := []byte(`{
		"macaddress": "02:42:ac:1d:00:04",
		"application": "rsp_collector",
		"providerId": -1,
		"dateTime": "2017-08-25T22:29:23.816Z",
		"type": "urn:x-intel:context:retailsensingplatform:alerts",
		"value": {
			"device_id": "RSP123",
		  "facilities": [
			"Tavern"
		  ],
		  "alert_description": "Test Alert",
		  "severity": "high",
		  "sent_on": 1503700192960
		}
		}`)

	rrsAlert = NewRRSAlert(missingAlertNumber)
	resetBaselineAlert = rrsAlert.IsInventoryUnloadAlert()
	if resetBaselineAlert {
		t.Fatal("expecting returning false since alert_number field is missing")
	}

	junkInput := []byte("junk data")
	rrsAlert = NewRRSAlert(junkInput)

	resetBaselineAlert = rrsAlert.IsInventoryUnloadAlert()
	if resetBaselineAlert {
		t.Fatal("expecting returning false since the input is corrupted")
	}
}
