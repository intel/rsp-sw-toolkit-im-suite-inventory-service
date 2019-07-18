package jsonrpc

type InventoryEvent struct {
	Notification // embed
	Params InventoryEventParams `json:"params"`
}

type InventoryEventParams struct {
	SentOn    int64      `json:"sent_on"`
	GatewayId string     `json:"gateway_id,omitempty"` // ok to be empty for handheld
	Data      []TagEvent `json:"data"`
}

// TagEvent is the model of the tag event received from gateway
type TagEvent struct {
	EpcCode         string `json:"epc_code"`
	Tid             string `json:"tid"`
	EpcEncodeFormat string `json:"epc_encode_format"`
	FacilityID      string `json:"facility_id"`
	Location        string `json:"location"`
	EventType       string `json:"event_type,omitempty"`
	Timestamp       int64  `json:"timestamp"`
}
