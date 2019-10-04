package jsonrpc

import "github.com/pkg/errors"

type ControllerStatusUpdate struct {
	Notification
	Params ControllerStatusUpdateParams `json:"params"`
}

type ControllerStatusUpdateParams struct {
	DeviceId string `json:"device_id"`
	Status   string `json:"status"`
}

func (notif *ControllerStatusUpdate) Validate() error {
	if notif.Params.DeviceId == "" {
		return errors.New("missing device_id field")
	}
	if notif.Params.Status == "" {
		return errors.New("missing status field")
	}

	return notif.Notification.Validate()
}
