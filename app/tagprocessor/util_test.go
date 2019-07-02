package tagprocessor

import (
	"math"
	"testing"
)

const (
	// floatPrecision is the largest difference allowed for comparing floating point numbers in this test file
	floatPrecision = 1e-12
)

func TestRssiConversions(t *testing.T) {
	sampleRssis := []float64{-640, -320, -654, -1000, -290, -126, -1, 0, 1, 100, 333, 950}

	for _, sampleRssi := range sampleRssis {
		mw := rssiToMilliwatts(sampleRssi)
		rssi := milliwattsToRssi(mw)
		if math.Abs(sampleRssi - rssi) > floatPrecision {
			t.Errorf("Converting rssi to mw and back resulted in a different value %v dBm -> %v mw -> %v dBm, Diff: %v",
				sampleRssi, mw, rssi, math.Abs(sampleRssi - rssi))
		}
	}
}
