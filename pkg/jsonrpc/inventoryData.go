package jsonrpc

import "github.com/pkg/errors"

type InventoryData struct {
	Notification                     // embed
	Params       InventoryDataParams `json:"params"`
}

type InventoryDataParams struct {
	SentOn         int64       `json:"sent_on"`
	Period         int         `json:"period"`
	DeviceId       string      `json:"device_id"`
	Location       GpsLocation `json:"location"`
	FacilityId     string      `json:"facility_id"`
	MotionDetected bool        `json:"motion_detected"`
	Data           []TagRead   `json:"data"`
}

type TagRead struct {
	Epc        string `json:"epc"`
	Tid        string `json:"tid"`
	AntennaId  int    `json:"antenna_id"`
	LastReadOn int64  `json:"last_read_on"`
	Rssi       int    `json:"rssi"`
	Phase      int    `json:"phase"`
	Frequency  int    `json:"frequency"`
}

type GpsLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude"`
}

func (data *InventoryData) Validate() error {
	if data.Params.DeviceId == "" {
		return errors.New("missing device_id field")
	}
	if data.Params.Data == nil || len(data.Params.Data) == 0 {
		return errors.New("missing data field")
	}

	return data.Notification.Validate()
}
