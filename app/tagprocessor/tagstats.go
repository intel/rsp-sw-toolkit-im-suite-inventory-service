/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package tagprocessor

import "github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/pkg/jsonrpc"

// TagStats helps keep track of tag read rssi values over time
type TagStats struct {
	LastRead     int64
	readInterval *CircularBuffer
	rssiMw       *CircularBuffer
}

func NewTagStats() *TagStats {
	return &TagStats{
		readInterval: NewCircularBuffer(defaultWindowSize),
		rssiMw:       NewCircularBuffer(defaultWindowSize),
	}
}

func (stats *TagStats) update(read *jsonrpc.TagRead) {
	if stats.LastRead != 0 {
		stats.readInterval.AddValue(float64(read.LastReadOn - stats.LastRead))
	}
	stats.LastRead = read.LastReadOn

	mw := rssiToMilliwatts(float64(read.Rssi) / 10.0)
	stats.rssiMw.AddValue(mw)
}

func (stats *TagStats) getRssiMeanDBM() float64 {
	return milliwattsToRssi(stats.rssiMw.GetMean())
}

func (stats *TagStats) getCount() int {
	return stats.rssiMw.GetCount()
}
