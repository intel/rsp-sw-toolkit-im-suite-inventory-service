package jsonrpc

import (
	"encoding/json"
	"github.com/pkg/errors"
)

const (
	NoPersonality = "NONE"
)

type SensorBasicInfo struct {
	DeviceId        string          `json:"device_id"`
	ConnectionState string          `json:"connection_state"`
	ReadState       string          `json:"read_state"`
	BehaviorId      string          `json:"behavior_id"`
	FacilityId      string          `json:"facility_id"`
	Personality     string          `json:"personality"`
	Aliases         []string        `json:"aliases"`
	Alerts          json.RawMessage `json:"alerts"`
}

func (info *SensorBasicInfo) Validate() error {
	if info.DeviceId == "" {
		return errors.New("missing device_id field")
	}
	if info.FacilityId == "" {
		return errors.New("missing facility_id field")
	}
	if info.Personality == "" {
		info.Personality = NoPersonality
	}

	return nil
}
