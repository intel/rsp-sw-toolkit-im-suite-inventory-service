/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package tagprocessor

import (
	"database/sql"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/sensor"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/jsonrpc"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
	"sync"
	"time"
)

var (
	inventory   = make(map[string]*Tag)
	exitingTags = make(map[string][]*Tag)

	weighter = newRssiAdjuster()

	inventoryMutex = &sync.Mutex{}
)

const (
	unknown         = "UNKNOWN"
	epcEncodeFormat = "tbd"
)

// ProcessInventoryData todo: desc
func ProcessInventoryData(dbs *sql.DB, invData *jsonrpc.InventoryData) (*jsonrpc.InventoryEvent, error) {

	rsp, err := sensor.GetOrCreateRSP(dbs, invData.Params.DeviceId)
	if err != nil {
		return nil, errors.Wrapf(err, "issue trying to retrieve sensor %s from database", invData.Params.DeviceId)
	}

	logrus.Debugf("sentOn: %v, deviceId: %s, facId: %s, reads: %d, personality: %s, aliases: %v, offset: %v ms",
		invData.Params.SentOn, rsp.DeviceId, invData.Params.FacilityId, len(invData.Params.Data), rsp.Personality, rsp.Aliases, helper.UnixMilliNow()-invData.Params.SentOn)

	facId := invData.Params.FacilityId

	if rsp.FacilityId != facId {
		logrus.Debugf("Updating sensor %s facilityId to %s", rsp.DeviceId, facId)
		rsp.FacilityId = facId
		if err = sensor.Upsert(dbs, rsp); err != nil {
			logrus.Errorf("unable to upsert sensor %s. cause: %v", rsp.DeviceId, err)
		}
	}

	invEvent := jsonrpc.NewInventoryEvent()

	for _, read := range invData.Params.Data {
		processReadData(invEvent, &read, rsp)
	}

	return invEvent, nil
}

func processReadData(invEvent *jsonrpc.InventoryEvent, read *jsonrpc.TagRead, rsp *sensor.RSP) {
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
		if rsp.IsPOSSensor() {
			break
		}

		tag.setState(Present)
		addEvent(invEvent, tag, Arrival)
		break

	case Present:
		if rsp.IsPOSSensor() {
			if !checkDepartPOS(invEvent, tag) {
				checkMovement(invEvent, tag, &prev)
			}
		} else {
			checkExiting(rsp, tag)
			checkMovement(invEvent, tag, &prev)
		}
		break

	case Exiting:
		if rsp.IsPOSSensor() {
			checkDepartPOS(invEvent, tag)
		} else {
			if !rsp.IsExitSensor() && rsp.DeviceId == tag.DeviceLocation {
				tag.setState(Present)
			}
			checkMovement(invEvent, tag, &prev)
		}
		break

	case DepartedExit:
		if rsp.IsPOSSensor() {
			break
		}

		doTagReturn(invEvent, tag, &prev)
		checkExiting(rsp, tag)
		break

	case DepartedPos:
		if rsp.IsPOSSensor() {
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
	if !rsp.IsExitSensor() || rsp.DeviceId != tag.DeviceLocation {
		return
	}
	addExiting(rsp.FacilityId, tag)
}

func OnSchedulerRunState(runState *jsonrpc.SchedulerRunState) {
	// clear any cached exiting tag status
	logrus.Infof("Scheduler run state has changed to %s. Clearing exiting status of all tags.", runState.Params.RunState)
	clearExiting()
}

func clearExiting() {
	inventoryMutex.Lock()
	defer inventoryMutex.Unlock()

	for _, tags := range exitingTags {
		for _, tag := range tags {
			// test just to be sure, this should not be necessary but belt and suspenders
			if tag.state == Exiting {
				tag.setStateAt(Present, tag.LastArrived)
			}
		}
	}
	exitingTags = make(map[string][]*Tag)
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

func DoAgeoutTask() int {
	inventoryMutex.Lock()
	defer inventoryMutex.Unlock()

	expiration := helper.UnixMilli(time.Now().Add(
		time.Hour * time.Duration(-config.AppConfig.AgeOutHours)))

	// it is safe to remove from map while iterating in golang
	var numRemoved int
	for epc, tag := range inventory {
		if tag.LastRead < expiration {
			numRemoved++
			delete(inventory, epc)
		}
	}

	logrus.Infof("inventory ageout removed %d tags", numRemoved)
	return numRemoved
}

func DoAggregateDepartedTask() *jsonrpc.InventoryEvent {
	inventoryMutex.Lock()
	defer inventoryMutex.Unlock()

	// acquire lock BEFORE getting the timestamps, otherwise they can be invalid if we have to wait for the lock
	now := helper.UnixMilliNow()
	expiration := now - int64(config.AppConfig.AggregateDepartedThresholdMillis)

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

	return invEvent
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
