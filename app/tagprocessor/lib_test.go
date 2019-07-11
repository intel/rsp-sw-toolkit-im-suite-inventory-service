package tagprocessor

import "testing"

func TestTagArrival(t *testing.T) {
	ds := newTestDataset(3)

	back := generateTestSensor(backStock, NoPersonality)
	frontPos := generateTestSensor(salesFloor, POS)

	back.minRssiDbm10X = -600

	ds.readTag(0, back, -580, 1)
	ds.readTag(1, back, -620, 1)
	ds.readTag(2, frontPos, rssiMin, 1)

	ds.updateTagRefs()

	if err := ds.verifyTag(0, Present, back); err != nil {
		t.Error(err)
	}
	// tag1 will NOT arrive due to having an rssi too low
	if ds.tags[1] != nil {
		t.Errorf("expected tag with index 1 to be nil, but was %v", ds.tags[1])
	}
	if err := ds.verifyTag(2, Unknown, frontPos); err != nil {
		t.Error(err)
	}

	// todo: check for 1 arrival events
}

func TestTagMove(t *testing.T) {
	ds := newTestDataset(3)

	front := generateTestSensor(salesFloor, NoPersonality)
	back1 := generateTestSensor(backStock, NoPersonality)
	back2 := generateTestSensor(backStock, NoPersonality)
	back3 := generateTestSensor(backStock, NoPersonality)

	// start all tags in the back stock
	ds.readAll(back1, rssiMin, 1)
	ds.updateTagRefs()

	// move tag 0 to the front
	ds.readTag(0, front, rssiStrong, 4)
	if err := ds.verifyTag(0, Present, front); err != nil {
		t.Error(err)
	}
	// todo: check for events

	// move tag 1 to same facility, different sensor
	ds.readTag(1, back2, rssiStrong, 4)
	if err := ds.verifyTag(1, Present, back2); err != nil {
		t.Error(err)
	}
	// todo: check for events

	// test that tag stays at new location even with concurrent reads from weaker sensor
	// MOVE back doesn't happen with weak RSSI
	ds.readTag(1, back3, rssiWeak, 4)
	// todo: this appears broken in my code as it is failing
	if err := ds.verifyTag(1, Present, back2); err != nil {
		t.Error(err)
	}

	// move tag 2 to a different antenna port on same sensor
	ds.tagReads[2].AntennaId = 33
	ds.readTag(2, back1, rssiStrong, 4)
	if err := ds.verifyTag(2, Present, back1); err != nil {
		t.Error(err)
	}
}

func TestBasicExit(t *testing.T) {
	ds := newTestDataset(9)

	back := generateTestSensor(backStock, NoPersonality)
	frontExit := generateTestSensor(salesFloor, Exit)
	front := generateTestSensor(salesFloor, NoPersonality)

	// get it in the system
	ds.readAll(back, rssiMin, 4)
	ds.updateTagRefs()

	// one tag read by an EXIT will not make the tag go exiting.
	ds.readAll(frontExit, rssiMin, 1)
	if err := ds.verifyAll(Present, back); err != nil {
		t.Error(err)
	}

	// moving to an exit sensor will put tag in exiting
	// moving to an exit sensor in another facility will generate departure / arrival
	ds.readAll(frontExit, rssiWeak, 10)
	if err := ds.verifyAll(Exiting, frontExit); err != nil {
		t.Error(err)
	}
	// todo: need to check for events that were generated

	// clear exiting by moving to another sensor
	// done in a loop to simulate being read simultaneously, not 20 on one sensor, and 20 on another
	for i := 0; i < 20; i++ {
		ds.readAll(frontExit, rssiMin, 1)
		ds.readAll(front, rssiStrong, 1)
	}
	if err := ds.verifyAll(Present, front); err != nil {
		t.Error(err)
	}

	ds.readAll(frontExit, rssiMax, 20)
	if err := ds.verifyAll(Exiting, frontExit); err != nil {
		t.Error(err)
	}
}

func TestExitingArrivalDepartures(t *testing.T) {
	ds := newTestDataset(5)

	back := generateTestSensor(backStock, NoPersonality)
	frontExit := generateTestSensor(salesFloor, Exit)
	front := generateTestSensor(salesFloor, NoPersonality)

	ds.readAll(back, rssiMin, 4)

	ds.updateTagRefs()
	if err := ds.verifyAll(Present, back); err != nil {
		t.Error(err)
	}

	// one tag read by an EXIT will not make the tag go exiting.
	ds.readAll(frontExit, rssiWeak, 1)
	if err := ds.verifyAll(Present, back); err != nil {
		t.Error(err)
	}

	// go to exiting state
	ds.readAll(frontExit, rssiWeak, 10)
	if err := ds.verifyAll(Exiting, frontExit); err != nil {
		t.Error(err)
	}

	// clear exiting by moving to another sensor
	ds.readAll(frontExit, rssiMin, 20)
	ds.readAll(front, rssiStrong, 20)
	if err := ds.verifyAll(Present, front); err != nil {
		t.Error(err)
	}

	// go exiting again
	ds.readAll(frontExit, rssiMax, 20)
	if err := ds.verifyAll(Exiting, frontExit); err != nil {
		t.Error(err)
	}

	// todo: need to check the events have been generated??
}

func TestTagDepartAndReturnFromExit(t *testing.T) {

}

func TestTagDepartAndReturnPOS(t *testing.T) {

}
