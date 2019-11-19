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

package tag

import (
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	odata "github.impcloud.net/RSP-Inventory-Suite/go-odata/postgresql"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/web"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	tagsTable      = "tags"
	jsonb          = "data"
	epcColumn      = "epc"
	facilityColumn = "facility_id"
	// UndefinedProductID is the constant to set the product id when it cannot be decoded
	UndefinedProductID = "undefined"
	// encodingInvalid is the constant to set when epc encoding cannot be decoded
	encodingInvalid = "encoding:invalid"
)

type tagsDataWrapper struct {
	ID   []uint8 `db:"id" json:"id"`
	Data Tag     `db:"data" json:"data"`
}

// Retrieve retrieves tags from database based on Odata query and a size limit
//nolint:dupl
func Retrieve(dbs *sql.DB, query url.Values, maxSize int) (interface{}, *CountType, error) {

	// Metrics
	metrics.GetOrRegisterGauge(`Inventory.Retrieve.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Inventory.Retrieve.Success`, nil)
	mFindErr := metrics.GetOrRegisterGauge("Inventory.Retrieve.Find-Error", nil)
	mInputErr := metrics.GetOrRegisterGauge("Inventory.Retrieve.Input-Error", nil)
	mFindLatency := metrics.GetOrRegisterTimer(`Inventory.Retrieve.Find-Latency`, nil)

	/* Cursor is not a priority for the upcoming release it will be added in future if required */
	//var paging PagingType

	// If count is true, and only $count is set return total count of the collection
	if len(query["$count"]) > 0 && len(query) < 2 {

		return countHandler(dbs)
	}

	if len(query["$top"]) > 0 {

		topVal, err := strconv.Atoi(query["$top"][0])
		if err != nil {
			return nil, nil, errors.Wrap(web.ErrValidation, "invalid $top value")
		}

		if topVal > maxSize {
			query["$top"][0] = strconv.Itoa(maxSize)
		}

	} else {
		query["$top"] = []string{strconv.Itoa(maxSize)} // Apply size limit to the odata query
	}

	// Else, run filter query and return slice of Tag
	retrieveTimer := time.Now()

	rows, err := odata.ODataSQLQuery(query, tagsTable, jsonb, dbs)
	if err != nil {
		if errors.Cause(err) == odata.ErrInvalidInput {
			mInputErr.Update(1)
			return nil, nil, errors.Wrap(web.ErrInvalidInput, err.Error())
		}
		return nil, nil, errors.Wrap(err, "error in retrieving tags")
	}

	mFindLatency.Update(time.Since(retrieveTimer))
	defer rows.Close()

	tagSlice := make([]Tag, 0)

	inlineCount := 0

	// Loop through the results and append them to a slice
	for rows.Next() {

		tagsDataWrapper := new(tagsDataWrapper)
		err := rows.Scan(&tagsDataWrapper.ID, &tagsDataWrapper.Data)
		if err != nil {
			mFindErr.Update(1)
			return nil, nil, err
		}
		tagSlice = append(tagSlice, tagsDataWrapper.Data)
		inlineCount++

	}
	if err = rows.Err(); err != nil {
		mFindErr.Update(1)
		return nil, nil, err
	}

	// Check if inlinecount is set
	isInlineCount := query["$inlinecount"]
	countQuery := query["$count"]

	if len(isInlineCount) > 0 && isInlineCount[0] == "allpages" {
		mSuccess.Update(1)
		return tagSlice, &CountType{Count: &inlineCount}, nil
	} else if len(countQuery) > 0 {
		mSuccess.Update(1)
		return nil, &CountType{Count: &inlineCount}, nil
	}

	/*var pagingType *PagingType
	if len(query["$top"]) > 0 {
		pagingType = &paging
	}*/

	mSuccess.Update(1)
	return tagSlice, nil, nil
}

// RetrieveOdataAll retrieves all tags from the database that matches the query without any size limit
func RetrieveOdataAll(dbs *sql.DB, query url.Values) ([]Tag, error) {

	// Metrics
	metrics.GetOrRegisterGauge(`Inventory.RetrieveOdataAll.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Inventory.RetrieveOdataAll.Success`, nil)
	mFindErr := metrics.GetOrRegisterGauge("Inventory.RetrieveOdataAll.Find-Error", nil)
	mFindLatency := metrics.GetOrRegisterTimer(`Inventory.RetrieveOdataAll.Find-Latency`, nil)

	// Else, run filter query and return slice of Tag
	retrieveTimer := time.Now()

	rows, err := odata.ODataSQLQuery(query, tagsTable, jsonb, dbs)
	if err != nil {
		if errors.Cause(err) == odata.ErrInvalidInput {
			mFindErr.Update(1)
			return nil, errors.Wrap(err, "Invalid Input")
		}
		return nil, errors.Wrap(err, "Error retrieving all tags based on odata query")
	}
	mFindLatency.Update(time.Since(retrieveTimer))
	defer rows.Close()

	tagSlice := make([]Tag, 0)

	// Loop through the results and append them to a slice
	for rows.Next() {

		tagsDataWrapper := new(tagsDataWrapper)
		err := rows.Scan(&tagsDataWrapper.ID, &tagsDataWrapper.Data)
		if err != nil {
			mFindErr.Update(1)
			return nil, err
		}
		tagSlice = append(tagSlice, tagsDataWrapper.Data)
	}
	if err = rows.Err(); err != nil {
		mFindErr.Update(1)
		return nil, err
	}
	mSuccess.Update(1)
	return tagSlice, nil
}

func countHandler(dbs *sql.DB) (interface{}, *CountType, error) {

	mSuccess := metrics.GetOrRegisterGauge(`Inventory.Retrieve.Success`, nil)
	mCountErr := metrics.GetOrRegisterGauge("Inventory.Retrieve.Count-Error", nil)

	var count int

	row := dbs.QueryRow("SELECT count(*) FROM " + tagsTable)
	err := row.Scan(&count)
	if err != nil {
		mCountErr.Update(1)
		return nil, nil, err
	}

	mSuccess.Update(1)
	return nil, &CountType{Count: &count}, nil
}

// Value implements driver.Valuer interfaces
func (tag *Tag) Value() (driver.Value, error) {
	return json.Marshal(tag)
}

// Scan implements sql.Scanner interfaces
func (tag *Tag) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, tag)
}

// FindByEpc searches DB for tag based on the epc value
// Returns the tag if found or empty tag if it does not exist
func FindByEpc(dbs *sql.DB, epc string) (Tag, error) {

	// Metrics
	metrics.GetOrRegisterGauge(`Inventory.FindByEpc.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Inventory.FindByEpc.Success`, nil)
	mFindByEpcErr := metrics.GetOrRegisterGauge("Inventory.FindByEpc.Find-Error", nil)
	mFindLatency := metrics.GetOrRegisterTimer(`Inventory.FindByEpc.Find-Latency`, nil)

	var tag Tag

	selectQuery := fmt.Sprintf(`SELECT %s FROM %s WHERE %s ->> %s = %s LIMIT 1`,
		pq.QuoteIdentifier(jsonb),
		pq.QuoteIdentifier(tagsTable),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteLiteral(epcColumn),
		pq.QuoteLiteral(epc),
	)

	retrieveTimer := time.Now()
	if err := dbs.QueryRow(selectQuery).Scan(&tag); err != nil {

		if err == sql.ErrNoRows {
			return Tag{}, nil
		}

		mFindByEpcErr.Update(1)
		return Tag{}, err
	}

	mFindLatency.Update(time.Since(retrieveTimer))

	mSuccess.Update(1)
	return tag, nil
}

// Replace bulk upserts tags into database
func Replace(dbs *sql.DB, tagData []Tag) error {

	// Metrics
	metrics.GetOrRegisterGauge(`Inventory.Replace.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Inventory.Replace.Success`, nil)
	mValidationErr := metrics.GetOrRegisterGauge(`Inventory.Replace.Validation-Error`, nil)
	mBulkErr := metrics.GetOrRegisterGauge(`Inventory.Replace.Bulk-Error`, nil)

	if len(tagData) == 0 {
		return nil
	}

	for _, tag := range tagData {
		if tag.Epc == "" {
			mValidationErr.Update(1)
			return errors.Wrap(web.ErrValidation, "Unable to add new tag with empty EPC code")
		}

		obj, err := json.Marshal(tag)
		if err != nil {
			return err
		}

		upsertClause := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s) 
									 ON CONFLICT (( %s  ->> %s )) 
									 DO UPDATE SET %s = %s.%s || %s; `,
			pq.QuoteIdentifier(tagsTable),
			pq.QuoteIdentifier(jsonb),
			pq.QuoteLiteral(string(obj)),
			pq.QuoteIdentifier(jsonb),
			pq.QuoteLiteral(epcColumn),
			pq.QuoteIdentifier(jsonb),
			pq.QuoteIdentifier(tagsTable),
			pq.QuoteIdentifier(jsonb),
			pq.QuoteLiteral(string(obj)),
		)

		_, err = dbs.Exec(upsertClause)
		if err != nil {
			mBulkErr.Update(1)
			return err
		}
	}
	mSuccess.Update(1)
	return nil
}

// Delete removes tag from database based on epc
// nolint :dupl
func Delete(dbs *sql.DB, epc string) error {

	// Metrics
	metrics.GetOrRegisterGauge(`Inventory.Delete.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Inventory.Delete.Success`, nil)
	mDeleteErr := metrics.GetOrRegisterGauge(`Inventory.Delete.Delete-Error`, nil)
	mErrNotFound := metrics.GetOrRegisterGauge(`Inventory.Delete.NotFound-Error`, nil)
	mDeleteLatency := metrics.GetOrRegisterTimer(`Inventory.Delete.Delete-Latency`, nil)

	selectQuery := fmt.Sprintf(`DELETE FROM %s WHERE %s ->> %s = %s;`,
		pq.QuoteIdentifier(tagsTable),
		pq.QuoteIdentifier(jsonb),
		pq.QuoteLiteral(epcColumn),
		pq.QuoteLiteral(epc),
	)

	deleteTimer := time.Now()
	if _, err := dbs.Exec(selectQuery); err != nil {
		if err == sql.ErrNoRows {
			mErrNotFound.Update(1)
			return web.ErrNotFound
		}
		mDeleteErr.Update(1)
		return errors.Wrap(err, "error in deleting a tag")
	}
	mDeleteLatency.Update(time.Since(deleteTimer))

	mSuccess.Update(1)
	return nil
}

// DeleteTagCollection removes tag collection from database
// nolint :dupl
func DeleteTagCollection(dbs *sql.DB) error {

	// Metrics
	metrics.GetOrRegisterGauge(`Inventory.DeleteTagCollection.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Inventory.DeleteTagCollection.Success`, nil)
	mDeleteAllErr := metrics.GetOrRegisterGauge(`Inventory.DeleteTagCollection.DeleteTagCollection-Error`, nil)
	mErrNotFound := metrics.GetOrRegisterGauge(`Inventory.DeleteTagCollection.NotFound-Error`, nil)
	mDeleteAllLatency := metrics.GetOrRegisterTimer(`Inventory.DeleteTagCollection.DeleteTagCollection-Latency`, nil)

	selectQuery := fmt.Sprintf(`DELETE FROM %s;`,
		pq.QuoteIdentifier(tagsTable),
	)

	deleteTimer := time.Now()
	if _, err := dbs.Exec(selectQuery); err != nil {
		if err == sql.ErrNoRows {
			mErrNotFound.Update(1)
			return web.ErrNotFound
		}
		mDeleteAllErr.Update(1)
		return errors.Wrap(err, "error in deletion of all tags")
	}
	mDeleteAllLatency.Update(time.Since(deleteTimer))

	mSuccess.Update(1)
	return nil
}

// DecodeTagData extracts a ProductID and URI from tag data, according to the
// configured tag decoders. If none of the decoders can successfully decode the
// data, it returns `encodingInvalid` for both.
func DecodeTagData(tagData string) (productID string, URI string, err error) {
	productID, URI = encodingInvalid, encodingInvalid
	tagDataBytes, err := hex.DecodeString(tagData)
	if err != nil {
		err = errors.Wrap(err, "tag data is not valid hex")
		return
	}

	var decodingErrors []string
	for idx, decoder := range config.AppConfig.TagDecoders {
		if productID, URI, err = decoder.Decode(tagDataBytes); err == nil {
			return
		}
		decodingErrors = append(decodingErrors,
			fmt.Sprintf("decoder %d (%T) unable to decode tag data: %s",
				idx+1, decoder, err))
		gaugeName := fmt.Sprintf("Inventory.DecodeTagData.%T", decoder)
		metrics.GetOrRegisterGauge(gaugeName, nil).Update(1)
	}

	metrics.GetOrRegisterGauge(
		`Inventory.DecodeTagData.CalculateProductCodeError`, nil).Update(1)
	return encodingInvalid, encodingInvalid, errors.Errorf(
		"unable to decode tag data with any of the configured decoders:\n\t%s",
		strings.Join(decodingErrors, "\n\t"))
}

// Update updates a tag in the database
func Update(dbs *sql.DB, epc string, facilityId string, object map[string]string) error {

	// Metrics
	metrics.GetOrRegisterGauge(`Inventory.Update.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Inventory.Update.Success`, nil)
	mUpdateErr := metrics.GetOrRegisterGauge(`Inventory.Update.Update-Error`, nil)
	mErrNotFound := metrics.GetOrRegisterGauge(`Inventory.Update.NotFound-Error`, nil)
	mUpdateLatency := metrics.GetOrRegisterTimer(`Inventory.Update.Update-Latency`, nil)

	for key, value := range object {
		updateStmt := fmt.Sprintf(`UPDATE %s SET %s = jsonb_set(%s, '{%s}', '%s')
					WHERE (%s ->> %s = %s AND %s ->> %s = %s) returning %s;`,
			pq.QuoteIdentifier(tagsTable),
			pq.QuoteIdentifier(jsonb),
			pq.QuoteIdentifier(jsonb),
			pq.QuoteIdentifier(key),
			pq.QuoteIdentifier(value),
			pq.QuoteIdentifier(jsonb),
			pq.QuoteLiteral(epcColumn),
			pq.QuoteLiteral(epc),
			pq.QuoteIdentifier(jsonb),
			pq.QuoteLiteral(facilityColumn),
			pq.QuoteLiteral(facilityId),
			pq.QuoteLiteral(epcColumn),
		)

		updateTimer := time.Now()
		result, err := dbs.Exec(updateStmt)
		if err != nil {
			mUpdateErr.Update(1)
			return err
		} else {
			updatedRow, err := result.RowsAffected()
			{
				if err != nil {
					mUpdateErr.Update(1)
					return err
				}
				if updatedRow == 0 {
					mErrNotFound.Update(1)
					return web.ErrNotFound
				}
			}
		}
		mUpdateLatency.Update(time.Since(updateTimer))
	}

	mSuccess.Update(1)
	return nil
}
