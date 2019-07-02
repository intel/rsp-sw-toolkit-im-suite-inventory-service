package tagprocessor

import "math"

func rssiToMilliwatts(rssi float64) float64 {
	return math.Pow(10, rssi/10.0)
}

func milliwattsToRssi(mw float64) float64 {
	return math.Log10(mw) * 10.0
}
