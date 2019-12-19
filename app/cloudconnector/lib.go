/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package cloudconnector

import (
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/app/cloudconnector/event"
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/app/config"
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/app/tag"
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/pkg/jsonrpc"
	"github.com/intel/rsp-sw-toolkit-im-suite-utilities/helper"
)

func SendEvent(invEvent *jsonrpc.InventoryEvent, tagData []tag.Tag) error {
	// todo: do we need to handle splitting the data into segments of 250 max?
	payload := event.DataPayload{
		SentOn:             helper.UnixMilliNow(),
		ControllerId:       invEvent.Params.ControllerId,
		EventSegmentNumber: 1,
		TotalEventSegments: 1,
		TagEvent:           tagData,
	}
	triggerCloudConnectorEndpoint := config.AppConfig.CloudConnectorUrl + config.AppConfig.CloudConnectorApiGatewayEndpoint

	return event.TriggerCloudConnector(invEvent.Params.ControllerId, payload.SentOn, payload.TotalEventSegments, payload.EventSegmentNumber, tagData, triggerCloudConnectorEndpoint)
}
