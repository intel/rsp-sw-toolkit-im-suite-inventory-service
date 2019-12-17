/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package sensor

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/intel/rsp-sw-toolkit-im-suite-utilities/go-metrics"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"time"
)

const (
	rspConfigTable = "rspconfig"
	jsonb          = "data"
	deviceIdColumn = "device_id"
)

// Value implements driver.Valuer interfaces
func (rsp *RSP) Value() (driver.Value, error) {
	return json.Marshal(rsp)
}

// Scan implements sql.Scanner interfaces
func (rsp *RSP) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, rsp)
}

// FindRSP searches DB for RSP based on the device_id value
// Returns the RSP if found or empty RSP if it does not exist
func FindRSP(dbs *sql.DB, deviceId string) (*RSP, error) {
	retrieveTimer := time.Now()

	// Metrics
	metrics.GetOrRegisterGauge(`Sensor.FindRSP.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Sensor.FindRSP.Success`, nil)
	mFindErr := metrics.GetOrRegisterGauge("Sensor.FindRSP.Find-Error", nil)
	defer metrics.GetOrRegisterTimer(`Sensor.FindRSP.Find-Latency`, nil).Update(time.Since(retrieveTimer))

	rsp := new(RSP)

	selectQuery := fmt.Sprintf(`SELECT %s FROM %s WHERE %s ->> %s = %s LIMIT 1`,
		pq.QuoteIdentifier(jsonb),
		pq.QuoteIdentifier(rspConfigTable),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteLiteral(deviceIdColumn),
		pq.QuoteLiteral(deviceId),
	)

	if err := dbs.QueryRow(selectQuery).Scan(&rsp); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		mFindErr.Update(1)
		return nil, errors.Wrapf(err, "error in finding rsp")
	}

	mSuccess.Update(1)
	return rsp, nil
}

// Upsert takes a pointer to an rsp config and either adds it to the DB if it is new,
// or updates its values if it is existing
func Upsert(dbs *sql.DB, rsp *RSP) error {

	upsertTimer := time.Now()

	// Metrics
	metrics.GetOrRegisterGaugeCollection(`Sensor.RSP.Upsert.Attempt`, nil).Add(1)
	mSuccess := metrics.GetOrRegisterGaugeCollection(`Sensor.RSP.Upsert.Success`, nil)
	mUpsertErr := metrics.GetOrRegisterGaugeCollection(`Sensor.RSP.Upsert.Error`, nil)
	defer metrics.GetOrRegisterTimer(`Sensor.RSP.Upsert.Latency`, nil).Update(time.Since(upsertTimer))

	obj, err := json.Marshal(rsp)
	if err != nil {
		return errors.Wrapf(err, "error in marshalling an rsp before upsert")
	}

	upsertStmt := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s) 
									 ON CONFLICT (( %s  ->> %s )) 
									 DO UPDATE SET %s = %s.%s || %s; `,
		pq.QuoteIdentifier(rspConfigTable),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteLiteral(string(obj)),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteLiteral(deviceIdColumn),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteIdentifier(rspConfigTable),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteLiteral(string(obj)),
	)

	_, err = dbs.Exec(upsertStmt)
	if err != nil {
		mUpsertErr.Add(1)
		return errors.Wrapf(err, "error in upserting an rsp")
	}

	mSuccess.Add(1)
	return nil
}
