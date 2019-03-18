/*
 * INTEL CONFIDENTIAL
 * Copyright (2017) Intel Corporation.
 *
 * The source code contained or described herein and all documents related to the source code ("Material")
 * are owned by Intel Corporation or its suppliers or licensors. Title to the Material remains with
 * Intel Corporation or its suppliers and licensors. The Material may contain trade secrets and proprietary
 * and confidential information of Intel Corporation and its suppliers and licensors, and is protected by
 * worldwide copyright and trade secret laws and treaty provisions. No part of the Material may be used,
 * copied, reproduced, modified, published, uploaded, posted, transmitted, distributed, or disclosed in
 * any way without Intel/'s prior express written permission.
 * No license under any patent, copyright, trade secret or other intellectual property right is granted
 * to or conferred upon you by disclosure or delivery of the Materials, either expressly, by implication,
 * inducement, estoppel or otherwise. Any license under such intellectual property rights must be express
 * and approved by Intel in writing.
 * Unless otherwise agreed by Intel in writing, you may not remove or alter this notice or any other
 * notice embedded in Materials by Intel or Intel's suppliers or licensors in any way.
 */

package statemodel

import (
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/encodingscheme"
	"strings"
	"time"

	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/tag"
)

//IsTagWhitelisted determines if the tag received from gateway
//belongs to the list of whitelisted epcs and returns true or false
func IsTagWhitelisted(epc string, whiteList []string) bool {
	for i := range whiteList {
		if strings.HasPrefix(epc, whiteList[i]) {
			return true
		}
	}
	return false
}

//UpdateTag updates the existing tag or creates a new tag.
//Sets certain fields based on current tag values and the
//qualified events state model
//nolint :gocyclo
func UpdateTag(currentState tag.Tag, newTagEvent tag.TagEvent, source string) tag.Tag {

	isNewTag := currentState.IsEmpty() || currentState.IsShippingNoticeEntry()

	if isNewTag {
		currentState.Epc = newTagEvent.EpcCode
		currentState.ProductID, currentState.URI = tag.DecodeTagData(currentState.Epc)
		currentState.FilterValue, _ = encodingscheme.GetItemFilter(currentState.Epc)
		currentState.Event = GetNewTagEvent(newTagEvent.EventType)
		currentState.Arrived = newTagEvent.Timestamp
		currentState.LocationHistory = []tag.LocationHistory{}

		//if new event from gw is anything but departed set it epc state to present
		if newTagEvent.EventType != DepartedEvent {
			currentState.EpcState = PresentEpcState
		} else {
			currentState.EpcState = DepartedEpcState
		}

		//On new tag set the qualified state to "unknown"
		currentState.QualifiedState = UnknownQualifiedState
	} else {
		// Not a new TAG
		if config.AppConfig.NewerHandheldHavePriority &&
			source == "fixed" && currentState.Source == "handheld" &&
			currentState.LastRead > newTagEvent.Timestamp {
			// Skip updating existing newer handheld tag with incoming older fixed tag
			return currentState
		}

		//if any existing tags do not have a qualified state
		//update it with the "unknown" value
		if len(currentState.QualifiedState) == 0 {
			currentState.QualifiedState = UnknownQualifiedState
		}

		//if any existing tags do not have a gtin value
		//call the update to populate it from its epc value
		if len(currentState.ProductID) == 0 {
			currentState.ProductID, _ = tag.DecodeTagData(currentState.Epc)
			currentState.FilterValue, _ = encodingscheme.GetItemFilter(currentState.Epc)
		}

		if newTagEvent.EventType == CycleCountEvent && currentState.EpcState == PresentEpcState {
			currentState.CycleCount = true
		} else {
			currentState.CycleCount = false
		}

	}

	currentState.FacilityID = newTagEvent.FacilityID

	newState := tag.Tag(currentState)

	newState.LastRead = getBestLastRead(currentState.LastRead, newTagEvent.Timestamp, currentState.Source, source)
	newState.EpcEncodeFormat = newTagEvent.EpcEncodeFormat
	newState.Tid = newTagEvent.Tid
	newState.Source = source

	//We only want to update or change certain fields if the current
	//epc state and the new event both do not equal departed
	if !(currentState.EpcState == DepartedEpcState && newTagEvent.EventType == DepartedEvent) {

		//only update the event if it is not a new tag
		if !isNewTag {
			newState.Event = GetUpdatedEvent(currentState.EpcState, currentState.Event, newTagEvent.EventType)
		}

		//Add to the location history only if the new tag event does not equal departed
		if newTagEvent.EventType != DepartedEvent {
			locationToAdd := tag.LocationHistory{
				Location:  newTagEvent.Location,
				Timestamp: newTagEvent.Timestamp,
				Source:    source}

			newState.LocationHistory = AddLocationIfNew(newState.LocationHistory, locationToAdd)
			// Go's Unix time is in seconds so convert the last read timestamp (milliseconds) to seconds
			newState.TTL = time.Unix(newState.LastRead/1000, 0)
		}

		//update epc state
		newState.EpcState = GetEpcState(currentState.EpcState, newState)
	}

	//if the new determined event results in departed then set the ttl to the const ttl for departed events
	if newState.Event == DepartedEvent {
		// Go's Unix time is in seconds so convert the last read timestamp (milliseconds) to seconds
		newState.TTL = time.Unix(newState.LastRead/1000, 0)
	}

	return newState
}

//GetNewTagEvent determines the event based on the event received
//from gateway.  Arrival and Departed are the only return value options
func GetNewTagEvent(eventType string) string {
	var newEventType string
	switch eventType {
	case MovedEvent, CycleCountEvent, ArrivalEvent, ReturnedEvent:
		newEventType = ArrivalEvent
	case DepartedEvent:
		newEventType = DepartedEvent
	}
	return newEventType
}

//GetUpdatedEvent determines event based on the current tag's even
//and what event was received from the gateway
func GetUpdatedEvent(currentEpcState string, currentEvent string, newEvent string) string {
	if (currentEpcState == DepartedEpcState && newEvent != DepartedEvent) || newEvent == ReturnedEvent {
		return ArrivalEvent
	}
	if len(newEvent) == 0 || newEvent == CycleCountEvent {
		return currentEvent
	}
	return newEvent
}

//GetEpcState determines the epc state value based on the event
//received from the gateway
func GetEpcState(currentEpcState string, newState tag.Tag) string {
	var epcState string
	switch newState.Event {
	case MovedEvent, CycleCountEvent, ArrivalEvent, ReturnedEvent:
		epcState = PresentEpcState
	case DepartedEvent:
		if currentEpcState != DepartedEpcState {
			epcState = DepartedEpcState
		} else {
			epcState = newState.EpcState
		}
	}
	return epcState
}

//AddLocationIfNew adds the location history to the array if that location history
//was not the last one added or updates the timestamp of the location if it was
//just added.  Maintains only a certain max number of items (MaxLocationHistory)
func AddLocationIfNew(locationHistory []tag.LocationHistory, locationToAdd tag.LocationHistory) []tag.LocationHistory {

	if len(locationHistory) == 0 || (len(locationHistory) > 0 && locationHistory[0].Location != locationToAdd.Location) {

		locationHistory = append([]tag.LocationHistory{
			locationToAdd},
			locationHistory...)

		if len(locationHistory) > MaxLocationHistory {
			locationHistory = locationHistory[:MaxLocationHistory]
		}
	} else if locationHistory[0].Location == locationToAdd.Location {
		locationHistory[0].Timestamp = locationToAdd.Timestamp
	}

	return locationHistory
}

func getBestLastRead(currentLastRead int64, newLastRead int64, currentSource string, newSource string) int64 {
	if currentSource == newSource && currentLastRead > newLastRead {
		return currentLastRead
	}

	return newLastRead
}
