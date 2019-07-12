package tagprocessor

import (
	"strconv"
	"strings"
)

type RfidSensor struct {
	DeviceId      string
	FacilityId    string
	Personality   Personality
	isInDeepScan  bool
	minRssiDbm10X int
}

func NewRfidSensor(deviceId string) *RfidSensor {
	return &RfidSensor{
		DeviceId:    deviceId,
		Personality: NoPersonality,
		FacilityId:  unknown,
	}
}

func (sensor *RfidSensor) getAntennaAlias(antennaId int) string {
	var sb strings.Builder
	sb.WriteString(sensor.DeviceId)
	sb.WriteString("-")
	sb.WriteString(strconv.Itoa(antennaId))
	return sb.String()
}
