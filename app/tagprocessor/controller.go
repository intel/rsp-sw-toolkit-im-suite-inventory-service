package tagprocessor

import (
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
	"strconv"
	"strings"
)

// TODO: Clear exiting tags on run state change notification from the gateway?
//public void onScheduleRunState(ScheduleRunState _current, SchedulerSummary _summary) {
//log.info("onScheduleRunState: {}", _current);
//clearExiting();
//scheduleRunState = _current;
//}

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

func (tag *Tag) update(sensor *RfidSensor, read *TagRead, weighter *rssiAdjuster) {
	// todo: implement

	srcAlias := sensor.getAntennaAlias(read.AntennaId)

	if read.Tid != "" {
		tag.Tid = read.Tid
	}

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

	// todo:: continue here!!!

	locationStats, found := tag.deviceStatsMap[tag.Location]
	if !found {
		// this means the tag has never been read (somehow)
		tag.Location = srcAlias
		tag.DeviceLocation = sensor.DeviceId
		tag.FacilityId = sensor.FacilityId
		tag.addHistory(sensor, read.LastReadOn)
	} else if curStats.getN() > 2 {
		weight := 0.0
		if weighter != nil {
			weight = weighter.getWeight(locationStats.LastRead, sensor)
		}

		//logrus.Debugf("%f, %f", curStats.getRssiMeanDBM(), locationStats.getRssiMeanDBM())

		if curStats.getRssiMeanDBM() > locationStats.getRssiMeanDBM()+weight {
			tag.Location = srcAlias
			tag.DeviceLocation = sensor.DeviceId
			tag.FacilityId = sensor.FacilityId
			tag.addHistory(sensor, read.LastReadOn)
		}
	}
}

func (stats *TagStats) update(read *TagRead) {
	if stats.LastRead != -1 {
		stats.readInterval.addValue(float64(read.LastReadOn - stats.LastRead))
	}
	stats.LastRead = read.LastReadOn
	mw := rssiToMilliwatts(float64(read.Rssi) / 10.0)
	stats.rssiMw.addValue(mw)
}

func (stats *TagStats) getRssiMeanDBM() float64 {
	return milliwattsToRssi(stats.rssiMw.getMean())
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

func (tag *Tag) addHistory(sensor *RfidSensor, timestamp int64) {
	// todo: implement
}

func (sensor *RfidSensor) getAntennaAlias(antennaId int) string {
	var sb strings.Builder
	sb.WriteString(sensor.DeviceId)
	sb.WriteString("-")
	sb.WriteString(strconv.Itoa(antennaId))
	return sb.String()
}

func (weighter *rssiAdjuster) getWeight(lastRead int64, sensor *RfidSensor) float64 {
	profile := weighter.mobilityProfile

	if sensor.isInDeepScan {
		return profile.T
	}

	// todo: is it safe to convert int64 to float64?
	weight := (profile.M * float64(helper.UnixMilliNow()-lastRead)) + profile.B

	// check if weight needs to be capped at threshold ceiling
	if weight > profile.T {
		weight = profile.T
	}

	return weight
}

func (stats *TagStats) getN() int {
	return stats.rssiMw.getN()
}

// find b such that at 60 seconds, y = 3.0
// b = y - (m*x)
func (profile *MobilityProfile) calcB() {
	profile.B = profile.T - (profile.M * profile.A)
}

func (stats *CircularBuffer) addValue(value float64) {
	stats.values[stats.counter%stats.windowSize] = value
	stats.counter++
}

func (stats *CircularBuffer) getN() int {
	//logrus.Debugf("N: %d, %d", stats.counter, stats.windowSize)
	if stats.counter >= stats.windowSize {
		return stats.windowSize
	}
	return stats.counter
}

func (stats *CircularBuffer) getMean() float64 {
	n := stats.getN()
	var total float64
	for i := 0; i < n; i++ {
		total += stats.values[i]
	}
	return total / float64(n)
}
