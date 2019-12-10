/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package facility

// Facility represents a facility model
//swagger:model Facility
type Facility struct {
	// Facility name
	Name string `json:"name"  db:"name"`
	// The coefficients used in the probabilistic inventory algorithm
	Coefficients Coefficients `json:"coefficients"  db:"coefficients"`
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
	DailyInventoryPercentage float64 `json:"dailyinventorypercentage" db:"dailyinventorypercentage"`
	// Probability of an unreadable tag becoming readable again each day (i.e. moved or retagged)
	ProbUnreadToRead float64 `json:"probunreadtoread" db:"probunreadtoread"`
	// Probability of a tag in the store being read by the overhead sensor each day
	ProbInStoreRead float64 `json:"probinstoreread" db:"probinstoreread"`
	// Probability of an exit error (missed 'departed' event) occurring
	ProbExitError float64 `json:"probexiterror" db:"probexiterror"`
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
