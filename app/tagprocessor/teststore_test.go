package tagprocessor

import (
	"fmt"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/jsonrpc"
	"sync/atomic"
)

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

	tagSerialCounter uint32
	sensorIdCounter  uint32 = 150000 - 1

	//sensorFront01 = &RfidSensor{DeviceId: "RSP-150000", FacilityId: salesFloor, Personality: NoPersonality}
	//sensorFront02 = &RfidSensor{DeviceId: "RSP-150001", FacilityId: salesFloor, Personality: NoPersonality}
	//sensorFront03 = &RfidSensor{DeviceId: "RSP-150002", FacilityId: salesFloor, Personality: NoPersonality}
	//
	//sensorFrontPOS  = &RfidSensor{DeviceId: "RSP-150003", FacilityId: salesFloor, Personality: POS}
	//sensorFrontExit = &RfidSensor{DeviceId: "RSP-150004", FacilityId: salesFloor, Personality: Exit}
	//
	//sensorBack01 = &RfidSensor{DeviceId: "RSP-150005", FacilityId: backStock, Personality: NoPersonality}
	//sensorBack02 = &RfidSensor{DeviceId: "RSP-150006", FacilityId: backStock, Personality: NoPersonality}
	//sensorBack03 = &RfidSensor{DeviceId: "RSP-150007", FacilityId: backStock, Personality: NoPersonality}
	//
	//sensorCold01 = &RfidSensor{DeviceId: "RSP-150008", FacilityId: facilityCold, Personality: NoPersonality}
	//sensorDry01  = &RfidSensor{DeviceId: "RSP-150009", FacilityId: facilityDry, Personality: NoPersonality}
	//
	//sensorA01     = &RfidSensor{DeviceId: "RSP-150010", FacilityId: facilityA, Personality: NoPersonality}
	//sensorB01     = &RfidSensor{DeviceId: "RSP-150011", FacilityId: facilityB, Personality: NoPersonality}
	//sensorCexit01 = &RfidSensor{DeviceId: "RSP-150012", FacilityId: facilityC, Personality: Exit}
	//sensorCexit02 = &RfidSensor{DeviceId: "RSP-150013", FacilityId: facilityC, Personality: Exit}
)

func generateTestSensor(facilityId string, personality Personality) *RfidSensor {
	sensorId := atomic.AddUint32(&sensorIdCounter, 1)

	return &RfidSensor{
		DeviceId:    fmt.Sprintf("RSP-%06d", sensorId),
		FacilityId:  facilityId,
		Personality: personality,
	}
}

func generateReadData(lastRead int64) *jsonrpc.TagRead {
	serial := atomic.AddUint32(&tagSerialCounter, 1)

	return &jsonrpc.TagRead{
		Epc:        fmt.Sprintf("EPC%06d", serial),
		Tid:        fmt.Sprintf("TID%06d", serial),
		Frequency:  defaultFrequency,
		Rssi:       rssiMin,
		LastReadOn: lastRead,
	}
}
