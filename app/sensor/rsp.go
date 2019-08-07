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
	NoPersonality Personality = "None"
	Exit          Personality = "Exit"
	POS           Personality = "POS"
	FittingRoom   Personality = "FittingRoom"
)

type RSP struct {
	DeviceId      string      `json:"device_id" bson:"device_id"`
	FacilityId    string      `json:"facility_id" bson:"facility_id"`
	Personality   Personality `json:"personality" bson:"personality"`
	Aliases       []string    `json:"aliases" bson:"aliases"`
	IsInDeepScan  bool        `json:"-" bson:"-"`
	MinRssiDbm10X int         `json:"min_rssi_dbm_10x" bson:"min_rssi_dbm_10x"`
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

func (rsp *RSP) UpdateFromConfig(notification jsonrpc.SensorConfigNotification) {
	rsp.FacilityId = notification.Params.FacilityId
	rsp.Personality = Personality(notification.Params.Personality)
	rsp.Aliases = notification.Params.Aliases
}

func (rsp *RSP) AntennaAlias(antennaId int) string {
	var sb strings.Builder
	sb.WriteString(rsp.DeviceId)
	sb.WriteString("-")
	sb.WriteString(strconv.Itoa(antennaId))
	return sb.String()
}

func (rsp *RSP) RssiInRange(rssi int) bool {
	return rsp.MinRssiDbm10X == 0 || rssi >= rsp.MinRssiDbm10X
}

func (rsp *RSP) IsExitSensor() bool {
	return rsp.Personality == Exit
}

func (rsp *RSP) IsPOSSensor() bool {
	return rsp.Personality == POS
}

// Empty returns whether this RSP has data or not.
// all RSPs require a deviceId, so simply check for that field
func (rsp *RSP) IsEmpty() bool {
	return rsp.DeviceId == ""
}
