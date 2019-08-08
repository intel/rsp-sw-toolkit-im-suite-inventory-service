package sensor

import (
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/jsonrpc"
	"strconv"
	"strings"
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
	DeviceId      string      `json:"device_id" bson:"device_id"`
	FacilityId    string      `json:"facility_id" bson:"facility_id"`
	Personality   Personality `json:"personality" bson:"personality"`
	Aliases       []string    `json:"aliases" bson:"aliases"`
	IsInDeepScan  bool        `json:"-" bson:"-"`
	MinRssiDbm10X int         `json:"min_rssi,omitempty" bson:"min_rssi,omitempty"`
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
	var sb strings.Builder
	sb.WriteString(rsp.DeviceId)
	sb.WriteString("-")
	sb.WriteString(strconv.Itoa(antennaId))
	return sb.String()
}

// RssiInRange returns true if either the RSP is not filtering by minimum RSSI, or the RSSI is >= the minimum RSSI configured
func (rsp *RSP) RssiInRange(rssi int) bool {
	return rsp.MinRssiDbm10X == 0 || rssi >= rsp.MinRssiDbm10X
}

// IsExitSensor returns true if this RSP has the EXIT personality
func (rsp *RSP) IsExitSensor() bool {
	return rsp.Personality == Exit
}

// IsPOSSensor returns true if this RSP has the POS personality
func (rsp *RSP) IsPOSSensor() bool {
	return rsp.Personality == POS
}

// IsEmpty returns whether this RSP has data or not.
// all RSPs require a deviceId, so simply check for that field
func (rsp *RSP) IsEmpty() bool {
	return rsp.DeviceId == ""
}
