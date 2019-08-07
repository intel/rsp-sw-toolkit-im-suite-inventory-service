package tagprocessor

import (
	"github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/sensor"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/jsonrpc"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
	"sync"
	"time"
)

var (
	inventory   = make(map[string]*Tag) // todo: TreeMap?
	exitingTags = make(map[string][]*Tag)

	rfidSensors = make(map[string]*sensor.RSP)

	weighter = newRssiAdjuster()

	inventoryMutex = &sync.Mutex{}
	sensorMutex    = &sync.Mutex{}
)

const (
	unknown         = "UNKNOWN"
	epcEncodeFormat = "tbd"
)

// TODO: Clear exiting tags on run state change notification from the gateway?
//public void onScheduleRunState(ScheduleRunState _current, SchedulerSummary _summary) {
//log.info("onScheduleRunState: {}", _current);
//clearExiting();
//scheduleRunState = _current;
//}

// ProcessInventoryData todo: desc
func ProcessInventoryData(invData *jsonrpc.InventoryData) (*jsonrpc.InventoryEvent, error) {
	rsp := lookupRSP(invData.Params.DeviceId)
	facId := invData.Params.FacilityId

	if rsp.FacilityId != facId {
		logrus.Debugf("Updating sensor %s facilityId to %s.\nSensor Map: %#v", rsp.DeviceId, facId, rfidSensors)
		rsp.FacilityId = facId
	}

	invEvent := jsonrpc.NewInventoryEvent()

	for _, read := range invData.Params.Data {
		processReadData(invEvent, &read, rsp)
	}

	return invEvent, nil
}

func lookupRSP(deviceId string) *sensor.RSP {
	sensorMutex.Lock()
	defer sensorMutex.Unlock()

	rsp, found := rfidSensors[deviceId]

	if !found {
		rsp = sensor.NewRSP(deviceId)
		rfidSensors[deviceId] = rsp
	}

	return rsp
}

func processReadData(invEvent *jsonrpc.InventoryEvent, read *jsonrpc.TagRead, rsp *sensor.RSP) {
	if !rsp.RssiInRange(read.Rssi) {
		return
	}

	inventoryMutex.Lock()

	tag, exists := inventory[read.Epc]
	if !exists {
		tag = NewTag(read.Epc)
		inventory[read.Epc] = tag
	}

	prev := tag.asPreviousTag()
	tag.update(rsp, read, &weighter)

	switch prev.state {

	case Unknown:
		// Point of sale NEVER adds new tags to the inventory
		// for the use case of POS reader might be the first
		// sensor in the store hallway to see a tag etc. so
		// need to prevent premature departures
		if rsp.POSSensor() {
			break
		}

		tag.setState(Present)
		addEvent(invEvent, tag, Arrival)
		break

	case Present:
		if rsp.POSSensor() {
			if !checkDepartPOS(invEvent, tag) {
				checkMovement(invEvent, tag, &prev)
			}
		} else {
			checkExiting(rsp, tag)
			checkMovement(invEvent, tag, &prev)
		}
		break

	case Exiting:
		if rsp.POSSensor() {
			checkDepartPOS(invEvent, tag)
		} else {
			if !rsp.ExitSensor() && rsp.DeviceId == tag.DeviceLocation {
				tag.setState(Present)
			}
			checkMovement(invEvent, tag, &prev)
		}
		break

	case DepartedExit:
		if rsp.POSSensor() {
			break
		}

		doTagReturn(invEvent, tag, &prev)
		checkExiting(rsp, tag)
		break

	case DepartedPos:
		if rsp.POSSensor() {
			break
		}

		// Such a tag must remain in the DEPARTED state for
		// a configurable amount of time (i.e. 1 day)
		if tag.LastDeparted < (tag.LastRead - int64(config.AppConfig.PosReturnThresholdMillis)) {
			doTagReturn(invEvent, tag, &prev)
			checkExiting(rsp, tag)
		}
		break
	}

	inventoryMutex.Unlock()
}

func checkDepartPOS(invEvent *jsonrpc.InventoryEvent, tag *Tag) bool {
	// if tag is ever read by a POS, it immediately generates a departed event
	// as long as it has been seen by our system for a minimum period of time first
	expiration := tag.LastRead - int64(config.AppConfig.PosDepartedThresholdMillis)

	if tag.LastArrived < expiration {
		tag.setState(DepartedPos)
		addEvent(invEvent, tag, Departed)
		logrus.Debugf("Departed POS: %v", tag)
		return true
	}

	return false
}

func checkMovement(invEvent *jsonrpc.InventoryEvent, tag *Tag, prev *previousTag) {
	if prev.location != "" && prev.location != tag.Location {
		if prev.facilityId != "" && prev.facilityId != tag.FacilityId {
			// change facility (depart old facility, arrive new facility)
			addEventDetails(invEvent, tag.Epc, tag.Tid, prev.location, prev.facilityId, Departed, prev.lastRead)
			addEvent(invEvent, tag, Arrival)
		} else {
			addEvent(invEvent, tag, Moved)
		}
	}
}

func checkExiting(rsp *sensor.RSP, tag *Tag) {
	if !rsp.ExitSensor() || rsp.DeviceId != tag.DeviceLocation {
		return
	}
	addExiting(rsp.FacilityId, tag)
}

func clearExiting() {
	inventoryMutex.Lock()
	for _, tags := range exitingTags {
		for _, tag := range tags {
			// test just to be sure, this should not be necessary but belt and suspenders
			if tag.state == Exiting {
				tag.setStateAt(Present, tag.LastArrived)
			}
		}
	}
	exitingTags = make(map[string][]*Tag)
	inventoryMutex.Unlock()
}

func addExiting(facilityId string, tag *Tag) {
	tag.setState(Exiting)

	tags, found := exitingTags[facilityId]
	if !found {
		exitingTags[facilityId] = []*Tag{tag}
	} else {
		exitingTags[facilityId] = append(tags, tag)
	}
}

func doTagReturn(invEvent *jsonrpc.InventoryEvent, tag *Tag, prev *previousTag) {
	if prev.facilityId != "" && prev.facilityId == tag.FacilityId {
		addEvent(invEvent, tag, Returned)
	} else {
		addEvent(invEvent, tag, Arrival)
	}
	tag.setState(Present)
}

// todo: when to call this? on a schedule?
func ageout() int {
	expiration := helper.UnixMilli(time.Now().Add(
		time.Hour * time.Duration(-config.AppConfig.AgeOutHours)))

	inventoryMutex.Lock()

	// it is safe to remove from map while iterating in golang
	var numRemoved int
	for epc, tag := range inventory {
		if tag.LastRead < expiration {
			numRemoved++
			delete(inventory, epc)
		}
	}

	inventoryMutex.Unlock()

	logrus.Infof("inventory ageout removed %d tags", numRemoved)
	return numRemoved
}

// todo: when to call this? on schedule?
func doAggregateDepartedTask() {
	now := helper.UnixMilliNow()
	expiration := now - int64(config.AppConfig.AggregateDepartedThresholdMillis)

	inventoryMutex.Lock()

	invEvent := jsonrpc.NewInventoryEvent()

	for _, tags := range exitingTags {
		keepIndex := 0
		for _, tag := range tags {

			if tag.state != Exiting {
				// there may be some edge cases where the tag state is invalid
				// skip and do not keep
				continue
			}

			if tag.LastRead < expiration {
				tag.setStateAt(DepartedExit, now)
				logrus.Debugf("Departed %v", tag)
				addEvent(invEvent, tag, Departed)
			} else {
				// if the tag is to be kept, put it back in the slice
				tags[keepIndex] = tag
				keepIndex++
			}
		}
		// shrink to fit actual size
		tags = tags[:keepIndex]
	}

	// todo: do something with invEvent

	inventoryMutex.Unlock()
}

func addEvent(invEvent *jsonrpc.InventoryEvent, tag *Tag, event Event) {
	addEventDetails(invEvent, tag.Epc, tag.Tid, tag.Location, tag.FacilityId, event, tag.LastRead)
}

func addEventDetails(invEvent *jsonrpc.InventoryEvent, epc string, tid string, location string, facilityId string, event Event, timestamp int64) {
	logrus.Infof("Sending event {epc: %s, tid: %s, event_type: %s, facility_id: %s, location: %s, timestamp: %d}",
		epc, tid, event, facilityId, location, timestamp)

	invEvent.AddTagEvent(jsonrpc.TagEvent{
		Timestamp:       timestamp,
		Location:        location,
		Tid:             tid,
		EpcCode:         epc,
		EpcEncodeFormat: epcEncodeFormat,
		EventType:       string(event),
		FacilityID:      facilityId,
	})
}
