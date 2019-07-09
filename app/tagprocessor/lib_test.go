package tagprocessor

import "testing"

func TestTagArrival(t *testing.T) {
	ds := newTestDataset(t)
	ds.initialize(3, rssiMin)

	sensorBack01.minRssiDbm10X = -600

	ds.readTag(0, sensorBack01, -580, 1)
	ds.readTag(1, sensorBack01, -620, 1)
	ds.readTag(2, sensorFrontPOS, rssiMin, 1)

	ds.updateTagRefs()

	ds.checkTag(0, Present, sensorBack01)
	// tag1 will NOT arrive due to having an rssi too low
	if ds.tags[1] != nil {
		t.Errorf("expected tag with index 1 to be nil, but was %v", ds.tags[1])
	}
	ds.checkTag(2, Unknown, sensorFrontPOS)

	// todo: check for 1 arrival events
}

func TestExitingArrivalDepartures(t *testing.T) {
	ds := newTestDataset(t)
	ds.initialize(5, rssiMin)

	ds.readAllTags(sensorBack01, rssiMin, 4)

	ds.updateTagRefs()
	ds.checkAllTags(Present, sensorBack01)

	// one tag read by an EXIT will not make the tag go exiting.
	ds.readAllTags(sensorFrontExit, rssiWeak, 1)
	ds.checkAllTags(Present, sensorBack01)

	// go to exiting state
	ds.readAllTags(sensorFrontExit, rssiWeak,10)
	ds.checkAllTags(Exiting, sensorFrontExit)

	// clear exiting by moving to another sensor
	ds.readAllTags(sensorFrontExit, rssiMin, 20)
	ds.readAllTags(sensorFront01, rssiStrong, 20)
	ds.checkAllTags(Present, sensorFront01)

	// go exiting again
	ds.readAllTags(sensorFrontExit, rssiMax, 20)
	ds.checkAllTags(Exiting, sensorFrontExit)

	// todo: need to check the events have been generated??
}
