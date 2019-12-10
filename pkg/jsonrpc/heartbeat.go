/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package jsonrpc

import "errors"

type Heartbeat struct {
	Notification                 // embed
	Params       HeartbeatParams `json:"params"`
}

type HeartbeatParams struct {
	SentOn   int64  `json:"sent_on"`
	DeviceId string `json:"device_id"`
}

func (hb *Heartbeat) Validate() error {
	if hb.Params.DeviceId == "" {
		return errors.New("missing device_id field")
	}

	return hb.Notification.Validate()
}
