/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package tagprocessor

import (
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/app/sensor"
	"github.com/intel/rsp-sw-toolkit-im-suite-utilities/helper"
)

type rssiAdjuster struct {
	mobilityProfile MobilityProfile
}

func newRssiAdjuster() rssiAdjuster {
	return rssiAdjuster{
		mobilityProfile: GetActiveMobilityProfile(),
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
