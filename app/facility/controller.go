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
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/lib/pq"
	odata "github.impcloud.net/RSP-Inventory-Suite/go-odata/postgresql"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/web"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
)

const facilitiesTable = "facilities"
const jsonb = "data"
const nameColumn = "name"
const coefficientsColumn = "coefficients"

type facilityDataWrapper struct {
	ID   []uint8  `db:"id" json:"id"`
	Data Facility `db:"data" json:"data"`
}

// Update receives a facility_id(name) and body to be updated in the facility collection
func UpdateCoefficients(dbs *sql.DB, facility Facility) error {

	// Metrics
	metrics.GetOrRegisterGauge(`Inventory.Update-Facility.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Inventory.Update-Facility.Success`, nil)
	mUpdateErr := metrics.GetOrRegisterGauge(`Inventory.Update-Facility.Update-Error`, nil)
	mErrNotFound := metrics.GetOrRegisterGauge(`Inventory.Update-Facility.NotFound-Error`, nil)
	mUpdateLatency := metrics.GetOrRegisterTimer(`Inventory.Update-Facility.Update-Latency`, nil)

	coefficients, err := json.Marshal(facility.Coefficients)
	if err != nil {
		return err
	}

	upsertClause := fmt.Sprintf(`UPDATE %s SET %s = jsonb_set(%s, '{%s}', %s)
					WHERE %s ->> %s = %s`,
		pq.QuoteIdentifier(facilitiesTable),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteIdentifier(coefficientsColumn),
		pq.QuoteLiteral(string(coefficients)),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteLiteral(nameColumn),
		pq.QuoteLiteral(facility.Name),
	)

	updateTimer := time.Now()
	result, err := dbs.Exec(upsertClause)
	if err != nil {
		mUpdateErr.Update(1)
		return err
	}
	updatedRow, err := result.RowsAffected()

	if err != nil {
		mUpdateErr.Update(1)
		return err
	}
	if updatedRow == 0 {
		mErrNotFound.Update(1)
		return web.ErrNotFound
	}

	mUpdateLatency.Update(time.Since(updateTimer))

	mSuccess.Update(1)
	return nil
}

// Retrieve retrieves All facilities from database
//nolint:dupl
func Retrieve(dbs *sql.DB, query url.Values) (interface{}, *CountType, error) {

	// Metrics
	metrics.GetOrRegisterGauge(`Inventory.Retrieve-Facility.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Inventory.Retrieve-Facility.Success`, nil)
	mRetrieveErr := metrics.GetOrRegisterGauge("Inventory.Retrieve-Facility.Find-Error", nil)
	mInputErr := metrics.GetOrRegisterGauge("Inventory.Retrieve-Facility.Input-Error", nil)
	mCountErr := metrics.GetOrRegisterGauge("Inventory.Retrieve-Facility.Count-Error", nil)
	mRetrieveLatency := metrics.GetOrRegisterTimer(`Inventory.Retrieve-Facility.Find-Latency`, nil)

	countQuery := query["$count"]

	// If count is true, return count number
	if len(countQuery) > 0 && len(query) < 2 {

		var count int
		selectStmt := fmt.Sprintf(`SELECT count(*) from %s`,
			pq.QuoteIdentifier(facilitiesTable),
		)

		row := dbs.QueryRow(selectStmt)
		err := row.Scan(&count)
		if err != nil {
			mCountErr.Update(1)
			return nil, nil, err
		}

		mSuccess.Update(1)
		return nil, &CountType{Count: &count}, nil
	}

	// Else, run filter query and return slice of Facilities
	retrieveTimer := time.Now()

	// Run OData PostgreSQL
	rows, err := odata.ODataSQLQuery(query, facilitiesTable, jsonb, dbs)
	if err != nil {
		if errors.Cause(err) == odata.ErrInvalidInput {
			mInputErr.Update(1)
			return nil, nil, errors.Wrap(web.ErrInvalidInput, err.Error())
		}
		return nil, nil, errors.Wrap(err, "error in retrieving facilities")
	}

	mRetrieveLatency.Update(time.Since(retrieveTimer))
	defer rows.Close()

	facilitySlice := make([]Facility, 0)

	inlineCount := 0

	// Loop through the results and append them to a slice
	for rows.Next() {

		facilityDataWrapper := new(facilityDataWrapper)
		err := rows.Scan(&facilityDataWrapper.ID, &facilityDataWrapper.Data)
		if err != nil {
			mRetrieveErr.Update(1)
			return nil, nil, err
		}
		facilitySlice = append(facilitySlice, facilityDataWrapper.Data)
		inlineCount++

	}
	if err = rows.Err(); err != nil {
		mRetrieveErr.Update(1)
		return nil, nil, err
	}

	// Check if $inlinecount or $count is set in combination with $filter
	isInlineCount := query["$inlinecount"]

	if len(isInlineCount) > 0 && isInlineCount[0] == "allpages" {
		mSuccess.Update(1)
		return facilitySlice, &CountType{Count: &inlineCount}, nil
	} else if len(countQuery) > 0 {
		mSuccess.Update(1)
		return nil, &CountType{Count: &inlineCount}, nil
	}

	mSuccess.Update(1)
	return facilitySlice, nil, nil
}

// Insert receives a slice of Facility and coefficients defaults.
// if facility is not in the database, inserts facility with default coefficients
// if facility is in database, skip it.
func Insert(dbs *sql.DB, facilities *[]Facility, coefficients Coefficients) error {

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

		// If facility is not in the database, set default coefficients and insert it
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
func CreateFacilityMap(dbs *sql.DB) (map[string]Facility, error) {

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

	facMap := make(map[string]Facility, len(facilities))

	for _, item := range facilities {
		facMap[item.Name] = item
	}

	mSuccess.Update(1)
	return facMap, nil

}

func findAll(dbs *sql.DB) ([]Facility, error) {

	selectQuery := fmt.Sprintf(`SELECT %s FROM %s`,
		pq.QuoteIdentifier(jsonb),
		pq.QuoteIdentifier(facilitiesTable),
	)

	rows, err := dbs.Query(selectQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var facilitySlice []Facility

	for rows.Next() {

		facilityDataWrapper := new(facilityDataWrapper)
		err := rows.Scan(&facilityDataWrapper.Data)
		if err != nil {
			return nil, err
		}
		facilitySlice = append(facilitySlice, facilityDataWrapper.Data)

	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return facilitySlice, nil
}

func insert(dbs *sql.DB, facility Facility) error {

	obj, err := json.Marshal(facility)
	if err != nil {
		return err
	}

	insertStmt := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s); `,
		pq.QuoteIdentifier(facilitiesTable),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteLiteral(string(obj)),
	)

	_, err = dbs.Exec(insertStmt)
	if err != nil {
		return errors.Wrap(err, "error in inserting facility")
	}

	return nil
}

// Delete removes facility based on name
// nolint :dupl
func Delete(dbs *sql.DB, name string) error {

	// Metrics
	metrics.GetOrRegisterGauge(`Inventory.Delete-Facility.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Inventory.Delete-Facility.Success`, nil)
	mDeleteErr := metrics.GetOrRegisterGauge(`Inventory.Delete-Facility.Delete-Error`, nil)
	mErrNotFound := metrics.GetOrRegisterGauge(`Inventory.Delete-Facility.NotFound-Error`, nil)
	mDeleteLatency := metrics.GetOrRegisterTimer(`Inventory.Delete-Facility.Delete-Latency`, nil)

	selectQuery := fmt.Sprintf(`DELETE FROM %s WHERE %s ->> %s = %s;`,
		pq.QuoteIdentifier(facilitiesTable),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteLiteral(nameColumn),
		pq.QuoteLiteral(name),
	)

	deleteTimer := time.Now()
	if _, err := dbs.Exec(selectQuery); err != nil {
		if err == sql.ErrNoRows {
			mErrNotFound.Update(1)
			return web.ErrNotFound
		}
		mDeleteErr.Update(1)
		return errors.Wrap(err, "error in deleting facility")
	}
	mDeleteLatency.Update(time.Since(deleteTimer))

	mSuccess.Update(1)
	return nil
}

// Value implements driver.Valuer interfaces
func (facility Facility) Value() (driver.Value, error) {
	return json.Marshal(facility)
}

// Scan implements sql.Scanner interfaces
func (facility *Facility) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, facility)
}
