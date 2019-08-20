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

package event

import (
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/tag"
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
