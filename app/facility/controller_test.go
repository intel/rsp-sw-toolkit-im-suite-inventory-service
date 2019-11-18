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
	os.Exit(m.Run())
}

//nolint:dupl
func TestNoDataRetrieve(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	clearAllData(t, dbs)

	testURL, err := url.Parse("http://localhost/test?$top=10&$select=name,age")
	if err != nil {
		t.Error("failed to parse test url")
	}

	_, _, err = Retrieve(dbs, testURL.Query())
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
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	clearAllData(t, dbs)
	insertSample(t, dbs)

	testURL, err := url.Parse("http://localhost/test?$top=10&$select=name,age")
	if err != nil {
		t.Error("failed to parse test url")
	}

	facilities, count, err := Retrieve(dbs, testURL.Query())

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

	clearAllData(t, dbs)
	insertSample(t, dbs)
}

// nolint :dupl
func TestDelete(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	clearAllData(t, dbs)

	// have to insert something before we can delete it
	insertSample(t, dbs)

	if err := Delete(dbs, t.Name()); err != nil {
		if err == web.ErrNotFound {
			t.Fatal("Facility Not found, nothing to delete")
		}
		t.Error("Unable to delete facility")
	}
}

func TestInsertFacilities(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

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

	if err := Insert(dbs, &facilities, coefficients); err != nil {
		t.Errorf("error inserting facilities %s", err.Error())
	}
}

func TestUpdateExistingItem(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	clearAllData(t, dbs)
	insertSampleCustom(t, dbs, "TestUpdateExistingItem")
	// Mock data
	var updateFacility Facility
	updateFacility.Name = "TestUpdateExistingItem"
	updateFacility.Coefficients.DailyInventoryPercentage = 0.15
	updateFacility.Coefficients.ProbUnreadToRead = 0.15
	updateFacility.Coefficients.ProbInStoreRead = 0.15
	updateFacility.Coefficients.ProbExitError = 0.15

	if err := UpdateCoefficients(dbs, updateFacility); err != nil {
		if err == web.ErrNotFound {
			t.Error("Facility NOT FOUND")
		} else {
			t.Errorf("error updating facility: %s", err.Error())
		}
	}
}

//nolint:dupl
func TestDelete_nonExistItem(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	// we will try to delete random gibberish

	if err := Delete(dbs, "emptyId"); err != nil {
		if err == web.ErrNotFound {
			// because we didn't find it, it should succeed
			t.Log("Facility NOT FOUND, this is the expected result")
		} else {
			t.Error("Expected to not be able to delete")
		}
	}
}

func TestUpdate_nonExistItem(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	clearAllData(t, dbs)
	// Mock data
	var updateFacility Facility
	updateFacility.Name = "TestUpdateExistingItem"
	updateFacility.Coefficients.DailyInventoryPercentage = 0.15
	updateFacility.Coefficients.ProbUnreadToRead = 0.15
	updateFacility.Coefficients.ProbInStoreRead = 0.15
	updateFacility.Coefficients.ProbExitError = 0.15

	if err := UpdateCoefficients(dbs, updateFacility); err != nil {
		if err == web.ErrNotFound {
			t.Log("Facility NOT FOUND")
		} else {
			t.Errorf("error updating facility: %s", err.Error())
		}
	}
}
