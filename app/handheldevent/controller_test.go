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
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/integrationtest"
	"net/url"
	"os"
	"reflect"
	"testing"
	"time"
)

var dbHost integrationtest.DBHost

func TestMain(m *testing.M) {
	dbHost = integrationtest.InitHost("handheldEvent_test")
	defer dbHost.Close()
	os.Exit(m.Run())
}

//nolint: dupl
func TestNoDataRetrieve(t *testing.T) {
	testDB := dbHost.CreateDB(t)
	defer testDB.DB.Close()

	clearAllData(t, testDB.DB)

	testURL, err := url.Parse("http://localhost/test?$top=10&$select=event,timestamp")
	if err != nil {
		t.Error("failed to parse test url")
	}

	_, _, err = Retrieve(testDB.DB, testURL.Query())
	if err != nil {
		t.Error("Unable to retrieve handheld events")
	}
}
func TestWithDataRetrieve(t *testing.T) {
	testDB := dbHost.CreateDB(t)
	defer testDB.DB.Close()

	insertSample(t, testDB.DB)

	testURL, err := url.Parse("http://localhost/test?$top=10&$select=event,timestamp")
	if err != nil {
		t.Error("failed to parse test url")
	}

	events, count, err := Retrieve(testDB.DB, testURL.Query())

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

	testDB := dbHost.CreateDB(t)
	defer testDB.DB.Close()

	insertSample(t, testDB.DB)

	_, count, err := Retrieve(testDB.DB, testURL.Query())

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

	testDB := dbHost.CreateDB(t)
	defer testDB.DB.Close()

	_, count, err := Retrieve(testDB.DB, testURL.Query())

	if count == nil {
		t.Error("expecting CountType result")
	}

	if err != nil {
		t.Error("Unable to retrieve total count")
	}
}

func TestInsert(t *testing.T) {
	testDB := dbHost.CreateDB(t)
	defer testDB.DB.Close()

	insertSample(t, testDB.DB)
}

func TestInsertHandHeldEvents(t *testing.T) {
	testDB := dbHost.CreateDB(t)
	defer testDB.DB.Close()

	insertSample(t, testDB.DB)
	eventNames := []string{"FullScanStart", "FullScanStop", "Calculate"}

	for _, eventName := range eventNames {
		var eventData HandheldEvent

		eventData.Event = eventName
		eventData.Timestamp = time.Now().Unix()

		if err := Insert(testDB.DB, eventData); err != nil {
			t.Errorf("error inserting handheld event %s", err.Error())
		}
	}
}

func insertSample(t *testing.T, db *sql.DB) {
	var eventData HandheldEvent

	eventData.Event = "FullScanStart"
	eventData.Timestamp = time.Now().Unix()

	if err := Insert(db, eventData); err != nil {
		t.Error("Unable to insert handheld Event")
	}
}

//nolint:dupl
func clearAllData(t *testing.T, db *sql.DB) {
	selectQuery := fmt.Sprintf(`DELETE FROM %s`,
		pq.QuoteIdentifier(handheldEventsTable),
	)

	_, err := db.Exec(selectQuery)
	if err != nil {
		t.Errorf("Unable to delete data from %s: %s", handheldEventsTable, err)
	}
}
