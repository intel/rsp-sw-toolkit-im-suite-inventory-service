package tagprocessor

import "github.impcloud.net/RSP-Inventory-Suite/utilities/helper"

type rssiAdjuster struct {
	mobilityProfile MobilityProfile
	//todo
}

func NewRssiAdjuster() rssiAdjuster {
	return rssiAdjuster{
		mobilityProfile: NewMobilityProfile(),
	}
}

func (weighter *rssiAdjuster) getWeight(lastRead int64, sensor *RfidSensor) float64 {
	profile := weighter.mobilityProfile

	if sensor.isInDeepScan {
		return profile.T
	}

	// todo: is it safe to convert int64 to float64?
	weight := (profile.M * float64(helper.UnixMilliNow()-lastRead)) + profile.B

	// check if weight needs to be capped at threshold ceiling
	if weight > profile.T {
		weight = profile.T
	}

	return weight
}
