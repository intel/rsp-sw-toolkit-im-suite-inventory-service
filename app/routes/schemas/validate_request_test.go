/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package schemas

import (
	"encoding/json"
	"strings"
	"testing"
)

//nolint[: dupl[, cyclo, ...]]
func TestValidatePostCurrentInventoryRequest(t *testing.T) {
	requestJSON := []byte(`{
		"qualified_state":"sold",
		"facility_id": "store001",
		"epc_state":"sold",
		"starttime":1482624000000,
		"endtime":1483228800000
	  }`)
	result, err := ValidateSchemaRequest(requestJSON, PostCurrentInventorySchema)
	if err != nil {
		t.Errorf("Error validating the json schema %s", err)
	}
	if !result.Valid() {
		t.Errorf("Validation of Json schema failed %s", result.Errors())
	}

	invalidRequest := []byte(`{
		"test":"sold"
	  }`)
	result, err = ValidateSchemaRequest(invalidRequest, PostCurrentInventorySchema)
	if err != nil {
		t.Errorf("Error validating the json schema %s", err)
	}
	if result.Valid() {
		t.Fatal("Failed to catch json schema validation error, no additional fields allowed'")
	}

	invalidRequest = []byte(`{
		"qualified_state":"sold",
		"facility_id": 1
	  }`)
	result, err = ValidateSchemaRequest(invalidRequest, PostCurrentInventorySchema)
	if err != nil {
		t.Errorf("Error validating the json schema %s", err)
	}
	if result.Valid() {
		t.Fatal("Failed to catch json schema validation error, facility_id type is wrong")
	}
}

//nolint: dupl
func TestValidateDeleteEpcContextRequest(t *testing.T) {
	//nolint: dupl
	requestJSON := []byte(`{
		"facility_id":"store001",
		"epc":"30143639F84191AD22900204"
	  }`)
	result, _ := ValidateSchemaRequest(requestJSON, DeleteEpcContextSchema)
	if !result.Valid() {
		t.Errorf("Validation of Json schema failed %s", result.Errors())
	}

	invalidRequest := []byte(`{
		"facility_id":"store001"
	  }`)
	result, _ = ValidateSchemaRequest(invalidRequest, DeleteEpcContextSchema)
	if result.Valid() {
		t.Fatal("Failed to catch json schema validation error, required field 'epc'")
	}

	expectedString := `{
		"errors": [
		   {
			  "field": "epc",
			  "errortype": "required",
			  "value": {
				"facility_id":"store001"
			  },
			  "description": "epc is required"
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
		"epc":"30143639F84191AD22900204",
		"test":10
	  }`)
	result, _ = ValidateSchemaRequest(invalidRequest, DeleteEpcContextSchema)
	if result.Valid() {
		t.Fatal("Failed to catch json schema validation error, additional properties")
	}
}
