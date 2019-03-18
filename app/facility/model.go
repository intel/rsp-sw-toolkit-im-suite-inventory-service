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

package facility

// Facility represents a facility model
//swagger:model Facility
type Facility struct {
	// Facility name
	Name string `json:"name"`
	// The coefficients used in the probabilistic inventory algorithm
	Coefficients Coefficients `json:"coefficients"`
}

// CountType represents a wrapper for count and inlinecount
type CountType struct {
	Count *int `json:"count"`
}

// Response is the model used to return the query response
//swagger:model resultsResponse
type Response struct {
	// Array containing results of query
	Results interface{} `json:"results"`
	// Count of records for query
	Count *int `json:"count,omitempty"`
}

// Coefficients represents a set of attributes to calculate confidence
type Coefficients struct {
	// Percent of inventory that is sold daily
	DailyInventoryPercentage float64 `json:"dailyinventorypercentage"`
	// Probability of an unreadable tag becoming readable again each day (i.e. moved or retagged)
	ProbUnreadToRead float64 `json:"probunreadtoread"`
	// Probability of a tag in the store being read by the overhead sensor each day
	ProbInStoreRead float64 `json:"probinstoreread"`
	// Probability of an exit error (missed 'departed' event) occurring
	ProbExitError float64 `json:"probexiterror"`
}

// RequestBody represents a struct for the requestBody to Update facility collection
//swagger:ignore
type RequestBody struct {
	FacilityID               string  `json:"facility_id"`
	DailyInventoryPercentage float64 `json:"dailyinventorypercentage"`
	ProbUnreadToRead         float64 `json:"probunreadtoread"`
	ProbInStoreRead          float64 `json:"probinstoreread"`
	ProbExitError            float64 `json:"probexiterror"`
}
