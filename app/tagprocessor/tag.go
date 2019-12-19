/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package tagprocessor

import (
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/app/sensor"
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/pkg/jsonrpc"
)

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
	History   []*TagHistory

	deviceStatsMap map[string]*TagStats // todo: TreeMap??
}

func NewTag(epc string) *Tag {
	return &Tag{
		Location:       unknown,
		FacilityId:     unknown,
		DeviceLocation: unknown,
		Direction:      Stationary,
		state:          Unknown,
		deviceStatsMap: make(map[string]*TagStats),
		Epc:            epc,
	}
}

func (tag *Tag) asPreviousTag() previousTag {
	return previousTag{
		location:       tag.Location,
		deviceLocation: tag.DeviceLocation,
		facilityId:     tag.FacilityId,
		lastRead:       tag.LastRead,
		lastDeparted:   tag.LastDeparted,
		lastArrived:    tag.LastArrived,
		state:          tag.state,
		direction:      tag.Direction,
	}
}

func (tag *Tag) update(rsp *sensor.RSP, read *jsonrpc.TagRead, weighter *rssiAdjuster) {
	// todo: double check the implementation on this code
	// todo: it may not be complete

	srcAlias := rsp.AntennaAlias(read.AntennaId)

	// only set Tid if it is present
	if read.Tid != "" {
		tag.Tid = read.Tid
	}

	// update timestamp
	tag.LastRead = read.LastReadOn

	curStats, found := tag.deviceStatsMap[srcAlias]
	if !found {
		curStats = NewTagStats()
		tag.deviceStatsMap[srcAlias] = curStats
	}
	curStats.update(read)

	if tag.Location == srcAlias {
		// nothing to do
		return
	}

	locationStats, found := tag.deviceStatsMap[tag.Location]
	if !found {
		// this means the tag has never been read (somehow)
		tag.Location = srcAlias
		tag.DeviceLocation = rsp.DeviceId
		tag.FacilityId = rsp.FacilityId
		tag.addHistory(rsp, read.LastReadOn)
	} else if curStats.getCount() > 2 {
		weight := 0.0
		if weighter != nil {
			weight = weighter.getWeight(locationStats.LastRead, rsp)
		}

		//logrus.Debugf("%f, %f", curStats.getRssiMeanDBM(), locationStats.getRssiMeanDBM())

		if curStats.getRssiMeanDBM() > locationStats.getRssiMeanDBM()+weight {
			tag.Location = srcAlias
			tag.DeviceLocation = rsp.DeviceId
			tag.FacilityId = rsp.FacilityId
			tag.addHistory(rsp, read.LastReadOn)
		}
	}
}

func (tag *Tag) setState(newState TagState) {
	tag.setStateAt(newState, tag.LastRead)
}

func (tag *Tag) setStateAt(newState TagState, timestamp int64) {
	// capture transition times
	switch newState {
	case Present:
		tag.LastArrived = timestamp
		break
	case DepartedExit:
	case DepartedPos:
		tag.LastDeparted = timestamp
		break
	}

	tag.state = newState
}

func (tag *Tag) addHistory(rsp *sensor.RSP, timestamp int64) {
	// todo: implement
}
