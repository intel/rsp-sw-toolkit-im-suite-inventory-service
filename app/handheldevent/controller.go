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
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/lib/pq"
	odata "github.impcloud.net/RSP-Inventory-Suite/go-odata/postgresql"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/web"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
	"net/url"
	"time"

	"github.com/pkg/errors"
)

const handheldEventsTable = "handheldevents"
const jsonb = "data"

type handheldEventWrapper struct {
	ID   []uint8       `db:"id" json:"id"`
	Data HandheldEvent `db:"data" json:"data"`
}

// Retrieve retrieves All handheld events from database
//nolint:dupl
func Retrieve(dbs *sql.DB, query url.Values) (interface{}, *CountType, error) {
	// Metrics
	metrics.GetOrRegisterGauge(`HandheldEvent.Retrieve.Attempt`, nil).Update(1)
	mCountErr := metrics.GetOrRegisterGauge("HandheldEvent.Retrieve.Count-Error", nil)
	mSuccess := metrics.GetOrRegisterGauge(`HandheldEvent.Retrieve.Success`, nil)
	mRetrieveErr := metrics.GetOrRegisterGauge("HandheldEvent.Retrieve.Retrieve-Error", nil)
	mInputErr := metrics.GetOrRegisterGauge("HandheldEvent.Retrieve.Input-Error", nil)
	mRetrieveLatency := metrics.GetOrRegisterTimer(`HandheldEvent.Retrieve.Retrieve-Latency`, nil)

	countQuery := query["$count"]

	// If only $count is set, return total count of the table
	if len(countQuery) > 0 && len(query) < 2 {

		var count int

		row := dbs.QueryRow("SELECT count(*) FROM " + handheldEventsTable)
		err := row.Scan(&count)
		if err != nil {
			mCountErr.Update(1)
			return nil, nil, err
		}

		mSuccess.Update(1)
		return nil, &CountType{Count: &count}, nil
	}

	// Else, run filter query and return slice of handheld events
	retrieveTimer := time.Now()

	// Run OData PostgreSQL
	rows, err := odata.ODataSQLQuery(query, handheldEventsTable, jsonb, dbs)
	if err != nil {
		if errors.Cause(err) == odata.ErrInvalidInput {
			mInputErr.Update(1)
			return nil, nil, errors.Wrap(web.ErrInvalidInput, err.Error())
		}
		return nil, nil, errors.Wrap(err, "error in retrieving handheld events")
	}
	mRetrieveLatency.Update(time.Since(retrieveTimer))
	defer rows.Close()

	eventSlice := make([]HandheldEvent, 0)

	inlineCount := 0

	// Loop through the results and append them to a slice
	for rows.Next() {

		handheldEventWrapper := new(handheldEventWrapper)
		err := rows.Scan(&handheldEventWrapper.ID, &handheldEventWrapper.Data)
		if err != nil {
			mRetrieveErr.Update(1)
			return nil, nil, err
		}
		eventSlice = append(eventSlice, handheldEventWrapper.Data)
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
		return eventSlice, &CountType{Count: &inlineCount}, nil
	} else if len(countQuery) > 0 {
		mSuccess.Update(1)
		return nil, &CountType{Count: &inlineCount}, nil
	}

	mSuccess.Update(1)
	return eventSlice, nil, nil
}

// Value implements driver.Valuer inferfaces
func (handheldEvent HandheldEvent) Value() (driver.Value, error) {
	return json.Marshal(handheldEvent)
}

// Scan implements sql.Scanner inferfaces
func (handheldEvent *HandheldEvent) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, handheldEvent)
}

// Insert to insert handheldEvent into database
func Insert(dbs *sql.DB, handheldEvent HandheldEvent) error {

	obj, err := json.Marshal(handheldEvent)
	if err != nil {
		return err
	}

	insertStmt := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s); `,
		pq.QuoteIdentifier(handheldEventsTable),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteLiteral(string(obj)),
	)

	_, err = dbs.Exec(insertStmt)
	if err != nil {
		return errors.Wrap(err, "error in inserting handheld event")
	}

	return nil
}
