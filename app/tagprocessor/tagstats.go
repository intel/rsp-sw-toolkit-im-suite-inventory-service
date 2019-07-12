package tagprocessor

type TagStats struct {
	LastRead     int64
	readInterval *CircularBuffer
	rssiMw       *CircularBuffer
	// todo: implement
}

func NewTagStats() *TagStats {
	return &TagStats{
		readInterval: NewCircularBuffer(defaultWindowSize),
		rssiMw:       NewCircularBuffer(defaultWindowSize),
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

func (stats *TagStats) getN() int {
	return stats.rssiMw.getN()
}
