package tagprocessor

type MobilityProfile struct {
	Id string `json:"id"`
	// using general slope forumla y = m(x) + b
	// where m is slope in dBm per millisecond
	M float64 `json:"m"`
	// dBm change threshold
	T float64 `json:"t"`
	// milliseconds of holdoff
	A float64 `json:"a"`
	// find b such that at 60 seconds, y = 3.0
	// b = y - (m*x)
	B float64 `json:"b"`
}

/*
  "id": "asset_tracking_default",
  "a": 0.0,
  "m": -.008,
  "t": 6.0

  "id": "retail_garment_default",
  "a": 60000.0,
  "m": -.0005,
  "t": 6.0
*/
func NewMobilityProfile() MobilityProfile {
	profile := MobilityProfile{
		Id: "asset_tracking_default",
		M:  -0.008,
		T:  6.0,
		A:  0.0,
	}
	profile.calcB()
	return profile
}

// find b such that at 60 seconds, y = 3.0
// b = y - (m*x)
func (profile *MobilityProfile) calcB() {
	profile.B = profile.T - (profile.M * profile.A)
}
