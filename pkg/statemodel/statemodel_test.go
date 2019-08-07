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
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/integrationtest"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/jsonrpc"
	"testing"
	"time"

	"os"

	"github.com/pkg/errors"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	db "github.impcloud.net/RSP-Inventory-Suite/go-dbWrapper"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/tag"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
)

var existingTagTime = time.Now()
var newEventTagTime = existingTagTime.AddDate(0, 0, 1)

var dbHost integrationtest.DBHost

func TestMain(m *testing.M) {
	dbHost = integrationtest.InitHost("statemodel_test")
	os.Exit(m.Run())
}

func TestGetNewTagEventMoved(t *testing.T) {
	newTagEvent := GetNewTagEvent(MovedEvent)
	if newTagEvent != ArrivalEvent {
		t.Errorf("Failed. Expected %s, Received %s", ArrivalEvent, newTagEvent)
	}
}

func TestGetNewTagEventArrived(t *testing.T) {
	newTagEvent := GetNewTagEvent(ArrivalEvent)
	if newTagEvent != ArrivalEvent {
		t.Errorf("Failed. Expected %s, Received %s", ArrivalEvent, newTagEvent)
	}
}

func TestGetNewTagEventCycleCount(t *testing.T) {
	newTagEvent := GetNewTagEvent(CycleCountEvent)
	if newTagEvent != ArrivalEvent {
		t.Errorf("Failed. Expected %s, Received %s", ArrivalEvent, newTagEvent)
	}
}

func TestGetNewTagEventDeparted(t *testing.T) {
	newTagEvent := GetNewTagEvent(DepartedEvent)
	if newTagEvent != DepartedEvent {
		t.Errorf("Failed. Expected %s, Received %s", DepartedEvent, newTagEvent)
	}
}

func TestGetEpcStateMoved(t *testing.T) {
	gotTag := getHelperTag()
	gotTag.Event = MovedEvent
	epcState := GetEpcState(PresentEpcState, gotTag)
	if epcState != PresentEpcState {
		t.Errorf("Failed. Expected %s, Received %s", PresentEpcState, epcState)
	}
}

func TestGetEpcStateArrived(t *testing.T) {
	gotTag := getHelperTag()
	gotTag.Event = ArrivalEvent
	epcState := GetEpcState(PresentEpcState, gotTag)
	if epcState != PresentEpcState {
		t.Errorf("Failed. Expected %s, Received %s", PresentEpcState, epcState)
	}
}

func TestGetEpcStateCycleCount(t *testing.T) {
	gotTag := getHelperTag()
	gotTag.Event = CycleCountEvent
	epcState := GetEpcState(PresentEpcState, gotTag)
	if epcState != PresentEpcState {
		t.Errorf("Failed. Expected %s, Received %s", PresentEpcState, epcState)
	}
}

func TestGetEpcStateDeparted_PresentEpcState(t *testing.T) {
	gotTag := getHelperTag()
	gotTag.Event = DepartedEpcState
	epcState := GetEpcState(PresentEpcState, gotTag)
	if epcState != DepartedEpcState {
		t.Errorf("Failed. Expected %s, Received %s", DepartedEpcState, epcState)
	}
}

func TestGetEpcStateDeparted_DepartedEpcState(t *testing.T) {
	gotTag := getHelperTag()
	gotTag.Event = DepartedEpcState
	epcState := GetEpcState(DepartedEpcState, gotTag)
	if epcState != PresentEpcState {
		t.Errorf("Failed. Expected %s, Received %s", DepartedEpcState, epcState)
	}
}

func TestGetUpdatedEvent_NewEvent(t *testing.T) {
	newEvent := GetUpdatedEvent(PresentEpcState, ArrivalEvent, MovedEvent)
	if newEvent != MovedEvent {
		t.Errorf("Failed. Expected %s, Received %s", MovedEvent, newEvent)
	}
}

func TestGetUpdatedEvent_Departed_NotDeparted(t *testing.T) {
	newEvent := GetUpdatedEvent(DepartedEpcState, DepartedEvent, MovedEvent)
	if newEvent != ArrivalEvent {
		t.Errorf("Failed. Expected %s, Received %s", ArrivalEvent, newEvent)
	}
}

func TestGetUpdatedEvent_CycleCount(t *testing.T) {
	newEvent := GetUpdatedEvent(PresentEpcState, ArrivalEvent, CycleCountEvent)
	if newEvent != ArrivalEvent {
		t.Errorf("Failed. Expected %s, Received %s", ArrivalEvent, newEvent)
	}
}

func TestAddLocationIfNew(t *testing.T) {
	newLocationHistory := tag.LocationHistory{
		Location:  "old_location",
		Timestamp: helper.UnixMilliNow(),
		Source:    "fixed"}

	var newLocationHistoryArr []tag.LocationHistory
	locationHistoryArr := AddLocationIfNew(newLocationHistoryArr, newLocationHistory)
	if len(locationHistoryArr) != 1 {
		t.Errorf("Failed to set the location history of a new location history array")
	}
	if locationHistoryArr[0].Location != newLocationHistory.Location ||
		locationHistoryArr[0].Timestamp != newLocationHistory.Timestamp ||
		locationHistoryArr[0].Source != newLocationHistory.Source {
		t.Errorf("Location history to be set does not match what's in the array")
	}
}

func TestAddLocationIfNew_existingSameLocation(t *testing.T) {
	time1 := time.Now()
	time2 := time.Now().Add(time.Second * 5)

	oldLocationHistory := tag.LocationHistory{
		Location:  "old_location",
		Timestamp: helper.UnixMilli(time1),
	}

	newLocationHistory := tag.LocationHistory{
		Location:  "old_location",
		Timestamp: helper.UnixMilli(time2),
	}

	var newLocationHistoryArr []tag.LocationHistory
	newLocationHistoryArr = append(newLocationHistoryArr, []tag.LocationHistory{oldLocationHistory}...)

	locationHistoryArr := AddLocationIfNew(newLocationHistoryArr, newLocationHistory)
	if len(locationHistoryArr) > 1 {
		t.Errorf("Failed. Should not have added a new location history")
	}
	if locationHistoryArr[0].Timestamp != helper.UnixMilli(time2) {
		t.Errorf("Failed. Did not update timestamp of already existing location history")
	}
	if locationHistoryArr[0].Location != newLocationHistory.Location ||
		locationHistoryArr[0].Timestamp != newLocationHistory.Timestamp ||
		locationHistoryArr[0].Source != newLocationHistory.Source {
		t.Errorf("Location history to be set does not match what's in the array")
	}
}

func TestAddLocationIfNew_existingDifferentLocation(t *testing.T) {
	time1 := time.Now()
	time2 := time.Now().Add(time.Second * 5)

	oldLocationHistory := tag.LocationHistory{
		Location:  "old_location",
		Timestamp: helper.UnixMilli(time1),
	}
	newLocationHistory := tag.LocationHistory{
		Location:  "new_location",
		Timestamp: helper.UnixMilli(time2),
	}
	var newLocationHistoryArr []tag.LocationHistory
	newLocationHistoryArr = append(newLocationHistoryArr, []tag.LocationHistory{oldLocationHistory}...)

	locationHistoryArr := AddLocationIfNew(newLocationHistoryArr, newLocationHistory)
	if len(locationHistoryArr) != 2 {
		t.Errorf("Failed. Should not have added a new location history")
	}

	contains := false
	for _, location := range locationHistoryArr {
		if location.Location == "new_location" {
			contains = true
		}
	}

	if !contains {
		t.Errorf("Failed. Did not add new location to the location history")
	}
}

func TestAddLocationIfNew_10existingLocations(t *testing.T) {
	firstLocationHistory := tag.LocationHistory{
		Location:  "old_location_" + time.Now().String(),
		Timestamp: helper.UnixMilliNow(),
	}

	var newLocationHistoryArr []tag.LocationHistory

	for i := 0; i < 10; i++ {

		newLocationHistory := tag.LocationHistory{
			Location:  "old_location_" + time.Now().String(),
			Timestamp: helper.UnixMilliNow(),
		}

		newLocationHistoryArr = append(newLocationHistoryArr, []tag.LocationHistory{newLocationHistory}...)
	}

	locationHistoryArr := AddLocationIfNew(newLocationHistoryArr, firstLocationHistory)
	if len(locationHistoryArr) != MaxLocationHistory {
		t.Errorf("Failed. Expected Max Location History of %d, but length is %d", MaxLocationHistory, len(locationHistoryArr))
	}
}

func TestGetUpdateState_NewTag(t *testing.T) {
	tagState := UpdateTag(tag.Tag{}, getHelperTagEvent(), "fixed")
	if tagState.QualifiedState != UnknownQualifiedState {
		t.Errorf("Failed. Expected %s, received %s", UnknownQualifiedState, tagState.QualifiedState)
	}
}

func TestUpdateTag_HHPriorityOlderFixed(t *testing.T) {

	config.AppConfig.NewerHandheldHavePriority = true

	currentTagState := getHelperTag()
	currentTagState.Source = "handheld"
	currentTagState.FacilityID = "Zebra XYZ"
	currentTagState.LastRead = helper.UnixMilli(existingTagTime)

	newTagEvent := getHelperTagEvent()
	newTagEvent.Timestamp = helper.UnixMilli(existingTagTime.AddDate(0, 0, -1))

	source := "fixed"

	tagState := UpdateTag(currentTagState, newTagEvent, source)

	if tagState.Source != currentTagState.Source {
		t.Error("tagState Source should not have changed")
	}

	if tagState.FacilityID != currentTagState.FacilityID {
		t.Error("tagState FacilityID should not have changed")
	}

	if tagState.LastRead != currentTagState.LastRead {
		t.Error("tagState LastRead should not have changed")
	}
}

func TestUpdateTag_HHPriorityNewerFixed(t *testing.T) {
	// HH has priority, but newer fixed tag will overwrite.
	config.AppConfig.NewerHandheldHavePriority = true

	currentTagState := getHelperTag()
	currentTagState.Source = "handheld"
	currentTagState.FacilityID = "Zebra XYZ"
	currentTagState.LastRead = helper.UnixMilli(existingTagTime.AddDate(0, 0, -1))

	newTagEvent := getHelperTagEvent()
	newTagEvent.Timestamp = helper.UnixMilli(existingTagTime)

	source := "fixed"
	tagState := UpdateTag(currentTagState, newTagEvent, source)

	if tagState.Source != source {
		t.Error("tagState Source should have changed")
	}

	if tagState.FacilityID != newTagEvent.FacilityID {
		t.Error("tagState FacilityID should have changed")
	}

	if tagState.LastRead != newTagEvent.Timestamp {
		t.Error("tagState LastRead should have changed")
	}
}

func TestUpdateTag_NotHHPriorityOlderFixed(t *testing.T) {
	// HH has no priority, so any fixed tag will overwrite.

	config.AppConfig.NewerHandheldHavePriority = false

	currentTagState := getHelperTag()
	currentTagState.Source = "handheld"
	currentTagState.FacilityID = "Zebra XYZ"
	currentTagState.LastRead = helper.UnixMilli(existingTagTime)

	newTagEvent := getHelperTagEvent()
	newTagEvent.Timestamp = helper.UnixMilli(existingTagTime.AddDate(0, 0, -1))

	source := "fixed"
	tagState := UpdateTag(currentTagState, newTagEvent, source)

	if tagState.Source != source {
		t.Error("tagState Source should have changed")
	}

	if tagState.FacilityID != newTagEvent.FacilityID {
		t.Error("tagState FacilityID should have changed")
	}

	if tagState.LastRead != newTagEvent.Timestamp {
		t.Error("tagState LastRead should have changed")
	}
}

func TestUpdateTag_NotHHPriorityNewerFixed(t *testing.T) {
	// HH has no priority, so any fixed tag will overwrite.

	config.AppConfig.NewerHandheldHavePriority = false

	currentTagState := getHelperTag()
	currentTagState.Source = "handheld"
	currentTagState.FacilityID = "Zebra XYZ"
	currentTagState.LastRead = helper.UnixMilli(existingTagTime.AddDate(0, 0, -1))

	newTagEvent := getHelperTagEvent()
	newTagEvent.Timestamp = helper.UnixMilli(existingTagTime)

	source := "fixed"
	tagState := UpdateTag(currentTagState, newTagEvent, source)

	if tagState.Source != source {
		t.Error("tagState Source should have changed")
	}

	if tagState.FacilityID != newTagEvent.FacilityID {
		t.Error("tagState FacilityID should have changed")
	}

	if tagState.LastRead != newTagEvent.Timestamp {
		t.Error("tagState LastRead should have changed")
	}
}

func TestGetUpdateState_DepartedTag_ValidEPC(t *testing.T) {

	currentTagState := getHelperTag()
	currentTagState.Event = "departed"

	newTagEvent := getHelperTagEvent()
	newTagEvent.EventType = "departed"
	newTagEvent.FacilityID = "NewFacility"

	tagState := UpdateTag(currentTagState, newTagEvent, "fixed")

	// Check if facilityID was updated
	if currentTagState.FacilityID == tagState.FacilityID {
		t.Error("tagState FacilityID must be different from currentTagState")
	}

	if tagState.TTL.Unix() != time.Unix(tagState.LastRead/1000, 0).Unix() {
		t.Errorf("Failed: TTL value incorrect")
	}
	//Empty Qualified State field sent so should be set to unknown
	if tagState.QualifiedState != UnknownQualifiedState {
		t.Errorf("Failed. Expected %s, received %s", UnknownQualifiedState, tagState.QualifiedState)
	}
}

func TestLastReadTTLSyncedFixedNewLastRead(t *testing.T) {
	// Default values created are new last read > current last read
	currentTagState := getHelperTag()
	newTagEvent := getHelperTagEvent()

	tagState := UpdateTag(currentTagState, newTagEvent, "fixed")

	if tagState.LastRead != newTagEvent.Timestamp {
		t.Errorf("Failed: LastRead not set to new value")
	}

	if tagState.TTL.Unix() != time.Unix(tagState.LastRead/1000, 0).Unix() {
		t.Errorf("Failed: TTL value incorrect")
	}
}

func TestLastReadTTLSyncedFixedOldLastRead(t *testing.T) {
	// Default values created are new last read > current last read
	currentTagState := getHelperTag()
	newTagEvent := getHelperTagEvent()
	// Make new tag have older last read
	newTagEvent.Timestamp = helper.UnixMilli(existingTagTime.AddDate(0, 0, -1))

	tagState := UpdateTag(currentTagState, newTagEvent, "fixed")

	if tagState.LastRead != currentTagState.LastRead {
		t.Errorf("Failed: LastRead not set to old value")
	}

	if tagState.TTL.Unix() != time.Unix(tagState.LastRead/1000, 0).Unix() {
		t.Errorf("Failed: TTL value incorrect")
	}
}

func TestLastReadTTLSyncedNewFixedOldHandheld(t *testing.T) {
	// Default values created are new last read > current last read
	currentTagState := getHelperTag()
	newTagEvent := getHelperTagEvent()
	newTagEvent.Timestamp = helper.UnixMilli(existingTagTime.Add(-1))

	// Note that source of new tag is "handheld". Default for current is 'fixed"
	tagState := UpdateTag(currentTagState, newTagEvent, "handheld")

	if tagState.LastRead != newTagEvent.Timestamp {
		t.Errorf("Failed: LastRead not set to new value")
	}

	if tagState.TTL.Unix() != time.Unix(tagState.LastRead/1000, 0).Unix() {
		t.Errorf("Failed: TTL value incorrect")
	}
}

func TestIsTagWhitelisted_False(t *testing.T) {

	tagEvent := getHelperTagEvent()
	whitelisted := IsTagWhitelisted(tagEvent.EpcCode, []string{"300,301"})

	if whitelisted {
		t.Errorf("Failed. Expected whitelist to be false, received true")
	}
}

func TestIsTagWhitelisted_True(t *testing.T) {

	tagEvent := getHelperTagEvent()
	config.AppConfig.EpcFilters = []string{"301"}
	tagEvent.EpcCode = "301402662C3A5F904C19939D"
	config.AppConfig.EpcFilters = append(config.AppConfig.EpcFilters, tagEvent.EpcCode)
	whitelisted := IsTagWhitelisted(tagEvent.EpcCode, config.AppConfig.EpcFilters)

	if !whitelisted {
		t.Errorf("Failed. Expected whitelist to be true, received false")
	}
}

func TestFindTagByEpc(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	insertSample(t, dbs, getHelperTag())
	foundTag, err := tag.FindByEpc(dbs, getHelperTag().Epc)
	if err != nil {
		t.Errorf("Failed.  Problem calling tag.FindByEpc")
	}
	if foundTag.Epc != getHelperTag().Epc {
		t.Errorf("Failed. Did not retrieve the expected tag.")
	}
	err = tag.Delete(dbs, getHelperTag().Epc)
	if err != nil {
		t.Error("Error on Delete")
	}
}

//The following tests are functional to validate the expected
//states based on the qualified state model
//nolint :gocyclo
func TestArrived_New(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	helperTag := getHelperTag()
	helperTagEvent := getHelperTagEvent()
	foundTag, err := tag.FindByEpc(dbs, helperTag.Epc)

	if err != nil {
		t.Errorf("Failed.  Problem calling tag.FindByEpc")
	}
	if !foundTag.IsEmpty() && !foundTag.IsShippingNoticeEntry() {
		t.Errorf("Failed. Tag should not exist in DB.")
	}

	updatedTag := UpdateTag(foundTag, helperTagEvent, "fixed")

	if updatedTag.Arrived != helperTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with no Arrived Tag in DB.")
	}
	if updatedTag.LastRead != helperTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with no Arrived Tag in DB.")
	}
	if updatedTag.Event != ArrivalEvent {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with no Arrived Tag in DB.")
	}
	if updatedTag.EpcState != PresentEpcState {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with no Arrived Tag in DB.")
	}
	if updatedTag.QualifiedState != UnknownQualifiedState {
		t.Errorf("Updated Tag failed. Expect %s, Received %s", UnknownQualifiedState, updatedTag.QualifiedState)
	}
	if len(updatedTag.LocationHistory) != 1 {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with no Arrived Tag in DB.")
	}
	if updatedTag.LocationHistory[0].Location != helperTagEvent.Location {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with no Arrived Tag in DB.")
	}
	if updatedTag.LocationHistory[0].Timestamp != helperTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with no Arrived Tag in DB.")
	}

	// Last read is in milliseconds and TTL is Seconds precision so have to convert last read to seconds to compare
	if updatedTag.TTL.Unix() != updatedTag.LastRead/1000 {
		t.Errorf("Updated Tag failed state changes for TTL. TTL=%d vs LastRead=%d", updatedTag.TTL.Unix(), updatedTag.LastRead)
	}

	tag.Delete(dbs, getHelperTag().Epc)
}

//nolint :goclyclo
func TestArrived_ExistPresent(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	helperTag := getHelperTag()
	helperTagEvent := getHelperTagEvent()
	insertSample(t, dbs, helperTag)
	foundTag, err := tag.FindByEpc(dbs, helperTag.Epc)

	if err != nil {
		t.Errorf("Failed.  Problem calling tag.FindByEpc")
	}
	if foundTag.IsEmpty() || foundTag.IsShippingNoticeEntry() {
		t.Errorf("Failed. Tag should exist in DB.")
	}

	updatedTag := UpdateTag(foundTag, helperTagEvent, "fixed")

	if updatedTag.Arrived != helperTag.Arrived {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.Event != ArrivalEvent {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.EpcState != helperTag.EpcState {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.LastRead != helperTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.QualifiedState == "" {
		t.Errorf("Updated Tag failed. Expect %s, Received %s", UnknownQualifiedState, updatedTag.QualifiedState)
	}
	if len(updatedTag.LocationHistory) != 1 {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.LocationHistory[0].Location != helperTagEvent.Location {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.LocationHistory[0].Timestamp != helper.UnixMilli(newEventTagTime) {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}

	tag.Delete(dbs, getHelperTag().Epc)
}

//nolint :goclyclo
func TestArrived_ExistDeparted(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	helperTag := getHelperTag()
	helperTag.EpcState = DepartedEpcState
	helperTagEvent := getHelperTagEvent()
	insertSample(t, dbs, helperTag)
	foundTag, err := tag.FindByEpc(dbs, helperTag.Epc)

	if err != nil {
		t.Errorf("Failed.  Problem calling tag.FindByEpc")
	}
	if foundTag.IsEmpty() || foundTag.IsShippingNoticeEntry() {
		t.Errorf("Failed. Tag should exist in DB.")
	}

	updatedTag := UpdateTag(foundTag, helperTagEvent, "fixed")

	if updatedTag.Arrived != helperTag.Arrived {
		t.Errorf("Updated Tag failed state changes for a arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.Event != ArrivalEvent {
		t.Errorf("Updated Tag failed state changes for a arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.EpcState != PresentEpcState {
		t.Errorf("Updated Tag failed state changes for a arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.LastRead != helperTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for a arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.QualifiedState == "" {
		t.Errorf("Updated Tag failed. Expect %s, Received %s", UnknownQualifiedState, updatedTag.QualifiedState)
	}
	if len(updatedTag.LocationHistory) != 1 {
		t.Errorf("Updated Tag failed state changes for a arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.LocationHistory[0].Location != helperTagEvent.Location {
		t.Errorf("Updated Tag failed state changes for a arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.LocationHistory[0].Timestamp != helper.UnixMilli(newEventTagTime) {
		t.Errorf("Updated Tag failed state changes for a arrived RSP Controller event with existing Arrived Tag in DB.")
	}

	if err := tag.Delete(dbs, helperTag.Epc); err != nil {
		t.Error(err)
	}
}

//nolint :gocyclo
func TestMoved_New(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	foundTag, err := tag.FindByEpc(dbs, getHelperTag().Epc)
	if err != nil {
		t.Errorf("Failed.  Problem calling tag.FindByEpc")
	}
	if !foundTag.IsEmpty() && !foundTag.IsShippingNoticeEntry() {
		t.Errorf("Failed. Tag should not exist in DB.")
	}

	movedTagEvent := getHelperTagEvent()
	movedTagEvent.EventType = MovedEvent

	updatedTag := UpdateTag(foundTag, movedTagEvent, "fixed")

	if updatedTag.Arrived != movedTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for a moved event from RSP Controller and no Tag in DB.")
	}
	if updatedTag.Event != ArrivalEvent {
		t.Errorf("Updated Tag failed state changes for a moved event from RSP Controller and no Tag in DB.")
	}
	if updatedTag.EpcState != PresentEpcState {
		t.Errorf("Updated Tag failed state changes for a moved event from RSP Controller and no Tag in DB.")
	}
	if updatedTag.LastRead != movedTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for a moved event from RSP Controller and no Tag in DB.")
	}
	if updatedTag.QualifiedState != UnknownQualifiedState {
		t.Errorf("Updated Tag failed. Expect %s, Received %s", UnknownQualifiedState, updatedTag.QualifiedState)
	}
	if len(updatedTag.LocationHistory) != 1 {
		t.Errorf("Updated Tag failed state changes for a moved event from RSP Controller and no Tag in DB.")
	}
	if updatedTag.LocationHistory[0].Location != movedTagEvent.Location {
		t.Errorf("Updated Tag failed state changes for a moved event from RSP Controller and no Tag in DB.")
	}
	if updatedTag.LocationHistory[0].Timestamp != movedTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for a moved event from RSP Controller and no Tag in DB.")
	}

	tag.Delete(dbs, getHelperTag().Epc)
}

//nolint :goclyclo
func TestMoved_ExistPresent(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	movedTag := getHelperTag()
	movedTag.Event = MovedEvent
	insertSample(t, dbs, movedTag)

	foundTag, err := tag.FindByEpc(dbs, movedTag.Epc)
	if err != nil {
		t.Errorf("Failed.  Problem calling tag.FindByEpc")
	}
	if foundTag.IsEmpty() || foundTag.IsShippingNoticeEntry() {
		t.Errorf("Failed. Tag should exist in DB.")
	}

	movedTagEvent := getHelperTagEvent()
	movedTagEvent.EventType = MovedEvent

	updatedTag := UpdateTag(foundTag, movedTagEvent, "fixed")

	if updatedTag.Arrived != movedTag.Arrived {
		t.Errorf("Updated Tag failed state changes for a moved RSP Controller event with existing Moved Tag in DB.")
	}
	if updatedTag.Event != MovedEvent {
		t.Errorf("Updated Tag failed state changes for a moved RSP Controller event with existing Moved Tag in DB.")
	}
	if updatedTag.EpcState != movedTag.EpcState {
		t.Errorf("Updated Tag failed state changes for a moved RSP Controller event with existing Moved Tag in DB.")
	}
	if updatedTag.LastRead != movedTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for a moved RSP Controller event with existing Moved Tag in DB.")
	}
	if updatedTag.QualifiedState == "" {
		t.Errorf("Updated Tag failed. Expect %s, Received %s", UnknownQualifiedState, updatedTag.QualifiedState)
	}
	if len(updatedTag.LocationHistory) != 1 {
		t.Errorf("Updated Tag failed state changes for a moved RSP Controller event with existing Moved Tag in DB.")
	}
	if updatedTag.LocationHistory[0].Location != movedTagEvent.Location {
		t.Errorf("Updated Tag failed state changes for a moved RSP Controller event with existing Moved Tag in DB.")
	}
	if updatedTag.LocationHistory[0].Timestamp != movedTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for a moved RSP Controller event with existing Moved Tag in DB.")
	}

	tag.Delete(dbs, getHelperTag().Epc)
}

// nolint :dupl
func TestMoved_ExistDeparted(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	movedTag := getHelperTag()
	movedTag.Event = MovedEvent
	movedTag.EpcState = DepartedEpcState
	insertSample(t, dbs, movedTag)
	foundTag, err := tag.FindByEpc(dbs, movedTag.Epc)

	if err != nil {
		t.Errorf("Failed.  Problem calling tag.FindByEpc")
	}
	if foundTag.IsEmpty() || foundTag.IsShippingNoticeEntry() {
		t.Errorf("Failed. Tag should exist in DB.")
	}

	movedTagEvent := getHelperTagEvent()
	movedTagEvent.EventType = MovedEvent

	updatedTag := UpdateTag(foundTag, movedTagEvent, "fixed")

	if updatedTag.Arrived != movedTag.Arrived {
		t.Errorf("Updated Tag failed state changes for a moved RSP Controller event with existing Moved Tag in DB.")
	}
	if updatedTag.Event != ArrivalEvent {
		t.Errorf("Updated Tag failed state changes for a moved RSP Controller event with existing Moved Tag in DB.")
	}
	if updatedTag.EpcState != PresentEpcState {
		t.Errorf("Updated Tag failed state changes for a moved RSP Controller event with existing Moved Tag in DB.")
	}
	if updatedTag.LastRead != movedTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for a moved RSP Controller event with existing Moved Tag in DB.")
	}
	if updatedTag.QualifiedState == "" {
		t.Errorf("Updated Tag failed. Expect %s, Received %s", UnknownQualifiedState, updatedTag.QualifiedState)
	}
	if len(updatedTag.LocationHistory) != 1 {
		t.Errorf("Updated Tag failed state changes for a moved RSP Controller event with existing Moved Tag in DB.")
	}
	if updatedTag.LocationHistory[0].Location != movedTagEvent.Location {
		t.Errorf("Updated Tag failed state changes for a moved RSP Controller event with existing Moved Tag in DB.")
	}
	if updatedTag.LocationHistory[0].Timestamp != movedTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for a moved RSP Controller event with existing Moved Tag in DB.")
	}

	tag.Delete(dbs, getHelperTag().Epc)
}

//nolint :gocyclo
func TestCycleCount_New(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	cycleCountTag := getHelperTag()
	cycleCountTag.Event = CycleCountEvent
	foundTag, err := tag.FindByEpc(dbs, cycleCountTag.Epc)

	if err != nil {
		t.Errorf("Failed.  Problem calling tag.FindByEpc")
	}
	if !foundTag.IsEmpty() && !foundTag.IsShippingNoticeEntry() {
		t.Errorf("Failed. Tag should not exist in DB.")
	}

	cycleCountTagEvent := getHelperTagEvent()
	cycleCountTagEvent.EventType = CycleCountEvent

	updatedTag := UpdateTag(foundTag, cycleCountTagEvent, "fixed")

	if updatedTag.Arrived != cycleCountTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for a cycle count event from RSP Controller and no Tag in DB.")
	}
	if updatedTag.Event != ArrivalEvent {
		t.Errorf("Updated Tag failed state changes for a cycle count event from RSP Controller and no Tag in DB.")
	}
	if updatedTag.EpcState != PresentEpcState {
		t.Errorf("Updated Tag failed state changes for a cycle count event from RSP Controller and no Tag in DB.")
	}
	if updatedTag.QualifiedState != UnknownQualifiedState {
		t.Errorf("Updated Tag failed. Expect %s, Received %s", UnknownQualifiedState, updatedTag.QualifiedState)
	}
	if updatedTag.LastRead != cycleCountTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for a cycle count event from RSP Controller and no Tag in DB.")
	}
	if len(updatedTag.LocationHistory) != 1 {
		t.Errorf("Updated Tag failed state changes for a cycle count event from RSP Controller and no Tag in DB.")
	}
	if updatedTag.LocationHistory[0].Location != cycleCountTagEvent.Location {
		t.Errorf("Updated Tag failed state changes for a cycle count event from RSP Controller and no Tag in DB.")
	}
	if updatedTag.LocationHistory[0].Timestamp != cycleCountTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for a cycle count event from RSP Controller and no Tag in DB.")
	}

	tag.Delete(dbs, getHelperTag().Epc)
}

//nolint :goclyclo
func TestCycleCount_ExistPresent(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	cycleCountTag := getHelperTag()
	cycleCountTag.Event = CycleCountEvent
	insertSample(t, dbs, cycleCountTag)
	foundTag, err := tag.FindByEpc(dbs, cycleCountTag.Epc)

	if err != nil {
		t.Errorf("Failed.  Problem calling tag.FindByEpc")
	}
	if foundTag.IsEmpty() || foundTag.IsShippingNoticeEntry() {
		t.Errorf("Failed. Tag should exist in DB.")
	}

	cycleCountTagEvent := getHelperTagEvent()
	cycleCountTagEvent.EventType = CycleCountEvent

	updatedTag := UpdateTag(foundTag, cycleCountTagEvent, "fixed")

	if updatedTag.Arrived != cycleCountTag.Arrived {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}

	if updatedTag.Event != cycleCountTag.Event {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}

	if updatedTag.EpcState != cycleCountTag.EpcState {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}

	if updatedTag.LastRead != cycleCountTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}

	if updatedTag.QualifiedState == "" {
		t.Errorf("Updated Tag failed. Expect %s, Received %s", UnknownQualifiedState, updatedTag.QualifiedState)
	}

	if len(updatedTag.LocationHistory) != 1 {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}

	if updatedTag.QualifiedState == "" {
		t.Errorf("Updated Tag failed. Expect %s, Received %s", UnknownQualifiedState, updatedTag.QualifiedState)
	}

	if updatedTag.LocationHistory[0].Location != cycleCountTagEvent.Location {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}

	if updatedTag.LocationHistory[0].Timestamp != cycleCountTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}

	tag.Delete(dbs, getHelperTag().Epc)
}

// nolint :dupl
func TestCycleCount_ExistDeparted(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	cycleCountTag := getHelperTag()
	cycleCountTag.Event = CycleCountEvent
	cycleCountTag.EpcState = DepartedEpcState
	insertSample(t, dbs, cycleCountTag)
	foundTag, err := tag.FindByEpc(dbs, cycleCountTag.Epc)

	if err != nil {
		t.Errorf("Failed.  Problem calling tag.FindByEpc")
	}
	if foundTag.IsEmpty() || foundTag.IsShippingNoticeEntry() {
		t.Errorf("Failed. Tag should exist in DB.")
	}

	cycleCountTagEvent := getHelperTagEvent()
	cycleCountTagEvent.EventType = CycleCountEvent

	updatedTag := UpdateTag(foundTag, cycleCountTagEvent, "fixed")

	if updatedTag.Arrived != cycleCountTag.Arrived {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.Event != ArrivalEvent {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.EpcState != PresentEpcState {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.LastRead != cycleCountTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if len(updatedTag.LocationHistory) != 1 {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.QualifiedState == "" {
		t.Errorf("Updated Tag failed. Expect %s, Received %s", UnknownQualifiedState, updatedTag.QualifiedState)
	}
	if updatedTag.LocationHistory[0].Location != cycleCountTagEvent.Location {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.LocationHistory[0].Timestamp != cycleCountTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}

	tag.Delete(dbs, getHelperTag().Epc)
}

//nolint :gocyclo
func TestDeparted_New(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	departedTag := getHelperTag()
	departedTag.Event = DepartedEvent
	foundTag, err := tag.FindByEpc(dbs, departedTag.Epc)

	if err != nil {
		t.Errorf("Failed.  Problem calling tag.FindByEpc")
	}
	if !foundTag.IsEmpty() && !foundTag.IsShippingNoticeEntry() {
		t.Errorf("Failed. Tag should not exist in DB.")
	}

	departedTagEvent := getHelperTagEvent()
	departedTagEvent.EventType = DepartedEvent
	departedTagEvent.Location = "Departed Location"

	updatedTag := UpdateTag(foundTag, departedTagEvent, "fixed")

	if updatedTag.LocationHistory == nil {
		t.Errorf("Updated Tag failed state changes due to LocationHistory nil instead of [].")
	}
	if updatedTag.Arrived != departedTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for a cycle count event from RSP Controller and no Tag in DB.")
	}
	if updatedTag.Event != DepartedEvent {
		t.Errorf("Updated Tag failed state changes for a cycle count event from RSP Controller and no Tag in DB.")
	}
	if updatedTag.EpcState != DepartedEpcState {
		t.Errorf("Updated Tag failed state changes for a cycle count event from RSP Controller and no Tag in DB.")
	}
	if updatedTag.LastRead != departedTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for a cycle count event from RSP Controller and no Tag in DB.")
	}
	if updatedTag.QualifiedState != UnknownQualifiedState {
		t.Errorf("Updated Tag failed. Expect %s, Received %s", UnknownQualifiedState, updatedTag.QualifiedState)
	}
	if len(updatedTag.LocationHistory) != 0 {
		t.Errorf("Updated Tag failed state changes for a cycle count event from RSP Controller and no Tag in DB.")
	}

	tag.Delete(dbs, getHelperTag().Epc)
}

//nolint :goclyclo
func TestDeparted_ExistPresent(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	departedTag := getHelperTag()
	departedTag.Event = DepartedEvent
	insertSample(t, dbs, departedTag)
	foundTag, err := tag.FindByEpc(dbs, departedTag.Epc)

	if err != nil {
		t.Errorf("Failed.  Problem calling tag.FindByEpc")
	}
	if foundTag.IsEmpty() || foundTag.IsShippingNoticeEntry() {
		t.Errorf("Failed. Tag should exist in DB.")
	}

	departedTagEvent := getHelperTagEvent()
	departedTagEvent.EventType = DepartedEvent

	updatedTag := UpdateTag(foundTag, departedTagEvent, "fixed")

	if updatedTag.Arrived != departedTag.Arrived {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.Event != DepartedEvent {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.EpcState != DepartedEpcState {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.LastRead != departedTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.QualifiedState == "" {
		t.Errorf("Updated Tag failed. Expect %s, Received %s", UnknownQualifiedState, updatedTag.QualifiedState)
	}
	if len(updatedTag.LocationHistory) != 1 {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.LocationHistory[0].Location != departedTag.LocationHistory[0].Location {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.LocationHistory[0].Timestamp != departedTag.LocationHistory[0].Timestamp {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}

	tag.Delete(dbs, getHelperTag().Epc)
}

//nolint :goclyclo
func TestDeparted_ExistDeparted(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	departedTag := getHelperTag()
	departedTag.Event = ArrivalEvent
	departedTag.EpcState = DepartedEpcState
	insertSample(t, dbs, departedTag)
	foundTag, err := tag.FindByEpc(dbs, departedTag.Epc)

	if err != nil {
		t.Errorf("Failed.  Problem calling tag.FindByEpc")
	}
	if foundTag.IsEmpty() || foundTag.IsShippingNoticeEntry() {
		t.Errorf("Failed. Tag should exist in DB.")
	}

	departedTagEvent := getHelperTagEvent()
	departedTagEvent.EventType = DepartedEvent

	updatedTag := UpdateTag(departedTag, departedTagEvent, "fixed")

	if updatedTag.Arrived != departedTag.Arrived {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.Event != departedTag.Event {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.EpcState != departedTag.EpcState {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.LastRead != departedTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.QualifiedState == "" {
		t.Errorf("Updated Tag failed. Expect %s, Received %s", UnknownQualifiedState, updatedTag.QualifiedState)
	}
	if len(updatedTag.LocationHistory) != 1 {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.LocationHistory[0].Location != departedTag.LocationHistory[0].Location {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.LocationHistory[0].Timestamp != departedTag.LocationHistory[0].Timestamp {
		t.Errorf("Updated Tag failed state changes for an arrived RSP Controller event with existing Arrived Tag in DB.")
	}

	tag.Delete(dbs, getHelperTag().Epc)
}

//nolint :gocyclo
func TestReturned_New(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	helperTag := getHelperTag()
	helperTagEvent := getHelperTagEvent()
	helperTagEvent.EventType = ReturnedEvent
	foundTag, err := tag.FindByEpc(dbs, helperTag.Epc)

	if err != nil {
		t.Errorf("Failed.  Problem calling tag.FindByEpc")
	}
	if !foundTag.IsEmpty() && !foundTag.IsShippingNoticeEntry() {
		t.Errorf("Failed. Tag should not exist in DB.")
	}

	updatedTag := UpdateTag(foundTag, helperTagEvent, "fixed")

	if updatedTag.Arrived != helperTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for an returned RSP Controller event with no Arrived Tag in DB.")
	}
	if updatedTag.LastRead != helperTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for an returned RSP Controller event with no Arrived Tag in DB.")
	}
	if updatedTag.Event != ArrivalEvent {
		t.Errorf("Updated Tag failed state changes for an returned RSP Controller event with no Arrived Tag in DB.")
	}
	if updatedTag.EpcState != PresentEpcState {
		t.Errorf("Updated Tag failed state changes for an returned RSP Controller event with no Arrived Tag in DB.")
	}
	if updatedTag.QualifiedState != UnknownQualifiedState {
		t.Errorf("Updated Tag failed. Expect %s, Received %s", UnknownQualifiedState, updatedTag.QualifiedState)
	}
	if len(updatedTag.LocationHistory) != 1 {
		t.Errorf("Updated Tag failed state changes for an returned RSP Controller event with no Arrived Tag in DB.")
	}
	if updatedTag.LocationHistory[0].Location != helperTagEvent.Location {
		t.Errorf("Updated Tag failed state changes for an returned RSP Controller event with no Arrived Tag in DB.")
	}
	if updatedTag.LocationHistory[0].Timestamp != helperTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for an returned RSP Controller event with no Arrived Tag in DB.")
	}

	// Last read is in milliseconds and TTL is Seconds precision so have to convert last read to seconds to compare
	if updatedTag.TTL.Unix() != updatedTag.LastRead/1000 {
		t.Errorf("Updated Tag failed state changes for TTL. TTL=%d vs LastRead=%d", updatedTag.TTL.Unix(), updatedTag.LastRead)
	}

	tag.Delete(dbs, getHelperTag().Epc)
}

//nolint :goclyclo
func TestReturned_ExistPresent(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	helperTag := getHelperTag()
	helperTagEvent := getHelperTagEvent()
	helperTagEvent.EventType = ReturnedEvent
	insertSample(t, dbs, helperTag)
	foundTag, err := tag.FindByEpc(dbs, helperTag.Epc)

	if err != nil {
		t.Errorf("Failed.  Problem calling tag.FindByEpc")
	}
	if foundTag.IsEmpty() || foundTag.IsShippingNoticeEntry() {
		t.Errorf("Failed. Tag should exist in DB.")
	}

	updatedTag := UpdateTag(foundTag, helperTagEvent, "fixed")

	if updatedTag.Arrived != helperTag.Arrived { //No Change
		t.Errorf("Updated Tag failed state changes for an returned RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.Event != ArrivalEvent {
		t.Errorf("Updated Tag failed state changes for an returned RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.EpcState != helperTag.EpcState {
		t.Errorf("Updated Tag failed state changes for an returned RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.LastRead != helperTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for an returned RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.QualifiedState == "" {
		t.Errorf("Updated Tag failed. Expect %s, Received %s", UnknownQualifiedState, updatedTag.QualifiedState)
	}
	if len(updatedTag.LocationHistory) != 1 {
		t.Errorf("Updated Tag failed state changes for an returned RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.LocationHistory[0].Location != helperTagEvent.Location {
		t.Errorf("Updated Tag failed state changes for an returned RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.LocationHistory[0].Timestamp != helper.UnixMilli(newEventTagTime) {
		t.Errorf("Updated Tag failed state changes for an returned RSP Controller event with existing Arrived Tag in DB.")
	}

	tag.Delete(dbs, getHelperTag().Epc)
}

//nolint :goclyclo
func TestReturned_ExistDeparted(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	helperTag := getHelperTag()
	helperTag.EpcState = DepartedEpcState
	helperTagEvent := getHelperTagEvent()
	helperTagEvent.EventType = ReturnedEvent
	insertSample(t, dbs, helperTag)
	foundTag, err := tag.FindByEpc(dbs, helperTag.Epc)

	if err != nil {
		t.Errorf("Failed.  Problem calling tag.FindByEpc")
	}
	if foundTag.IsEmpty() || foundTag.IsShippingNoticeEntry() {
		t.Errorf("Failed. Tag should exist in DB.")
	}

	updatedTag := UpdateTag(foundTag, helperTagEvent, "fixed")

	if updatedTag.Arrived != helperTag.Arrived {
		t.Errorf("Updated Tag failed state changes for a arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.Event != ArrivalEvent {
		t.Errorf("Updated Tag failed state changes for a arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.EpcState != PresentEpcState {
		t.Errorf("Updated Tag failed state changes for a arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.LastRead != helperTagEvent.Timestamp {
		t.Errorf("Updated Tag failed state changes for a arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.QualifiedState == "" {
		t.Errorf("Updated Tag failed. Expect %s, Received %s", UnknownQualifiedState, updatedTag.QualifiedState)
	}
	if len(updatedTag.LocationHistory) != 1 {
		t.Errorf("Updated Tag failed state changes for a arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.LocationHistory[0].Location != helperTagEvent.Location {
		t.Errorf("Updated Tag failed state changes for a arrived RSP Controller event with existing Arrived Tag in DB.")
	}
	if updatedTag.LocationHistory[0].Timestamp != helper.UnixMilli(newEventTagTime) {
		t.Errorf("Updated Tag failed state changes for a arrived RSP Controller event with existing Arrived Tag in DB.")
	}

	tag.Delete(dbs, helperTag.Epc)
}

func TestGetBestLastRead(t *testing.T) {
	currentLastRead := int64(1516684230167)
	currentSource := "fixed"
	newLastRead := int64(1516684230000)
	newSource := "fixed"

	expected := currentLastRead
	actual := getBestLastRead(currentLastRead, newLastRead, currentSource, newSource)

	if actual != expected {
		t.Error("getBestLastRead failed to return currentLastRead")
	}

	newSource = "handheld"
	expected = newLastRead
	actual = getBestLastRead(currentLastRead, newLastRead, currentSource, newSource)

	if actual != expected {
		t.Error("getBestLastRead failed to return newLastRead")
	}

	newLastRead = int64(1516684239999)
	newSource = "fixed"
	expected = newLastRead
	actual = getBestLastRead(currentLastRead, newLastRead, currentSource, newSource)

	if actual != expected {
		t.Error("getBestLastRead failed to return newLastRead")
	}
}

func getHelperTag() tag.Tag {
	locationHistory := tag.LocationHistory{
		Location:  "Front",
		Timestamp: helper.UnixMilli(existingTagTime)}

	var locationHistories []tag.LocationHistory
	locationHistories = append(locationHistories, []tag.LocationHistory{locationHistory}...)

	return tag.Tag{
		Arrived:         helper.UnixMilli(existingTagTime),
		EpcEncodeFormat: "tbd",
		Epc:             "303402662C3A5F904C19939D",
		EpcState:        "present",
		Event:           "arrived",
		FacilityID:      "TestFacility",
		ProductID:       "",
		Source:          "fixed",
		LastRead:        helper.UnixMilli(existingTagTime),
		LocationHistory: locationHistories,
		TTL:             time.Unix(helper.UnixMilli(existingTagTime)/1000, 0)}
}

func getHelperTagEvent() jsonrpc.TagEvent {

	return jsonrpc.TagEvent{
		EpcEncodeFormat: "tbd",
		EpcCode:         "303402662C3A5F904C19939E",
		EventType:       "arrival",
		FacilityID:      "TestFacility",
		Tid:             "",
		Location:        "Front",
		Timestamp:       helper.UnixMilli(newEventTagTime)}
}

func insertSample(t *testing.T, mydb *db.DB, tagForDB tag.Tag) {
	insertSampleCustom(t, mydb, tagForDB)
}

func insertSampleCustom(t *testing.T, mydb *db.DB, tagForDB tag.Tag) {

	if err := insert(mydb, tagForDB); err != nil {
		t.Error("Unable to insert tag")
	}
}

// nolint :dupl
func insert(dbs *db.DB, tag tag.Tag) error {

	execFunc := func(collection *mgo.Collection) (*mgo.ChangeInfo, error) {
		return collection.Upsert(bson.M{"epc": tag.Epc}, &tag)
	}

	const tagCollection = "tags"
	if _, err := dbs.ExecuteWithChangeInfo(tagCollection, execFunc); err != nil {
		return errors.Wrap(err, "db.tag.upsert()")
	}

	return nil
}
