package tagprocessor

import "fmt"

const (
	backStock    = "BackStock"
	salesFloor   = "SalesFloor"
	facilityCold = "Cold"
	facilityDry  = "Dry"
	facilityA    = "A"
	facilityB    = "B"
	facilityC    = "C"

	defaultFrequency = 927
)

var (
	rssiMin    = -95 * 10
	rssiMax    = -55 * 10
	rssiStrong = rssiMax - (rssiMax-rssiMin)/3
	rssiWeak   = rssiMin + (rssiMax-rssiMin)/3

	// todo: AtomicInteger??
	tagSerialNum = 1

	sensorFront01 = &RfidSensor{DeviceId: "RSP-150000", FacilityId: salesFloor, Personality: NoPersonality}
	sensorFront02 = &RfidSensor{DeviceId: "RSP-150001", FacilityId: salesFloor, Personality: NoPersonality}
	sensorFront03 = &RfidSensor{DeviceId: "RSP-150002", FacilityId: salesFloor, Personality: NoPersonality}

	sensorFrontPOS  = &RfidSensor{DeviceId: "RSP-150003", FacilityId: salesFloor, Personality: POS}
	sensorFrontExit = &RfidSensor{DeviceId: "RSP-150004", FacilityId: salesFloor, Personality: Exit}

	sensorBack01 = &RfidSensor{DeviceId: "RSP-150005", FacilityId: backStock, Personality: NoPersonality}
	sensorBack02 = &RfidSensor{DeviceId: "RSP-150006", FacilityId: backStock, Personality: NoPersonality}
	sensorBack03 = &RfidSensor{DeviceId: "RSP-150007", FacilityId: backStock, Personality: NoPersonality}

	sensorCold01 = &RfidSensor{DeviceId: "RSP-150008", FacilityId: facilityCold, Personality: NoPersonality}
	sensorDry01  = &RfidSensor{DeviceId: "RSP-150009", FacilityId: facilityDry, Personality: NoPersonality}

	sensorA01     = &RfidSensor{DeviceId: "RSP-150010", FacilityId: facilityA, Personality: NoPersonality}
	sensorB01     = &RfidSensor{DeviceId: "RSP-150011", FacilityId: facilityB, Personality: NoPersonality}
	sensorCexit01 = &RfidSensor{DeviceId: "RSP-150012", FacilityId: facilityC, Personality: Exit}
	sensorCexit02 = &RfidSensor{DeviceId: "RSP-150013", FacilityId: facilityC, Personality: Exit}
)

func generateReadData(lastRead int64, initialRssi int) *TagRead {
	serial := tagSerialNum
	tagSerialNum++

	return &TagRead{
		Epc:        fmt.Sprintf("EPC%06d", serial),
		Tid:        fmt.Sprintf("TID%06d", serial),
		Frequency:  defaultFrequency,
		Rssi:       initialRssi,
		LastReadOn: lastRead,
	}
}
