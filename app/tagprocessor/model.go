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

type Waypoint struct {
	DeviceId  string
	Timestamp int64
}

type TagHistory struct {
	Waypoints []Waypoint
	MaxSize   int
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
