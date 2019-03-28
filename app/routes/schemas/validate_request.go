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
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/web"

	"github.com/pkg/errors"
	"github.impcloud.net/RSP-Inventory-Suite/gojsonschema"
)

// ValidateSchemaRequest validates the api request body with the required json schema
func ValidateSchemaRequest(jsonBody []byte, schema string) (*gojsonschema.Result, error) {
	if len(jsonBody) == 0 {
		return nil, errors.Wrapf(web.ErrInvalidInput, "request body cannot be empty")
	}

	schemaLoader := gojsonschema.NewStringLoader(schema)
	documentLoader := gojsonschema.NewBytesLoader(jsonBody)

	validatorResult, err := gojsonschema.Validate(schemaLoader, documentLoader)

	if err != nil {
		return nil, errors.Wrapf(web.ErrInvalidInput, err.Error())
	}

	return validatorResult, nil
}

// ErrorList provides a collection of errors for processing
//swagger:response schemaValidation
type ErrorList struct {
	// The error list
	//in:body
	Errors []ErrReport `json:"errors"`
}

//ErrReport is used to wrap schema validation errors int json object
type ErrReport struct {
	Field       string      `json:"field"`
	ErrorType   string      `json:"errortype"`
	Value       interface{} `json:"value"`
	Description string      `json:"description"`
}

// BuildErrorsString concatenates errors and builds pretty error strings
func BuildErrorsString(resultsErrors []gojsonschema.ResultError) interface{} {

	var error ErrReport
	var errorSlice []ErrReport
	var errors ErrorList

	for _, err := range resultsErrors {

		// err.Field() is not set for "required" error
		var field string
		if property, ok := err.Details()["property"].(string); ok {
			field = property
		} else {
			field = err.Field()
		}

		// ignore extraneous "number_one_of" error
		if err.Type() == "number_one_of" {
			continue
		}
		error.Field = field
		error.Description = err.Description()
		error.ErrorType = err.Type()
		error.Value = err.Value()
		errorSlice = append(errorSlice, error)
	}
	errors.Errors = errorSlice

	return errors
}
