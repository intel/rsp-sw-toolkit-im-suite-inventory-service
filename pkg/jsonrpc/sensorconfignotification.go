package jsonrpc

import "github.com/pkg/errors"

type SensorConfigNotification struct {
	Notification
	Params SensorConfigNotificationParams `json:"params"`
}

type SensorConfigNotificationParams struct {
	DeviceId    string
	FacilityId  string
	Personality string
	Aliases     []string
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
