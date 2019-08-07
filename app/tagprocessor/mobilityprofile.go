package tagprocessor

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	assetTrackingDefault = MobilityProfile{
		Id:            "asset_tracking_default",
		Slope:         -0.008,
		Threshold:     6.0,
		HoldoffMillis: 0.0,
	}

	retailGarmentDefault = MobilityProfile{
		Id:            "retail_garment_default",
		Slope:         -0.0005,
		Threshold:     6.0,
		HoldoffMillis: 60000.0,
	}

	defaultProfile = MobilityProfile{
		Id:            "default",
		Slope:         assetTrackingDefault.Slope,
		Threshold:     assetTrackingDefault.Threshold,
		HoldoffMillis: assetTrackingDefault.HoldoffMillis,
	}

	mobilityProfiles = map[string]MobilityProfile{
		assetTrackingDefault.Id: assetTrackingDefault,
		retailGarmentDefault.Id: retailGarmentDefault,
		defaultProfile.Id:       defaultProfile,
	}
)

type MobilityProfile struct {
	Id string `json:"id"`
	// Slope (dBm per millisecond): Used to determine the weight applied to older RSSI values
	Slope float64 `json:"m"`
	// Threshold (dBm) RSSI threshold that must be exceeded for the tag to move from the previous sensor
	Threshold float64 `json:"t"`
	// HoldoffMillis (milliseconds) Amount of time in which the weight used is just the threshold, effectively the slope is not used
	HoldoffMillis float64 `json:"a"`
	// b = y - (m*x)
	YIntercept float64 `json:"b"`
}

// b = y - (m*x)
func (profile *MobilityProfile) calculateYIntercept() {
	profile.YIntercept = profile.Threshold - (profile.Slope * profile.HoldoffMillis)
}

func GetDefaultMobilityProfile() MobilityProfile {
	profile, err := GetMobilityProfile(defaultProfile.Id)

	// default should always exist
	if err != nil {
		err = errors.Wrapf(err, "default mobility profile with id %s does not exist!", defaultProfile.Id)
		logrus.Error(err)
		panic(err)
	}

	return profile
}

func GetMobilityProfile(id string) (MobilityProfile, error) {
	profile, ok := mobilityProfiles[id]
	if !ok {
		return MobilityProfile{}, fmt.Errorf("unable to find mobility profile with id: %s", id)
	}

	// check if y-intercept has been computed yet
	if profile.YIntercept == 0 {
		profile.calculateYIntercept()
		mobilityProfiles[profile.Id] = profile
	}

	return profile, nil
}
