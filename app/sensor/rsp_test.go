package sensor

import (
	"testing"
)

func TestRSPAntennaAlias(t *testing.T) {
	tests := []struct {
		deviceId  string
		antennaId int
		expected  string
	}{
		{
			deviceId:  "RSP-3F7DAC",
			antennaId: 0,
			expected:  "RSP-3F7DAC-0",
		},
		{
			deviceId:  "RSP-150000",
			antennaId: 10,
			expected:  "RSP-150000-10",
		},
		{
			deviceId:  "RSP-999999",
			antennaId: 3,
			expected:  "RSP-999999-3",
		},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			rsp := NewRSP(test.deviceId)
			alias := rsp.AntennaAlias(test.antennaId)
			if alias != test.expected {
				t.Errorf("Expected alias of %s, but got %s", test.expected, alias)
			}
		})
	}
}
