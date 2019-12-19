/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package schemas

import (
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/pkg/web"

	"github.com/intel/rsp-sw-toolkit-im-suite-gojsonschema"
	"github.com/pkg/errors"
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
