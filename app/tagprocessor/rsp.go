package tagprocessor

import (
	"strconv"
	"strings"
)

type RSP struct {
	DeviceId      string
	FacilityId    string
	Personality   Personality
	isInDeepScan  bool
	minRssiDbm10X int
}

func NewRSP(deviceId string) *RSP {
	return &RSP{
		DeviceId:    deviceId,
		Personality: NoPersonality,
		FacilityId:  unknown,
	}
}

func (sensor *RSP) getAntennaAlias(antennaId int) string {
	var sb strings.Builder
	sb.WriteString(sensor.DeviceId)
	sb.WriteString("-")
	sb.WriteString(strconv.Itoa(antennaId))
	return sb.String()
}
