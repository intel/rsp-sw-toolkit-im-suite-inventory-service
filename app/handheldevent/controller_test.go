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

package handheldevent

import (
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/integrationtest"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"

	"time"

	db "github.impcloud.net/RSP-Inventory-Suite/go-dbWrapper"
)

var dbHost integrationtest.DBHost

func TestMain(m *testing.M) {
	dbHost = integrationtest.InitHost("handHeldEvent_test")
	os.Exit(m.Run())
}

//nolint: dupl
func TestNoDataRetrieve(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	clearAllData(t, dbs)

	testURL, err := url.Parse("http://localhost/test?$top=10&$select=event,timestamp")
	if err != nil {
		t.Error("failed to parse test url")
	}

	_, _, err = Retrieve(dbs, testURL.Query())
	if err != nil {
		t.Error("Unable to retrieve handheld events")
	}
}
func TestWithDataRetrieve(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	insertSample(t, dbs)

	testURL, err := url.Parse("http://localhost/test?$top=10&$select=event,timestamp")
	if err != nil {
		t.Error("failed to parse test url")
	}

	events, count, err := Retrieve(dbs, testURL.Query())

	// Expecting nil count
	if count != nil {
		t.Error("expecting nil count")
	}

	if err != nil {
		t.Error("Unable to retrieve handheld events")
	}

	eventsSlice := reflect.ValueOf(events)

	if eventsSlice.Len() <= 0 {
		t.Error("Didn't retrieve any handheld events")
	}
}

//nolint:dupl
func TestRetrieveCount(t *testing.T) {

	testURL, err := url.Parse("http://localhost/test?$count")
	if err != nil {
		t.Error("failed to parse test url")
	}

	dbs := dbHost.CreateDB(t)
	defer dbs.Close()
	insertSample(t, dbs)

	_, count, err := Retrieve(dbs, testURL.Query())

	if count == nil {
		t.Error("expecting CountType result")
	}

	if err != nil {
		t.Error("Unable to retrieve total count")
	}
}

//nolint:dupl
func TestRetrieveInlinecount(t *testing.T) {

	testURL, err := url.Parse("http://localhost/test?$inlinecount=allpages")
	if err != nil {
		t.Error("failed to parse test url")
	}

	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	_, count, err := Retrieve(dbs, testURL.Query())

	if count == nil {
		t.Error("expecting CountType result")
	}

	if err != nil {
		t.Error("Unable to retrieve total count")
	}
}

func TestInsert(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()
	insertSample(t, dbs)
}

func TestInsertHandHeldEvents(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	insertSample(t, dbs)
	eventNames := []string{"FullScanStart", "FullScanStop", "Calculate"}

	for _, eventName := range eventNames {
		var eventData HandheldEvent

		eventData.Event = eventName
		eventData.Timestamp = time.Now().Unix()

		if err := Insert(dbs, eventData); err != nil {
			t.Errorf("error inserting handheld event %s", err.Error())
		}
	}
}

func insertSample(t *testing.T, mydb *db.DB) {
	var eventData HandheldEvent

	eventData.Event = "FullScanStart"
	eventData.Timestamp = time.Now().Unix()

	if err := Insert(mydb, eventData); err != nil {
		t.Error("Unable to insert handheld Event")
	}
}

//nolint:dupl
func clearAllData(t *testing.T, mydb *db.DB) {
	execFunc := func(collection *mgo.Collection) error {
		_, err := collection.RemoveAll(bson.M{})
		return err
	}

	if err := mydb.Execute(collection, execFunc); err != nil {
		t.Error("Unable to delete collection")
	}
}

func TestUpdateTTLIndex(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	ttlIndex := "ttl"
	purgingSeconds := 1800

	// Add index before updating
	if err := addIndex(t, dbs, ttlIndex); err != nil {
		t.Errorf("Error addIndex(): %s", err.Error())
	}

	if err := UpdateTTLIndexForHandheldEvents(dbs, purgingSeconds); err != nil {
		t.Errorf("Error UpdateTTLIndexForHandheldEvents(): %s", err.Error())
	}

	execFunc := func(collection *mgo.Collection) error {
		indexes, err := collection.Indexes()
		if err != nil {
			return err
		}
		// check if ttl index was updated
		for i, v := range indexes {
			if strings.Contains(v.Name, ttlIndex) {
				updatedExpireTime := int(v.ExpireAfter / time.Second)
				if updatedExpireTime != purgingSeconds {
					t.Error("Update of ttl index failed expected", purgingSeconds, "got", updatedExpireTime)
				}
			} else if i == (len(indexes) - 1) {
				t.Error("ttl index not found")
			}

		}
		return nil
	}

	if err := dbs.Execute(collection, execFunc); err != nil {
		t.Error("UpdateTTLIndexForHandheldEvents test failed", err.Error())
	}

	// Clear data and negative testing
	if err := dropIndex(t, dbs, ttlIndex); err != nil {
		t.Errorf("Error dropIndex(): %s", err.Error())
	}

	err := UpdateTTLIndexForHandheldEvents(dbs, purgingSeconds)
	if err == nil {
		t.Error("Update should have failed as ttl index does not exist")
	}
}

func addIndex(t *testing.T, mydb *db.DB, indexName string) error {

	index := mgo.Index{
		Key:         []string{indexName},
		Unique:      false,
		DropDups:    false,
		Background:  false,
		ExpireAfter: time.Duration(60) * time.Second,
	}

	execFunc := func(collection *mgo.Collection) error {
		return collection.EnsureIndex(index)
	}

	if err := mydb.Execute(collection, execFunc); err != nil {
		t.Error("Add index failed", err.Error())
	}

	return nil
}

func dropIndex(t *testing.T, mydb *db.DB, index string) error {

	execFunc := func(collection *mgo.Collection) error {
		return collection.DropIndex(index)
	}

	if err := mydb.Execute(collection, execFunc); err != nil {
		t.Error("Drop index failed", err.Error())
	}

	return nil
}
