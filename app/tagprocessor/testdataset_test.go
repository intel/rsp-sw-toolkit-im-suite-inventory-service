package tagprocessor

import (
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
	"testing"
)

type testDataset struct {
	t            *testing.T
	tagReads     []*TagRead
	tags         []*Tag
	readTimeOrig int64
}

func newTestDataset(t *testing.T) testDataset {
	return testDataset{t: t}
}

// will generate tagread objects but NOT ingest them yet
func (ds *testDataset) initialize(tagCount int, initialRssi int) {
	ds.tagReads = make([]*TagRead, tagCount)
	ds.tags = make([]*Tag, tagCount)
	ds.readTimeOrig = helper.UnixMilliNow()

	for i := 0; i < tagCount; i++ {
		ds.tagReads[i] = generateReadData(ds.readTimeOrig, initialRssi)
	}

	// resetEvents()
}

// update the tag pointers based on actual ingested data
func (ds *testDataset) updateTagRefs() {
	for i, tagRead := range ds.tagReads {
		ds.tags[i] = inventory[tagRead.Epc]
	}
}

func (ds *testDataset) setRssi(tagIndex int, rssi int) {
	ds.tagReads[tagIndex].Rssi = rssi
}

func (ds *testDataset) setRssiAll(rssi int) {
	for _, tagRead := range ds.tagReads {
		tagRead.Rssi = rssi
	}
}

func (ds *testDataset) setLastReadOnAll(timestamp int64) {
	for _, tagRead := range ds.tagReads {
		tagRead.LastReadOn = timestamp
	}
}

func (ds *testDataset) readTag(tagIndex int, sensor *RfidSensor, rssi int, times int) {
	ds.setRssi(tagIndex, rssi)

	for i := 0; i < times; i++ {
		processReadData(ds.tagReads[tagIndex], sensor)
	}
}

func (ds *testDataset) readAllTags(sensor *RfidSensor, rssi int, times int) {
	for tagIndex := range ds.tagReads {
		ds.readTag(tagIndex, sensor, rssi, times)
	}
}

func (ds *testDataset) size() int {
	return len(ds.tagReads)
}

func (ds *testDataset) checkAllTags(expectedState TagState, expectedSensor *RfidSensor) {
	for i := range ds.tags {
		ds.checkTag(i, expectedState, expectedSensor)
	}
}

func (ds *testDataset) checkTag(tagIndex int, expectedState TagState, expectedSensor *RfidSensor) {
	tag := ds.tags[tagIndex]

	if tag == nil {
		ds.t.Errorf("Expected tag index %d to not be nil!", tagIndex)
	} else if tag.state != expectedState {
		ds.t.Errorf("tag state %v does not match expected state %v for tag %v", tag.state, expectedState, tag)
	} else if tag.DeviceLocation != expectedSensor.DeviceId {
		ds.t.Errorf("tag location %v does not match expected sensor %v for tag %v", tag.DeviceLocation, expectedSensor.DeviceId, tag)
	}
}
