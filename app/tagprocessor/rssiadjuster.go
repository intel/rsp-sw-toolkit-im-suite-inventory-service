package tagprocessor

import (
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/sensor"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
)

type rssiAdjuster struct {
	mobilityProfile MobilityProfile
}

func newRssiAdjuster() rssiAdjuster {
	return rssiAdjuster{
		mobilityProfile: GetDefaultMobilityProfile(),
	}
}

func (weighter *rssiAdjuster) getWeight(lastRead int64, rsp *sensor.RSP) float64 {
	profile := weighter.mobilityProfile

	if rsp.IsInDeepScan {
		return profile.Threshold
	}

	weight := (profile.Slope * float64(helper.UnixMilliNow()-lastRead)) + profile.YIntercept

	// check if weight needs to be capped at threshold ceiling
	if weight > profile.Threshold {
		weight = profile.Threshold
	}

	return weight
}
