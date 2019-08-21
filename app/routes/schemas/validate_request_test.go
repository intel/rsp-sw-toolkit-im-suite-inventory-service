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

package schemas

import (
	"encoding/json"
	"strings"
	"testing"
)

//nolint[: dupl[, cyclo, ...]]
func TestValidateCurrentInventoryRequest(t *testing.T) {
	requestJSON := []byte(`{
		"qualified_state":"sold",
		"facility_id":"store001",
		"epc_state":"sold",
		"starttime":1482624000000,
		"endtime":1483228800000,
		"confidence":0.75,
		"cursor":"aGksIDovMSB0aGlz",
		"size":500,
		"count_only":true
	  }`)
	result, err := ValidateSchemaRequest(requestJSON, GetCurrentInventorySchema)
	if err != nil {
		t.Errorf("Error validating the json schema %s", err)
	}
	if !result.Valid() {
		t.Errorf("Validation of Json schema failed %s", result.Errors())
	}

	invalidRequest := []byte(`{
		"qualified_state":"sold"
	  }`)
	result, err = ValidateSchemaRequest(invalidRequest, GetCurrentInventorySchema)
	if err != nil {
		t.Errorf("Error validating the json schema %s", err)
	}
	if result.Valid() {
		t.Fatal("Failed to catch json schema validation error, required field 'facility_id'")
	}

	expectedString := `{
		"errors": [
			 {
					"field": "facility_id",
					"errortype": "required",
					"value": {
						 "qualified_state": "sold"
					},
					"description": "facility_id is required"
			 }
		]
 }`

	data, _ := json.MarshalIndent(BuildErrorsString(result.Errors()), "", "   ")
	actualString := string(data)
	act := strings.Replace(actualString, " ", "", -1)
	exp := strings.Replace(expectedString, " ", "", -1)
	exp = strings.Replace(exp, "\t", "", -1)
	if exp != act {
		t.Errorf("Expected string is %v but got %v", expectedString, actualString)
	}

	invalidRequest = []byte(`{
		"qualified_state":"sold",
		"facility_id":"store001",
		"test":10,
		"start":3948309,
		"count_only":true
	  }`)
	result, err = ValidateSchemaRequest(invalidRequest, GetCurrentInventorySchema)
	if err != nil {
		t.Errorf("Error validating the json schema %s", err)
	}
	if result.Valid() {
		t.Fatal("Failed to catch json schema validation error, additional properties")
	}

	invalidRequest = []byte(`{
		"qualified_state":1234,
		"facility_id":"store001",
		"count_only":true
	  }`)
	result, err = ValidateSchemaRequest(invalidRequest, GetCurrentInventorySchema)
	if err != nil {
		t.Errorf("Error validating the json schema %s", err)
	}
	if result.Valid() {
		t.Fatal("Failed to catch json schema validation error, qualified_state type is incorrect")
	}

	invalidRequest = []byte{}
	_, err = ValidateSchemaRequest(invalidRequest, GetCurrentInventorySchema)
	if err == nil {
		t.Fatal("Failed to catch json schema validation error, request body cannot be empty")
	}
}

//nolint: dupl
func TestValidateMissingTagsRequest(t *testing.T) {
	//nolint: dupl
	requestJSON := []byte(`{
		"facility_id":"store001",
		"time":1483228800000,
		"confidence":0.75,
		"cursor":"aGksIDovMSB0aGlz",
		"size":500,
		"count_only":true
	  }`)
	result, _ := ValidateSchemaRequest(requestJSON, MissingTagsSchema)
	if !result.Valid() {
		t.Errorf("Validation of Json schema failed %s", result.Errors())
	}

	invalidRequest := []byte(`{
		"confidence":0.75
	  }`)
	result, _ = ValidateSchemaRequest(invalidRequest, MissingTagsSchema)
	if result.Valid() {
		t.Fatal("Failed to catch json schema validation error, required field 'facility_id'")
	}

	expectedString := `{
		"errors": [
		   {
			  "field": "time",
			  "errortype": "required",
			  "value": {
				 "confidence": 0.75
			  },
			  "description": "time is required"
		   },
		   {
			  "field": "facility_id",
			  "errortype": "required",
			  "value": {
				 "confidence": 0.75
			  },
			  "description": "facility_id is required"
		   }
		]
	 }`

	data, _ := json.MarshalIndent(BuildErrorsString(result.Errors()), "", "   ")
	actualString := string(data)
	act := strings.Replace(actualString, " ", "", -1)
	exp := strings.Replace(expectedString, " ", "", -1)
	exp = strings.Replace(exp, "\t", "", -1)
	if exp != act {
		t.Errorf("Expected string is %v but got %v", expectedString, actualString)
	}

	invalidRequest = []byte(`{
		"facility_id":"store001",
		"time":1483228800000,
		"test":10,
		"start":3948309,
		"count_only":true
	  }`)
	result, _ = ValidateSchemaRequest(invalidRequest, MissingTagsSchema)
	if result.Valid() {
		t.Fatal("Failed to catch json schema validation error, additional properties")
	}
}
