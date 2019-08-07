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
	DeviceId      string
	FacilityId    string
	Personality   Personality
	Aliases       []string
	IsInDeepScan  bool
	MinRssiDbm10X int
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

func (rsp *RSP) ExitSensor() bool {
	return rsp.Personality == Exit
}

func (rsp *RSP) POSSensor() bool {
	return rsp.Personality == POS
}
