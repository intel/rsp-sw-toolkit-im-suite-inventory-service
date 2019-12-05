package cloudconnector

import (
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/cloudconnector/event"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/tag"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/jsonrpc"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
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
