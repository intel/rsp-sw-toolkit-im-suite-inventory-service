/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package event

import (
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/app/tag"
	"net/http"
)

// Payload for the cloudconnector
type TagEventPayload struct {
	URL     string      `json:"url"`
	Headers http.Header `json:"header"`
	Body    DataPayload `json:"payload"`
	Auth    Auth        `json:"auth"`
	Method  string      `json:"method"`
	IsAsync bool        `json:"isasync"`
}

type DataPayload struct {
	ControllerId       string    `json:"device_id"` //backend expects this to be "device_id" instead of controller_id
	SentOn             int64     `json:"sent_on"`
	TotalEventSegments int       `json:"total_event_segments"`
	EventSegmentNumber int       `json:"event_segment_number"`
	TagEvent           []tag.Tag `json:"data"`
}

// Auth contains the type and the endpoint of authentication
type Auth struct {
	AuthType string `json:"authtype"`
	Endpoint string `json:"endpoint"`
	Data     string `json:"data"`
}
