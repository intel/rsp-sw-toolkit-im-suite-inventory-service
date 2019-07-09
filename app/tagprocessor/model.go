package tagprocessor

const (
	defaultWindowSize = 20
)

type TagState string

const (
	Unknown      TagState = "Unknown"
	Present      TagState = "Present"
	Exiting      TagState = "Exiting"
	DepartedExit TagState = "DepartedExit"
	DepartedPos  TagState = "DepartedPos"
)

type TagDirection string

const (
	Stationary TagDirection = "Stationary"
	Toward     TagDirection = "Toward"
	Away       TagDirection = "Away"
)

type Personality string

const (
	NoPersonality Personality = "None"
	Exit          Personality = "Exit"
	POS           Personality = "POS"
	FittingRoom   Personality = "FittingRoom"
)

type TagEvent string

const (
	NoEvent    TagEvent = "none"
	Arrival    TagEvent = "arrival"
	Moved      TagEvent = "moved"
	Departed   TagEvent = "departed"
	Returned   TagEvent = "returned"
	CycleCount TagEvent = "cycle_count"
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

type rssiAdjuster struct {
	mobilityProfile MobilityProfile
	//todo
}

func NewRssiAdjuster() rssiAdjuster {
	return rssiAdjuster{
		mobilityProfile: NewMobilityProfile(),
	}
}

type Waypoint struct {
	DeviceId  string
	Timestamp int64
}

type TagHistory struct {
	Waypoints []Waypoint
	MaxSize   int
}

type TagStats struct {
	LastRead     int64
	readInterval *CircularBuffer
	rssiMw       *CircularBuffer
	// todo: implement
}

func NewTagStats() TagStats {
	return TagStats{
		readInterval: NewCircularBuffer(defaultWindowSize),
		rssiMw:       NewCircularBuffer(defaultWindowSize),
	}
}

type Tag struct {
	Epc string
	Tid string

	Location       string
	DeviceLocation string
	FacilityId     string

	LastRead     int64
	LastDeparted int64
	LastArrived  int64

	state     TagState
	Direction TagDirection
	History   []TagHistory

	deviceStatsMap map[string]TagStats // todo: TreeMap??
}

func NewTag(epc string) *Tag {
	return &Tag{
		Location:       unknown,
		FacilityId:     unknown,
		DeviceLocation: unknown,
		Direction:      Stationary,
		state:          Unknown,
		deviceStatsMap: make(map[string]TagStats),
		Epc:            epc,
	}
}

type previousTag struct {
	location       string
	deviceLocation string
	facilityId     string
	lastRead       int64
	lastDeparted   int64
	lastArrived    int64
	state          TagState
	direction      TagDirection
}

type TagRead struct {
	Epc        string `json:"epc"`
	Tid        string `json:"tid"`
	AntennaId  int    `json:"antenna_id"`
	LastReadOn int64  `json:"last_read_on"`
	Rssi       int    `json:"rssi"`
	Phase      int    `json:"phase"`
	Frequency  int    `json:"frequency"`
}

type gpsLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude"`
}

type PeriodicInventoryData struct {
	SentOn         int64       `json:"sent_on"`
	Period         int         `json:"period"`
	DeviceId       string      `json:"device_id"`
	Location       gpsLocation `json:"location"`
	FacilityId     string      `json:"facility_id"`
	MotionDetected bool        `json:"motion_detected"`
	Data           []TagRead   `json:"data"`
}

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

type CircularBuffer struct {
	windowSize int
	values     []float64
	counter    int
}

func NewCircularBuffer(windowSize int) *CircularBuffer {
	//logrus.Debugf("create: windowSize: %d", windowSize)
	return &CircularBuffer{
		windowSize: windowSize,
		values:     make([]float64, windowSize),
	}
}
