/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package facility

import (
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/integrationtest"
	"net/url"
	"os"
	"reflect"
	"testing"

	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/web"
)

var dbHost integrationtest.DBHost

func TestMain(m *testing.M) {
	dbHost = integrationtest.InitHost("facility_test")
	exitCode := m.Run()
	dbHost.Close()
	os.Exit(exitCode)
}

//nolint:dupl
func TestNoDataRetrieve(t *testing.T) {
	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	clearAllData(t, testDB.DB)

	testURL, err := url.Parse("http://localhost/test?$top=10&$select=name,age")
	if err != nil {
		t.Error("failed to parse test url")
	}

	_, _, err = Retrieve(testDB.DB, testURL.Query())
	if err != nil {
		t.Error("Unable to retrieve facilities")
	}
}

func clearAllData(t *testing.T, db *sql.DB) {
	selectQuery := fmt.Sprintf(`DELETE FROM %s`,
		pq.QuoteIdentifier(facilitiesTable),
	)

	_, err := db.Exec(selectQuery)
	if err != nil {
		t.Errorf("Unable to delete data from %s table: %s", facilitiesTable, err)
	}
}

func TestWithDataRetrieve(t *testing.T) {
	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	clearAllData(t, testDB.DB)
	insertSample(t, testDB.DB)

	testURL, err := url.Parse("http://localhost/test?$top=10&$select=name,age")
	if err != nil {
		t.Error("failed to parse test url")
	}

	facilities, count, err := Retrieve(testDB.DB, testURL.Query())

	// Expecting nil count
	if count != nil {
		t.Error("expecting nil count")
	}

	if err != nil {
		t.Error("Unable to retrieve facilities")
	}

	facilitySlice := reflect.ValueOf(facilities)

	if facilitySlice.Len() <= 0 {
		t.Error("Unable to retrieve facilities")
	}
}

func insertSample(t *testing.T, db *sql.DB) {
	insertSampleCustom(t, db, t.Name())
}

func insertSampleCustom(t *testing.T, db *sql.DB, sampleID string) {
	var facility Facility

	facility.Name = sampleID

	if err := insert(db, facility); err != nil {
		t.Error("Unable to insert facility", err)
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

	clearAllData(t, testDB.DB)
	insertSample(t, testDB.DB)
}

// nolint :dupl
func TestDelete(t *testing.T) {
	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	clearAllData(t, testDB.DB)

	// have to insert something before we can delete it
	insertSample(t, testDB.DB)

	if err := Delete(testDB.DB, t.Name()); err != nil {
		if err == web.ErrNotFound {
			t.Fatal("Facility Not found, nothing to delete")
		}
		t.Error("Unable to delete facility")
	}
}

func TestInsertFacilities(t *testing.T) {
	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	var facilities []Facility
	var facility Facility
	facility.Name = "Test"

	facilities = append(facilities, facility)

	var coefficients Coefficients
	// Random coefficient values
	coefficients.DailyInventoryPercentage = 0.1
	coefficients.ProbExitError = 0.1
	coefficients.ProbInStoreRead = 0.1
	coefficients.ProbUnreadToRead = 0.1

	if err := Insert(testDB.DB, &facilities, coefficients); err != nil {
		t.Errorf("error inserting facilities %s", err.Error())
	}
}

func TestUpdateExistingItem(t *testing.T) {
	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	clearAllData(t, testDB.DB)
	insertSampleCustom(t, testDB.DB, "TestUpdateExistingItem")
	// Mock data
	var updateFacility Facility
	updateFacility.Name = "TestUpdateExistingItem"
	updateFacility.Coefficients.DailyInventoryPercentage = 0.15
	updateFacility.Coefficients.ProbUnreadToRead = 0.15
	updateFacility.Coefficients.ProbInStoreRead = 0.15
	updateFacility.Coefficients.ProbExitError = 0.15

	if err := UpdateCoefficients(testDB.DB, updateFacility); err != nil {
		if err == web.ErrNotFound {
			t.Error("Facility NOT FOUND")
		} else {
			t.Errorf("error updating facility: %s", err.Error())
		}
	}
}

//nolint:dupl
func TestDelete_nonExistItem(t *testing.T) {
	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	// we will try to delete random gibberish

	if err := Delete(testDB.DB, "emptyId"); err != nil {
		if err == web.ErrNotFound {
			// because we didn't find it, it should succeed
			t.Log("Facility NOT FOUND, this is the expected result")
		} else {
			t.Error("Expected to not be able to delete")
		}
	}
}

func TestUpdate_nonExistItem(t *testing.T) {
	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	clearAllData(t, testDB.DB)
	// Mock data
	var updateFacility Facility
	updateFacility.Name = "TestUpdateExistingItem"
	updateFacility.Coefficients.DailyInventoryPercentage = 0.15
	updateFacility.Coefficients.ProbUnreadToRead = 0.15
	updateFacility.Coefficients.ProbInStoreRead = 0.15
	updateFacility.Coefficients.ProbExitError = 0.15

	if err := UpdateCoefficients(testDB.DB, updateFacility); err != nil {
		if err == web.ErrNotFound {
			t.Log("Facility NOT FOUND")
		} else {
			t.Errorf("error updating facility: %s", err.Error())
		}
	}
}
