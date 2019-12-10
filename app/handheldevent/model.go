/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package handheldevent

// HandheldEvent represents a handheld event model
//swagger:model HandheldEvent
type HandheldEvent struct {
	// Can be FullScanStart, FullScanComplete, or Calculate
	Event string `json:"event"`
	// Time event was received in epoch
	// min: 13
	Timestamp int64 `json:"timestamp"`
}

// CountType represents a wrapper for count and inlinecount
type CountType struct {
	Count *int `json:"count"`
}

// Response is the model used to return the query response
type Response struct {
	Results interface{} `json:"results"`
	Count   *int        `json:"count,omitempty"`
}
