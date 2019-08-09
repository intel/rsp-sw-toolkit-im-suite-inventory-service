package jsonrpc

import "github.com/pkg/errors"

type SensorConfigNotification struct {
	Notification
	Params SensorConfigNotificationParams `json:"params"`
}

type SensorConfigNotificationParams struct {
	DeviceId    string   `json:"device_id"`
	FacilityId  string   `json:"facility_id"`
	Personality string   `json:"personality"`
	Aliases     []string `json:"aliases"`
}

func (notif *SensorConfigNotification) Validate() error {
	if notif.Params.DeviceId == "" {
		return errors.New("missing device_id field")
	}
	if notif.Params.FacilityId == "" {
		return errors.New("missing facility_id field")
	}
	// Personality can be empty
	// Aliases can be empty???

	return notif.Notification.Validate()
}
