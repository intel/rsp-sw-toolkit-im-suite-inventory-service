package sensor

import (
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/jsonrpc"
	"strconv"
)

const (
	DefaultFacility = "DEFAULT_FACILITY"
)

type Personality string

const (
	NoPersonality Personality = "NONE"
	Exit          Personality = "EXIT"
	POS           Personality = "POS"
	FittingRoom   Personality = "FITTING_ROOM"
)

type RSP struct {
	DeviceId     string      `json:"device_id" db:"device_id"`
	FacilityId   string      `json:"facility_id" db:"facility_id"`
	Personality  Personality `json:"personality" db:"personality"`
	Aliases      []string    `json:"aliases" db:"aliases"`
	IsInDeepScan bool        `json:"-" bson:"-"`
}

func NewRSP(deviceId string) *RSP {
	rsp := RSP{
		DeviceId:    deviceId,
		Personality: NoPersonality,
		FacilityId:  DefaultFacility,
	}
	// setup a default alias for antenna 0
	rsp.Aliases = []string{rsp.AntennaAlias(0)}
	return &rsp
}

func NewRSPFromConfigNotification(notification *jsonrpc.SensorConfigNotification) *RSP {
	cfg := notification.Params
	return &RSP{
		DeviceId:    cfg.DeviceId,
		FacilityId:  cfg.FacilityId,
		Personality: Personality(cfg.Personality),
		Aliases:     cfg.Aliases,
	}
}

// AntennaAlias gets the string alias of an RSP based on the antenna port
// format is DeviceId-AntennaId,  ie. RSP-150009-0
func (rsp *RSP) AntennaAlias(antennaId int) string {
	return rsp.DeviceId + "-" + strconv.Itoa(antennaId)
}

// IsExitSensor returns true if this RSP has the EXIT personality
func (rsp *RSP) IsExitSensor() bool {
	return rsp.Personality == Exit
}

// IsPOSSensor returns true if this RSP has the POS personality
func (rsp *RSP) IsPOSSensor() bool {
	return rsp.Personality == POS
}
