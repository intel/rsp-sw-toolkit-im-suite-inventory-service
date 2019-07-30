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
