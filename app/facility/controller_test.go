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
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/integrationtest"
	"net/url"
	"os"
	"reflect"
	"testing"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"

	db "github.impcloud.net/RSP-Inventory-Suite/go-dbWrapper"
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
func TestWithDataRetrieve(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

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
	insertSample(t, dbs)
}

// nolint :dupl
func TestDelete(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

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

	var coefficientes Coefficients
	// Random coefficient values
	coefficientes.DailyInventoryPercentage = 0.1
	coefficientes.ProbExitError = 0.1
	coefficientes.ProbInStoreRead = 0.1
	coefficientes.ProbUnreadToRead = 0.1

	if err := Insert(dbs, &facilities, coefficientes); err != nil {
		t.Errorf("error inserting facilities %s", err.Error())
	}
}

func TestUpdateExistingItem(t *testing.T) {
	dbs := dbHost.CreateDB(t)
	defer dbs.Close()

	insertSampleCustom(t, dbs, "TestUpdateExistingItem")
	// Mock data
	updatedBody := make(map[string]interface{})
	updatedBody["dailyinventorypercentage"] = 0.15

	if err := UpdateCoefficients(dbs, "TestUpdateExistingItem", updatedBody); err != nil {
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

	// Mock data
	updatedBody := make(map[string]interface{})
	updatedBody["dailyinventorypercentage"] = 0.10

	if err := UpdateCoefficients(dbs, "facility", updatedBody); err != nil {
		if err == web.ErrNotFound {
			t.Log("Facility NOT FOUND")
		} else {
			t.Errorf("error updating facility: %s", err.Error())
		}
	}
}

func insertSample(t *testing.T, mydb *db.DB) {
	insertSampleCustom(t, mydb, t.Name())
}

func insertSampleCustom(t *testing.T, mydb *db.DB, sampleID string) {
	var facility Facility

	facility.Name = sampleID

	if err := insert(mydb, facility); err != nil {
		t.Error("Unable to insert facility")
	}
}

//nolint: dupl
func clearAllData(t *testing.T, mydb *db.DB) {
	execFunc := func(collection *mgo.Collection) error {
		_, err := collection.RemoveAll(bson.M{})
		return err
	}

	if err := mydb.Execute(facilityCollection, execFunc); err != nil {
		t.Error("Unable to delete collection")
	}
}
