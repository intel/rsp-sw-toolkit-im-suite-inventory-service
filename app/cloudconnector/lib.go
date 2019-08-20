package cloudconnector

import (
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/cloudconnector/event"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/tag"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/jsonrpc"
)

func SendEvent(invEvent *jsonrpc.InventoryEvent, tagData []tag.Tag) error {
	// todo: what else to put in here? seems like the data structure DataPayload  is based on a SAF bus artifact??
	payload := event.DataPayload{
		TagEvent: tagData,
	}
	triggerCloudConnectorEndpoint := config.AppConfig.CloudConnectorUrl + config.AppConfig.CloudConnectorApiGatewayEndpoint

	return event.TriggerCloudConnector(invEvent.Params.ControllerId, payload.SentOn, payload.TotalEventSegments, payload.EventSegmentNumber, tagData, triggerCloudConnectorEndpoint)
}
