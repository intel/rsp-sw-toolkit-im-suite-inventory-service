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
	"net/url"
	"reflect"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
	db "github.impcloud.net/RSP-Inventory-Suite/go-dbWrapper"
	odata "github.impcloud.net/RSP-Inventory-Suite/go-odata/mongo"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/web"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
)

const facilityCollection = "facilities"

// Update receives a facility_id(name) and body to be updated in the facility collection
func UpdateCoefficients(dbs *db.DB, facilityID string, body map[string]interface{}) error {

	// Metrics
	metrics.GetOrRegisterGauge(`Inventory.Update-Facility.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Inventory.Update-Facility.Success`, nil)
	mUpdateErr := metrics.GetOrRegisterGauge(`Inventory.Update-Facility.Update-Error`, nil)
	mErrNotFound := metrics.GetOrRegisterGauge(`Inventory.Update-Facility.NotFound-Error`, nil)
	mUpdateLatency := metrics.GetOrRegisterTimer(`Inventory.Update-Facility.Update-Latency`, nil)
	mEmptyBodyErr := metrics.GetOrRegisterGauge(`Inventory.Update-Facility.EmptyBody-Error`, nil)

	if body == nil {
		mEmptyBodyErr.Update(1)
		return errors.Wrap(web.ErrInvalidInput, "body request cannot be empty")
	}

	selector := bson.M{"name": facilityID}

	execFunc := func(collection *mgo.Collection) error {
		return collection.Update(selector, bson.M{"$set": bson.M{"coefficients": body}})
	}

	updateTimer := time.Now()
	if err := dbs.Execute(facilityCollection, execFunc); err != nil {
		if err == mgo.ErrNotFound {
			mErrNotFound.Update(1)
			return web.ErrNotFound
		}
		mUpdateErr.Update(1)
		return errors.Wrap(err, "db.facility.Update()")
	}
	mUpdateLatency.Update(time.Since(updateTimer))

	mSuccess.Update(1)
	return nil
}

// Retrieve retrieves All facilities from database
//nolint:dupl
func Retrieve(dbs *db.DB, query url.Values) (interface{}, *CountType, error) {

	// Metrics
	metrics.GetOrRegisterGauge(`Inventory.Retrieve-Facility.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Inventory.Retrieve-Facility.Success`, nil)
	mFindErr := metrics.GetOrRegisterGauge("Inventory.Retrieve-Facility.Find-Error", nil)
	mInputErr := metrics.GetOrRegisterGauge("Inventory.Retrieve-Facility.Input-Error", nil)
	mCountErr := metrics.GetOrRegisterGauge("Inventory.Retrieve-Facility.Count-Error", nil)
	mFindLatency := metrics.GetOrRegisterTimer(`Inventory.Retrieve-Facility.Find-Latency`, nil)
	mCountLatency := metrics.GetOrRegisterTimer(`Inventory.Retrieve-Facility.Count-Latency`, nil)

	var object []interface{}

	count := query["$count"]

	// If count is true, return count number
	if len(count) > 0 && len(query) < 2 {

		var count int
		var err error

		execFunc := func(collection *mgo.Collection) (int, error) {
			return odata.ODataCount(collection)
		}

		countTimer := time.Now()
		if count, err = dbs.ExecuteCount(facilityCollection, execFunc); err != nil {
			mCountErr.Update(1)
			return nil, nil, errors.Wrap(err, "db.facilities.Count()")
		}
		mCountLatency.Update(time.Since(countTimer))

		mSuccess.Update(1)
		return nil, &CountType{Count: &count}, nil
	}

	// Else, run filter query and return slice of Facilities
	execFunc := func(collection *mgo.Collection) error {
		return odata.ODataQuery(query, &object, collection)
	}

	retrieveTimer := time.Now()
	if err := dbs.Execute(facilityCollection, execFunc); err != nil {
		if errors.Cause(err) == odata.ErrInvalidInput {
			mInputErr.Update(1)
			return nil, nil, errors.Wrap(web.ErrInvalidInput, err.Error())
		}
		mFindErr.Update(1)
		return nil, nil, errors.Wrap(err, "db.facilities.find()")
	}
	mFindLatency.Update(time.Since(retrieveTimer))

	// Check if inlinecount is set
	inlineCount := query["$inlinecount"]
	var inCount int
	if len(inlineCount) > 0 {
		if inlineCount[0] == "allpages" {
			resultSlice := reflect.ValueOf(object)
			inCount = resultSlice.Len()
			return object, &CountType{Count: &inCount}, nil
		}
	}

	mSuccess.Update(1)
	return object, nil, nil

}

// Insert receives a slice of Facility and coefficients defaults.
// if facility is not in the database, inserts facility with default coefficients
// if facility is in database, skip it.
func Insert(dbs *db.DB, facilities *[]Facility, coefficients Coefficients) error {

	// Metrics
	metrics.GetOrRegisterGauge(`Inventory.Insert-Facility.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Inventory.Insert-Facility.Success`, nil)
	mInsertErr := metrics.GetOrRegisterGauge(`Inventory.Insert-Facility.Insert-Error`, nil)
	mGetFacilitiesErr := metrics.GetOrRegisterGauge(`Inventory.Insert-Facility.GetFacilities-Error`, nil)
	mInsertLatency := metrics.GetOrRegisterTimer(`Inventory.Insert-Facility.Insert-Latency`, nil)
	mGetFacilitiesLatency := metrics.GetOrRegisterTimer(`Inventory.Insert-Facility.GetFacilities-Latency`, nil)

	// Query all facilities in db
	getFacilitiesTimer := time.Now()
	facilitiesInDb, err := CreateFacilityMap(dbs)
	if err != nil {
		mGetFacilitiesErr.Update(1)
		return errors.Wrap(err, "")
	}
	mGetFacilitiesLatency.Update(time.Since(getFacilitiesTimer))

	insertTimer := time.Now()
	for _, facItem := range *facilities {

		// If facility is not in the dababase, set default coefficient and insert it into db
		// Skip if facility already stored in database
		if _, ok := facilitiesInDb[facItem.Name]; !ok {
			facItem.Coefficients.DailyInventoryPercentage = coefficients.DailyInventoryPercentage
			facItem.Coefficients.ProbExitError = coefficients.ProbExitError
			facItem.Coefficients.ProbInStoreRead = coefficients.ProbInStoreRead
			facItem.Coefficients.ProbUnreadToRead = coefficients.ProbUnreadToRead
			if err := insert(dbs, facItem); err != nil {
				mInsertErr.Update(1)
				return errors.Wrapf(err, "unable to insert facility %s", facItem.Name)
			}
		}
	}
	mInsertLatency.Update(time.Since(insertTimer))

	mSuccess.Update(1)
	return nil
}

// CreateFacilityMap builds a map[string] based of array of facilities for search efficiency
func CreateFacilityMap(dbs *db.DB) (map[string]Facility, error) {

	metrics.GetOrRegisterGauge(`Inventory.CreateFacilityMap.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Inventory.CreateFacilityMap.Success`, nil)
	mFindAllErr := metrics.GetOrRegisterGauge(`Inventory.CreateFacilityMap.FindAll-Error`, nil)
	mFindALlLatency := metrics.GetOrRegisterTimer(`Inventory.CreateFacilityMap.FindAll-Latency`, nil)

	findAllTimer := time.Now()
	facilities, err := findAll(dbs)
	if err != nil {
		mFindAllErr.Update(1)
		return nil, err
	}
	mFindALlLatency.Update(time.Since(findAllTimer))

	if facilities == nil {
		mSuccess.Update(1)
		return nil, nil
	}

	facMap := make(map[string]Facility, len(*facilities))

	for _, item := range *facilities {
		facMap[item.Name] = item
	}

	mSuccess.Update(1)
	return facMap, nil

}

func findAll(dbs *db.DB) (*[]Facility, error) {

	var facilities []Facility

	execFunc := func(collection *mgo.Collection) error {
		return collection.Find(nil).All(&facilities)
	}

	if err := dbs.Execute(facilityCollection, execFunc); err != nil {
		return nil, errors.Wrap(err, "db.facilities.find.All()")
	}

	if len(facilities) == 0 {
		return nil, nil
	}

	return &facilities, nil

}

func insert(dbs *db.DB, facility Facility) error {

	execFunc := func(collection *mgo.Collection) error {
		return collection.Insert(facility)
	}

	if err := dbs.Execute(facilityCollection, execFunc); err != nil {
		return errors.Wrap(err, "db.facility.insert()")
	}

	return nil
}

// Delete removes facility based on name
// nolint :dupl
func Delete(dbs *db.DB, name string) error {

	// Metrics
	metrics.GetOrRegisterGauge(`Inventory.Delete-Facility.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Inventory.Delete-Facility.Success`, nil)
	mDeleteErr := metrics.GetOrRegisterGauge(`Inventory.Delete-Facility.Delete-Error`, nil)
	mErrNotFound := metrics.GetOrRegisterGauge(`Inventory.Delete-Facility.NotFound-Error`, nil)
	mDeleteLatency := metrics.GetOrRegisterTimer(`Inventory.Delete-Facility.Delete-Latency`, nil)

	execFunc := func(collection *mgo.Collection) error {
		return collection.Remove(bson.M{"name": name})
	}

	deleteTimer := time.Now()
	if err := dbs.Execute(facilityCollection, execFunc); err != nil {
		if err == mgo.ErrNotFound {
			mErrNotFound.Update(1)
			return web.ErrNotFound
		}
		mDeleteErr.Update(1)
		return errors.Wrap(err, "db.facility.Delete()")
	}
	mDeleteLatency.Update(time.Since(deleteTimer))

	mSuccess.Update(1)
	return nil
}
