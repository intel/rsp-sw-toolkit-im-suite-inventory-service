/*
 * INTEL CONFIDENTIAL
 * Copyright (2018) Intel Corporation.
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

package dailyturn

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/tag"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/integrationtest"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
	"math"
	"os"
	"reflect"
	"strconv"
	"testing"
)

var (
	// epsilon is used to compare floating point numbers to each other
	epsilon = math.Nextafter(1.0, 2.0) - 1.0
)

var dbHost integrationtest.DBHost

func TestMain(m *testing.M) {
	dbHost = integrationtest.InitHost("dailyTurn_test")
	os.Exit(m.Run())
}

func TestFindHistoryByProductId(t *testing.T) {
	masterDb := dbHost.CreateDB(t)
	defer masterDb.Close()

	clearAllData(t, masterDb)

	productId := t.Name()
	insertSampleHistory(t, masterDb, productId, 0)

	if _, err := FindHistoryByProductId(masterDb, productId); err != nil {
		t.Error("Unable to query find history by productId", err.Error())
	}
}

func TestProcessIncomingASNList(t *testing.T) {
	masterDb := dbHost.CreateDB(t)
	defer masterDb.Close()

	clearAllData(t, masterDb)
	asnList := []tag.AdvanceShippingNotice{
		{
			EventTime: strconv.Itoa(int(helper.UnixMilliNow())),
			Items: []tag.ASNInputItem{
				{
					ItemGTIN: t.Name() + "_1",
					EPCs:     make([]string, 21),
				},
				{
					ItemGTIN: t.Name() + "_2",
					EPCs:     make([]string, 57),
				},
				{
					ItemGTIN: t.Name() + "_3",
					EPCs:     make([]string, 93),
				},
			},
		},
	}

	productIds := []string{t.Name() + "_1", t.Name() + "_2", t.Name() + "_3"}
	var oldTimestamp int64

	for _, productId := range productIds {
		history, _ := FindHistoryByProductId(masterDb, productId)
		if !reflect.DeepEqual(history, History{}) {
			t.Fatalf("History is expected to be empty")
		}
	}

	// insert some data to ensure daily turn is computed and not skipped
	for _, productId := range productIds {
		if err := insertTags(t, masterDb, productId, 100, 25, helper.UnixMilliNow()); err != nil {
			t.Fatal("Unable to insert sample tags into the database")
		}
	}

	ProcessIncomingASNList(masterDb, asnList)

	for _, productId := range productIds {
		history, _ := FindHistoryByProductId(masterDb, productId)
		if len(history.Records) != 0 || history.Timestamp < 1 {
			t.Fatalf("Expected history records to be 0 and timestamp to be > 1: %d, %d",
				len(history.Records), history.Timestamp)
		}

		// spoof timestamp
		history.Timestamp -= int64(2 * millisecondsInDay)
		if err := Upsert(masterDb, history); err != nil {
			t.Fatal("Unexpected error upserting data")
		}
		oldTimestamp = history.Timestamp
	}

	ProcessIncomingASNList(masterDb, asnList)

	for _, productId := range productIds {
		history, _ := FindHistoryByProductId(masterDb, productId)
		if len(history.Records) != 1 || history.Timestamp <= oldTimestamp {
			t.Fatalf("Expected history records to be 1 and timestamp to be > oldTimestamp: %d, %d, %d",
				len(history.Records), history.Timestamp, oldTimestamp)
		}

		// spoof timestamp
		history.Timestamp -= int64(2 * millisecondsInDay)
		if err := Upsert(masterDb, history); err != nil {
			t.Fatal("Unexpected error upserting data")
		}
		oldTimestamp = history.Timestamp
	}

	ProcessIncomingASNList(masterDb, asnList)

	for _, productId := range productIds {
		history, _ := FindHistoryByProductId(masterDb, productId)
		if len(history.Records) != 2 || history.Timestamp <= oldTimestamp {
			t.Fatalf("Expected history records to be 2 and timestamp to be > oldTimestamp: %d, %d, %d",
				len(history.Records), history.Timestamp, oldTimestamp)
		}
	}
}

func TestProcessIncomingASNList_TruncateHistoryRecords(t *testing.T) {
	masterDb := dbHost.CreateDB(t)
	defer masterDb.Close()

	clearAllData(t, masterDb)

	productId := t.Name()
	maxRecords := config.AppConfig.DailyTurnHistoryMaximum

	if err := insertTags(t, masterDb, productId, 100, 25, helper.UnixMilliNow()); err != nil {
		t.Fatal("Unable to insert sample tags into the database")
	}

	for i := 0; i < maxRecords*2; i++ {
		asnList := []tag.AdvanceShippingNotice{
			{
				EventTime: strconv.Itoa(int(helper.UnixMilliNow())),
				Items: []tag.ASNInputItem{
					{
						ItemGTIN: productId,
						EPCs:     make([]string, 1),
					},
				},
			},
		}

		ProcessIncomingASNList(masterDb, asnList)

		history, _ := FindHistoryByProductId(masterDb, productId)
		if len(history.Records) > maxRecords {
			t.Fatalf("Too many history records were found: %d. Expected to limit records to %d", len(history.Records), maxRecords)
		}

		// spoof timestamp
		history.Timestamp -= int64(2 * millisecondsInDay)
		if err := Upsert(masterDb, history); err != nil {
			t.Fatal("Unexpected error upserting data")
		}
	}
}

func TestRecord_ComputeDailyTurn(t *testing.T) {
	now := helper.UnixMilliNow()

	record := Record{
		Timestamp:         now,
		PreviousTimestamp: now - int64(millisecondsInDay),
		Departed:          100,
		Present:           300,
	}

	// duration is 1 day, 400 total, 100 departed => 1/4 departed per day, aka 0.25 daily turn
	expected := float64(0.25)

	if err := record.ComputeDailyTurn(); err != nil {
		t.Fatalf("unexpected error computing daily turn: %v", err.Error())
	}
	if math.Abs(record.DailyTurn-expected) > epsilon {
		t.Fatalf("Computed daily turn value of %f is not equal to the expected value of %f", record.DailyTurn, expected)
	}
}

func TestRecord_ComputeDailyTurn2(t *testing.T) {
	now := helper.UnixMilliNow()
	days := 3.5

	record := Record{
		Timestamp:         now,
		PreviousTimestamp: now - int64(days*millisecondsInDay),
		Departed:          237,
		Present:           958,
	}

	// duration is 3.5 days, 958 + 237 total, 237 departed => 237/1195 departed per 3.5 days => ~ 0.56666
	expected := float64(float64(record.Departed)/float64(record.Present+record.Departed)) / days

	if err := record.ComputeDailyTurn(); err != nil {
		t.Fatalf("unexpected error computing daily turn: %v", err.Error())
	}
	if math.Abs(record.DailyTurn-expected) > epsilon {
		t.Fatalf("Computed daily turn value of %f is not equal to the expected value of %f", record.DailyTurn, expected)
	}
}

func TestRecord_ComputeDailyTurn_ErrTimeTooShort(t *testing.T) {
	now := helper.UnixMilliNow()

	record := Record{
		Timestamp:         now,
		PreviousTimestamp: now - 1000,
		Departed:          100,
		Present:           300,
	}

	if err := record.ComputeDailyTurn(); err != ErrTimeTooShort {
		t.Fatalf("Expected error ErrTimeTooShort computing daily turn, but got: %v", err)
	}
}

func TestRecord_ComputeDailyTurn_ErrNoInventory(t *testing.T) {
	now := helper.UnixMilliNow()

	record := Record{
		Timestamp:         now,
		PreviousTimestamp: now - int64(millisecondsInDay),
		Departed:          0,
		Present:           0,
	}

	if err := record.ComputeDailyTurn(); err != ErrNoInventory {
		t.Fatalf("Expected error ErrNoInventory computing daily turn, but got: %v", err)
	}
}

func TestDailyTurnMinimumDataPoints(t *testing.T) {
	masterDb := dbHost.CreateDB(t)
	defer masterDb.Close()

	clearAllData(t, masterDb)
	productId := t.Name()

	now := helper.UnixMilliNow()

	history := History{
		DailyTurn: 0,
		Timestamp: now,
		ProductID: productId,
		Records:   []Record{},
	}

	config.AppConfig.DailyTurnMinimumDataPoints = 0
	history.ComputeAverageDailyTurn()
	// make sure we do not try and compute with 0 data points
	if history.DailyTurn > epsilon {
		t.Fatalf("Expected daily turn to be 0.0, but got %f", history.DailyTurn)
	}

	// set limit to 5 records, but only add 4 records
	config.AppConfig.DailyTurnMinimumDataPoints = 5
	history.Records = append(history.Records, Record{DailyTurn: 0.35})
	history.Records = append(history.Records, Record{DailyTurn: 0.35})
	history.Records = append(history.Records, Record{DailyTurn: 0.35})
	history.Records = append(history.Records, Record{DailyTurn: 0.35})
	history.ComputeAverageDailyTurn()
	// make sure we do not try and compute with 0 data points
	if history.DailyTurn > epsilon {
		t.Fatalf("Expected daily turn to be 0.0, but got %f", history.DailyTurn)
	}

	// lower the limit down to 4 and a value should be computed
	config.AppConfig.DailyTurnMinimumDataPoints = 4
	history.ComputeAverageDailyTurn()
	// make sure we do not try and compute with 0 data points
	if history.DailyTurn <= epsilon {
		t.Fatalf("Expected daily turn to be non-zero, but got %f", history.DailyTurn)
	}
}

func TestHistory_ComputeAverageDailyTurnMean(t *testing.T) {
	masterDb := dbHost.CreateDB(t)
	defer masterDb.Close()

	clearAllData(t, masterDb)
	productId := t.Name()

	now := helper.UnixMilliNow()
	config.AppConfig.DailyTurnComputeUsingMedian = false
	config.AppConfig.DailyTurnMinimumDataPoints = 2

	history := History{
		DailyTurn: 0,
		Timestamp: now,
		ProductID: productId,
		Records: []Record{
			{
				DailyTurn: 0.25,
			},
		},
	}

	history.ComputeAverageDailyTurn()
	// make sure we do not compute until we hit the configured minimum data points
	if history.DailyTurn > epsilon {
		t.Fatalf("Expected daily turn to be 0.0, but got %f", history.DailyTurn)
	}

	history.Records = append(history.Records, Record{DailyTurn: 0.35})
	expected := (0.25 + 0.35) / 2
	history.ComputeAverageDailyTurn()
	if math.Abs(expected-history.DailyTurn) > epsilon {
		t.Fatalf("Expected daily turn to be %f, but got %f", expected, history.DailyTurn)
	}

	history.Records = append(history.Records, Record{DailyTurn: 0.50})
	expected = (2*expected + 0.50) / 3
	history.ComputeAverageDailyTurn()
	if math.Abs(expected-history.DailyTurn) > epsilon {
		t.Fatalf("Expected daily turn to be %f, but got %f", expected, history.DailyTurn)
	}

	history.Records = append(history.Records, Record{DailyTurn: 0.47})
	expected = (3*expected + 0.47) / 4
	history.ComputeAverageDailyTurn()
	if math.Abs(expected-history.DailyTurn) > epsilon {
		t.Fatalf("Expected daily turn to be %f, but got %f", expected, history.DailyTurn)
	}
}

func TestHistory_ComputeAverageDailyTurnMedian(t *testing.T) {
	masterDb := dbHost.CreateDB(t)
	defer masterDb.Close()

	clearAllData(t, masterDb)
	productId := t.Name()

	now := helper.UnixMilliNow()
	config.AppConfig.DailyTurnComputeUsingMedian = true
	config.AppConfig.DailyTurnMinimumDataPoints = 2

	history := History{
		DailyTurn: 0,
		Timestamp: now,
		ProductID: productId,
		Records: []Record{
			{
				DailyTurn: 0.25,
			},
		},
	}

	history.ComputeAverageDailyTurn()
	// make sure we do not compute until we hit the configured minimum data points
	if history.DailyTurn > epsilon {
		t.Fatalf("Expected daily turn to be 0.0, but got %f", history.DailyTurn)
	}

	history.Records = append(history.Records, Record{DailyTurn: 0.35})
	expected := (0.25 + 0.35) / 2
	history.ComputeAverageDailyTurn()
	if math.Abs(expected-history.DailyTurn) > epsilon {
		t.Fatalf("Expected daily turn to be %f, but got %f", expected, history.DailyTurn)
	}

	history.Records = append(history.Records, Record{DailyTurn: 0.50})
	// median
	expected = 0.35
	history.ComputeAverageDailyTurn()
	if math.Abs(expected-history.DailyTurn) > epsilon {
		t.Fatalf("Expected daily turn to be %f, but got %f", expected, history.DailyTurn)
	}

	history.Records = append(history.Records, Record{DailyTurn: 0.47})
	// median
	expected = (0.35 + 0.47) / 2
	history.ComputeAverageDailyTurn()
	if math.Abs(expected-history.DailyTurn) > epsilon {
		t.Fatalf("Expected daily turn to be %f, but got %f", expected, history.DailyTurn)
	}
}

func TestFindPresentAndDepartedTagsSinceTimestamp(t *testing.T) {
	masterDb := dbHost.CreateDB(t)
	defer masterDb.Close()

	clearAllData(t, masterDb)

	productId := t.Name()

	expected := PresentOrDepartedResults{
		DepartedTags: 30,
		PresentTags:  70,
	}
	if err := insertTags(t, masterDb, productId, expected.PresentTags+expected.DepartedTags, expected.DepartedTags, helper.UnixMilliNow()); err != nil {
		t.Fatal("Unable to insert sample tags into the database")
	}

	// we should find all of the tags that we put in. this is searching for departed tags since yesterday
	result, err := FindPresentOrDepartedTagsSinceTimestamp(masterDb, productId, helper.UnixMilliNow()-int64(millisecondsInDay))
	if err != nil {
		t.Fatalf("Unable to find present or departed tags: %v", err.Error())
	}
	if result.PresentTags != expected.PresentTags {
		t.Fatalf("Present tags %d was not the expected %d", result.PresentTags, expected.PresentTags)
	}
	if result.DepartedTags != expected.DepartedTags {
		t.Fatalf("Departed tags %d was not the expected %d", result.DepartedTags, expected.DepartedTags)
	}

	// test that this will find 0 departed tags (because our timestamp is in the future)
	result, err = FindPresentOrDepartedTagsSinceTimestamp(masterDb, productId, helper.UnixMilliNow()+int64(millisecondsInDay))
	if err != nil {
		t.Fatalf("Unable to find present or departed tags: %v", err.Error())
	}
	if result.DepartedTags != 0 {
		t.Fatalf("Departed tags %d was not the expected 0", result.DepartedTags)
	}

	// insert tags with no last read
	clearAllData(t, masterDb)
	if err := insertTags(t, masterDb, productId, 500, 100, 0); err != nil {
		t.Fatal("Unable to insert sample tags into the database")
	}
	// test that tags with a last_read of 0 are ignored
	result, err = FindPresentOrDepartedTagsSinceTimestamp(masterDb, productId, helper.UnixMilliNow()-int64(millisecondsInDay))
	if err != nil {
		t.Fatalf("Unable to find present or departed tags: %v", err.Error())
	}
	// we expect the present tag count to stay the same (ignore tags with last_read=0)
	if result.PresentTags != 0 {
		t.Fatalf("Present tags %d was not the expected %d", result.PresentTags, 0)
	}
	// we expect the departed tag count to stay the same (ignore tags with last_read=0)
	if result.DepartedTags != 0 {
		t.Fatalf("Departed tags %d was not the expected %d", result.DepartedTags, 0)
	}
}

func insertSampleHistory(t *testing.T, db *sql.DB, sampleID string, lastTimestamp int64) {
	var history History
	history.ProductID = sampleID
	history.Timestamp = lastTimestamp

	if err := Upsert(db, history); err != nil {
		t.Error("Unable to upsert history")
	}
}

func insertTags(t *testing.T, db *sql.DB, productId string, tagCount int, departedCount int, lastRead int64) error {
	tags := make([]tag.Tag, tagCount)
	for i, tagItem := range tags {
		tagItem.Epc = productId + strconv.Itoa(+tagCount+i)
		tagItem.ProductID = productId
		tagItem.LastRead = lastRead
		if i < departedCount {
			tagItem.Event = departedEvent
		}
		tags[i] = tagItem

		obj, err := json.Marshal(tags[i])
		if err != nil {
			return err
		}

		insertStmt := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`,
			pq.QuoteIdentifier(tagsTable),
			pq.QuoteIdentifier(jsonb),
			pq.QuoteLiteral(string(obj)),
		)

		_, err = db.Exec(insertStmt)
		if err != nil {
			t.Error(err.Error())
			return errors.Wrap(err, "db.tag.insert()")
		}
	}
	return nil
}

//nolint: dupl
func clearAllData(t *testing.T, db *sql.DB) {
	selectQuery := fmt.Sprintf(`DELETE FROM %s`,
		pq.QuoteIdentifier(historyTable),
	)
	_, err := db.Exec(selectQuery)
	if err != nil {
		t.Errorf("Unable to delete data from %s table: %s", historyTable, err)
	}

	selectQuery = fmt.Sprintf(`DELETE FROM %s`,
		pq.QuoteIdentifier(tagsTable),
	)
	_, err = db.Exec(selectQuery)
	if err != nil {
		t.Errorf("Unable to delete data from %s table: %s", tagsTable, err)
	}
}
