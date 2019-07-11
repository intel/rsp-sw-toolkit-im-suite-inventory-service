package tagprocessor

import (
	"github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
	"sync"
	"time"
)

var (
	inventory   = make(map[string]*Tag) // todo: TreeMap?
	exitingTags = make(map[string][]*Tag)

	rfidSensors = make(map[string]*RfidSensor)

	weighter = NewRssiAdjuster()

	inventoryMutex = &sync.Mutex{}
)

const (
	unknown = "UNKNOWN"
)

// OnInventoryData todo: desc
func OnInventoryData(data PeriodicInventoryData) error {
	sensor := lookupSensor(data.DeviceId)
	if sensor.FacilityId != data.FacilityId {
		logrus.Debugf("Updating sensor %s facilityId to %s", sensor.DeviceId, data.FacilityId)
		sensor.FacilityId = data.FacilityId
	}

	for _, read := range data.Data {
		// todo: handle error?
		processReadData(&read, sensor)
	}

	return nil
}

func lookupSensor(deviceId string) *RfidSensor {
	sensor, found := rfidSensors[deviceId]

	if !found {
		sensor = NewRfidSensor(deviceId)
		rfidSensors[deviceId] = sensor
	}

	return sensor
}

func processReadData(read *TagRead, sensor *RfidSensor) {
	if sensor.minRssiDbm10X != 0 && read.Rssi < sensor.minRssiDbm10X {
		return
	}

	inventoryMutex.Lock()

	tag, exists := inventory[read.Epc]
	if !exists {
		tag = NewTag(read.Epc)
		inventory[read.Epc] = tag
	}

	prev := tag.asPreviousTag()
	tag.update(sensor, read, &weighter)

	switch prev.state {

	case Unknown:
		// Point of sale NEVER adds new tags to the inventory
		// for the use case of POS reader might be the first
		// sensor in the store hallway to see a tag etc. so
		// need to prevent premature departures
		if sensor.Personality == POS {
			break
		}

		tag.setState(Present)
		sendEvent(tag, Arrival)
		break

	case Present:
		if sensor.Personality == POS {
			if !checkDepartPOS(tag) {
				checkMovement(tag, &prev)
			}
		} else {
			checkExiting(sensor, tag)
			checkMovement(tag, &prev)
		}
		break

	case Exiting:
		if sensor.Personality == POS {
			checkDepartPOS(tag)
		} else {
			if sensor.Personality != Exit && sensor.DeviceId == tag.DeviceLocation {
				tag.setState(Present)
			}
			checkMovement(tag, &prev)
		}
		break

	case DepartedExit:
		if sensor.Personality == POS {
			break
		}

		doTagReturn(tag, &prev)
		checkExiting(sensor, tag)
		break

	case DepartedPos:
		if sensor.Personality == POS {
			break
		}

		// Such a tag must remain in the DEPARTED state for
		// a configurable amount of time (i.e. 1 day)
		if tag.LastDeparted < (tag.LastRead - config.AppConfig.PosReturnThresholdMillis) {
			doTagReturn(tag, &prev)
			checkExiting(sensor, tag)
		}
		break
	}

	inventoryMutex.Unlock()
}

func checkDepartPOS(tag *Tag) bool {
	// if tag is ever read by a POS, it immediately generates a departed event
	// as long as it has been seen by our system for a minimum period of time first
	expiration := tag.LastRead - config.AppConfig.PosDepartedThresholdMillis

	if tag.LastArrived < expiration {
		tag.setState(DepartedPos)
		sendEvent(tag, Departed)
		logrus.Debugf("Departed POS: %v", tag)
		return true
	}

	return false
}

func checkMovement(tag *Tag, prev *previousTag) {
	if prev.location != "" && prev.location != tag.Location {
		if prev.facilityId != "" && prev.facilityId != tag.FacilityId {
			// change facility (depart old facility, arrive new facility)
			sendEventDetails(tag.Epc, tag.Tid, prev.location, prev.facilityId, Departed, prev.lastRead)
			sendEvent(tag, Arrival)
		} else {
			sendEvent(tag, Moved)
		}
	}
}

func checkExiting(sensor *RfidSensor, tag *Tag) {
	if sensor.Personality != Exit || sensor.DeviceId != tag.DeviceLocation {
		return
	}
	addExiting(sensor.FacilityId, tag)
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

func doTagReturn(tag *Tag, prev *previousTag) {
	if prev.facilityId != "" && prev.facilityId == tag.FacilityId {
		sendEvent(tag, Returned)
	} else {
		sendEvent(tag, Arrival)
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
	expiration := now - config.AppConfig.AggregateDepartedThresholdMillis

	inventoryMutex.Lock()

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
				sendEvent(tag, Departed)
			} else {
				// if the tag is to be kept, put it back in the slice
				tags[keepIndex] = tag
				keepIndex++
			}
		}
		// shrink to fit actual size
		tags = tags[:keepIndex]
	}

	inventoryMutex.Unlock()
}

func sendEvent(tag *Tag, event TagEvent) {
	// todo: implement ingest into inventory
	logrus.Infof("Sending event %v for epc %s in %s %v", event, tag.Epc, tag.FacilityId, tag)
}

func sendEventDetails(epc string, tid string, location string, facilityId string, event TagEvent, lastRead int64) {
	// todo: implement ingest into inventory
	logrus.Infof("Sending event %v for epc %s in %s", event, epc, facilityId)
}
