/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package handheldevent

import (
	"database/sql"
	"fmt"
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/pkg/integrationtest"
	"github.com/lib/pq"
	"net/url"
	"os"
	"reflect"
	"testing"
	"time"
)

var dbHost integrationtest.DBHost

func TestMain(m *testing.M) {
	dbHost = integrationtest.InitHost("handheldEvent_test")
	exitCode := m.Run()
	dbHost.Close()
	os.Exit(exitCode)
}

//nolint: dupl
func TestNoDataRetrieve(t *testing.T) {
	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

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
	defer testDB.Close()

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
	defer testDB.Close()

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
	defer testDB.Close()

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
	defer testDB.Close()

	insertSample(t, testDB.DB)
}

func TestInsertHandHeldEvents(t *testing.T) {
	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

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
