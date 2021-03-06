/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/app/config"
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/app/dailyturn"
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/app/facility"
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/app/tag"
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/productdata"
	"github.com/intel/rsp-sw-toolkit-im-suite-utilities/helper"
	"github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const (
	historyTable = "dailyturnhistory"
)

func TestApplyConfidenceFacilitiesDontExist(t *testing.T) {
	result := buildProductData(0.0, 0.0, 0.0, 0.0, "00111111")
	testServer := buildTestServer(t, result)
	defer testServer.Close()

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	dailyInvPercConfig := config.AppConfig.DailyInventoryPercentage
	probUnreadToReadConfig := config.AppConfig.ProbUnreadToRead
	probInStoreConfig := config.AppConfig.ProbInStoreRead
	probExitErrorConfig := config.AppConfig.ProbExitError
	var tags = []tag.Tag{
		{
			FacilityID: "",
			LastRead:   helper.UnixMilli(time.Now().AddDate(0, 0, -1)),
		},
	}

	if err := ApplyConfidence(testDB.DB, tags, testServer.URL+"/skus"); err != nil {
		t.Fatalf("Error returned from applyConfidence %v", err)
	}
	for _, val := range tags {
		if !isProbabilisticPluginFound {
			if val.Confidence != 0 {
				t.Errorf("Confidence not set correctly when probabilistic plugin doesn't exit")
			}
		} else {
			configConf := confidenceCalc(dailyInvPercConfig,
				probUnreadToReadConfig,
				probInStoreConfig,
				probExitErrorConfig, val.LastRead)

			log.Warn(configConf)
			log.Warn(val.Confidence)

			if val.Confidence != configConf {
				t.Errorf("Confidence not set correctly for handheld data")
			}
		}

	}

}

func TestApplyConfidenceFacilitiesDontMatch(t *testing.T) {
	result := buildProductData(0.0, 0.0, 0.0, 0.0, "00111111")
	testServer := buildTestServer(t, result)
	defer testServer.Close()

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	dailyInvPercConfig := config.AppConfig.DailyInventoryPercentage
	probUnreadToReadConfig := config.AppConfig.ProbUnreadToRead
	probInStoreConfig := config.AppConfig.ProbInStoreRead
	probExitErrorConfig := config.AppConfig.ProbExitError
	var tags = []tag.Tag{
		{
			FacilityID: "Tavern",
			LastRead:   helper.UnixMilli(time.Now().AddDate(0, 0, -1)),
		},
	}

	insertFacilitiesHelper(t, testDB.DB)

	if err := ApplyConfidence(testDB.DB, tags, testServer.URL+"/skus"); err != nil {
		t.Fatalf("Error returned from applyConfidence %v", err)
	}

	for _, val := range tags {
		if !isProbabilisticPluginFound {
			if val.Confidence != 0 {
				t.Errorf("Confidence not set correctly when probabilistic plugin doesn't exit")
			}
		} else {
			configConf := confidenceCalc(
				dailyInvPercConfig,
				probUnreadToReadConfig,
				probInStoreConfig,
				probExitErrorConfig, val.LastRead)

			if val.Confidence != configConf {
				t.Errorf("Confidence not set correctly when no facility found")
			}
		}
	}

}

func TestApplyConfidenceProductIdCoeffOverridesFacilityCoeffMatch(t *testing.T) {
	result := buildProductData(0.2, 0.75, 0.2, 0.1, "00111111")
	testServer := buildTestServer(t, result)

	defer testServer.Close()

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	var tags = []tag.Tag{
		{
			FacilityID:      "Test",
			LastRead:        helper.UnixMilli(time.Now().AddDate(0, 0, -1)),
			Epc:             "30143639F84191AD22900204",
			EpcEncodeFormat: "tbd",
			ProductID:       "00111111",
			Event:           "cycle_count",
			LocationHistory: []tag.LocationHistory{
				{
					Location:  "RSP-950b44",
					Timestamp: 1506638821662,
					Source:    "fixed",
				}},
			Tid: "",
		},
	}

	insertFacilitiesHelper(t, testDB.DB)

	if err := ApplyConfidence(testDB.DB, tags, testServer.URL+"/skus"); err != nil {
		t.Fatalf("Error returned from applyConfidence %v", err)
	}

	for _, val := range tags {

		facilities, err := facility.CreateFacilityMap(testDB.DB)
		if err != nil {
			t.Fatalf("Couldn't create facilityItem map %v", err)
		}

		if !isProbabilisticPluginFound {
			if val.Confidence != 0 {
				t.Errorf("Confidence not set correctly when probabilistic plugin doesn't exit")
			}
		} else {
			facilityItem := facilities[val.FacilityID]
			facilityConf := confidenceCalc(
				facilityItem.Coefficients.DailyInventoryPercentage,
				facilityItem.Coefficients.ProbUnreadToRead,
				facilityItem.Coefficients.ProbInStoreRead,
				facilityItem.Coefficients.ProbExitError, val.LastRead)

			if val.Confidence == facilityConf {
				// product identifier coefficients should override facility coefficients, thus confidence should not be equal
				t.Error("Confidence not set correctly when product identifier has different coefficients than facility")
			}
		}
	}
}

func TestApplyConfidenceProductIdCoeffNull(t *testing.T) {
	result := productdata.Result{}
	testServer := buildTestServer(t, result)

	defer testServer.Close()

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	var tags = []tag.Tag{
		{
			FacilityID:      "Test",
			LastRead:        helper.UnixMilli(time.Now().AddDate(0, 0, -1)),
			Epc:             "30143639F84191AD22900204",
			EpcEncodeFormat: "tbd",
			ProductID:       "00111111",
			Event:           "cycle_count",
			LocationHistory: []tag.LocationHistory{
				{
					Location:  "RSP-950b44",
					Timestamp: 1506638821662,
					Source:    "fixed",
				}},
			Tid: "",
		},
	}

	insertFacilitiesHelper(t, testDB.DB)

	if err := ApplyConfidence(testDB.DB, tags, testServer.URL+"/skus"); err != nil {
		t.Fatalf("Error returned from applyConfidence %v", err)
	}

	for _, val := range tags {

		facilities, err := facility.CreateFacilityMap(testDB.DB)
		if err != nil {
			t.Fatalf("Couldn't create facilityItem map %v", err)
		}

		if !isProbabilisticPluginFound {
			if val.Confidence != 0 {
				t.Errorf("Confidence not set correctly when probabilistic plugin doesn't exit")
			}
		} else {
			facilityItem := facilities[val.FacilityID]
			facilityConf := confidenceCalc(
				facilityItem.Coefficients.DailyInventoryPercentage,
				facilityItem.Coefficients.ProbUnreadToRead,
				facilityItem.Coefficients.ProbInStoreRead,
				facilityItem.Coefficients.ProbExitError, val.LastRead)

			if val.Confidence != facilityConf {
				// product identifier coefficients should override facility coefficients, thus confidence should not be equal
				t.Error("Confidence not set correctly when product identifier has different coefficients than facility")
			}
		}
	}
}

func TestApplyConfidenceProductIdCoeffOverridesSomeNull(t *testing.T) {
	result := buildProductData(0.2, 0.75, 0, 0, "00111111")
	testServer := buildTestServer(t, result)

	defer testServer.Close()

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	var tags = []tag.Tag{
		{
			FacilityID:      "Test",
			LastRead:        helper.UnixMilli(time.Now().AddDate(0, 0, -1)),
			Epc:             "30143639F84191AD22900204",
			EpcEncodeFormat: "tbd",
			ProductID:       "00111111",
			Event:           "cycle_count",
			LocationHistory: []tag.LocationHistory{
				{
					Location:  "RSP-950b44",
					Timestamp: 1506638821662,
					Source:    "fixed",
				}},
			Tid: "",
		},
	}

	insertFacilitiesHelper(t, testDB.DB)

	if err := ApplyConfidence(testDB.DB, tags, testServer.URL+"/skus"); err != nil {
		t.Fatalf("Error returned from applyConfidence %v", err)
	}

	for _, val := range tags {

		facilities, err := facility.CreateFacilityMap(testDB.DB)
		if err != nil {
			t.Fatalf("Couldn't create facilityItem map %v", err)
		}
		if !isProbabilisticPluginFound {
			if val.Confidence != 0 {
				t.Errorf("Confidence not set correctly when probabilistic plugin doesn't exit")
			}
		} else {
			facilityItem := facilities[val.FacilityID]
			facilityConf := confidenceCalc(
				facilityItem.Coefficients.DailyInventoryPercentage,
				facilityItem.Coefficients.ProbUnreadToRead,
				facilityItem.Coefficients.ProbInStoreRead,
				facilityItem.Coefficients.ProbExitError, val.LastRead)

			if val.Confidence == facilityConf {
				// product identifier coefficients should override facility coefficients, thus confidence should not be equal
				t.Error("Confidence not set correctly when product identifier has different coefficients than facility")
			}
		}

	}
}

func TestApplyConfidenceFacilityCoeffMatch(t *testing.T) {
	result := buildProductData(0.0, 0.0, 0.0, 0.0, "00111111")
	testServer := buildTestServer(t, result)

	defer testServer.Close()

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	var tags = []tag.Tag{
		{
			FacilityID:      "Test",
			LastRead:        helper.UnixMilli(time.Now().AddDate(0, 0, -1)),
			Epc:             "30143639F84191AD22900204",
			EpcEncodeFormat: "tbd",
			ProductID:       "00111111",
			Event:           "cycle_count",
			LocationHistory: []tag.LocationHistory{
				{
					Location:  "RSP-950b44",
					Timestamp: 1506638821662,
					Source:    "fixed",
				}},
			Tid: "",
		},
	}

	insertFacilitiesHelper(t, testDB.DB)

	if err := ApplyConfidence(testDB.DB, tags, testServer.URL+"/skus"); err != nil {
		t.Fatalf("Error returned from applyConfidence %v", err)
	}

	for _, val := range tags {

		facilities, err := facility.CreateFacilityMap(testDB.DB)
		if err != nil {
			t.Fatalf("Couldn't create facilityItem map %v", err)
		}

		if !isProbabilisticPluginFound {
			if val.Confidence != 0 {
				t.Errorf("Confidence not set correctly when probabilistic plugin doesn't exit")
			}
		} else {

			facilityItem := facilities[val.FacilityID]
			facilityConf := confidenceCalc(
				facilityItem.Coefficients.DailyInventoryPercentage,
				facilityItem.Coefficients.ProbUnreadToRead,
				facilityItem.Coefficients.ProbInStoreRead,
				facilityItem.Coefficients.ProbExitError, val.LastRead)

			if val.Confidence != facilityConf {
				// product identifier coefficients should not override facility coefficients when they are equal to 0
				// thus confidence should be equal
				t.Error("Confidence not set correctly when product identifier has different coefficients than facility")
			}
		}

	}
}

func TestApplyConfidenceMixedTags(t *testing.T) {
	result := buildProductData(0.0, 0.0, 0.0, 0.0, "00111111")
	testServer := buildTestServer(t, result)

	defer testServer.Close()

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	var tags = []tag.Tag{
		{
			FacilityID: "Test",
			LastRead:   helper.UnixMilli(time.Now().AddDate(0, 0, -1)),
		},
		{
			FacilityID: "TestNoFacility",
			LastRead:   helper.UnixMilli(time.Now().AddDate(0, 0, -1)),
		},
	}

	insertFacilitiesHelper(t, testDB.DB)

	if confErr := ApplyConfidence(testDB.DB, tags, testServer.URL+"/skus"); confErr != nil {
		t.Fatalf("Error returned from applyConfidence %v", confErr)
	}

	var facilityConf float64

	dailyInvPercConfig := config.AppConfig.DailyInventoryPercentage
	probUnreadToReadConfig := config.AppConfig.ProbUnreadToRead
	probInStoreConfig := config.AppConfig.ProbInStoreRead
	probExitErrorConfig := config.AppConfig.ProbExitError
	facilities, err := facility.CreateFacilityMap(testDB.DB)

	if err != nil {
		t.Fatalf("Couldn't create facility map %v", err)
	}
	for _, val := range tags {

		if !isProbabilisticPluginFound {
			if val.Confidence != 0 {
				t.Errorf("Confidence not set correctly when probabilistic plugin doesn't exit")
			}
		} else {

			fac, foundFacility := facilities[val.FacilityID]
			if foundFacility {
				facilityConf = confidenceCalc(
					fac.Coefficients.DailyInventoryPercentage,
					fac.Coefficients.ProbUnreadToRead,
					fac.Coefficients.ProbInStoreRead,
					fac.Coefficients.ProbExitError, val.LastRead)
			} else {
				facilityConf = confidenceCalc(
					dailyInvPercConfig,
					probUnreadToReadConfig,
					probInStoreConfig,
					probExitErrorConfig, val.LastRead)
			}
			if val.Confidence != facilityConf {
				t.Error("Confidence not set correctly when facility found")
			}
		}

	}
}

func TestApplyConfidenceWithDailyTurn(t *testing.T) {

	result := buildProductData(0.0, 0.0, 0.0, 0.0, "00111111")
	testServer := buildTestServer(t, result)

	defer testServer.Close()

	testDB := dbHost.CreateDB(t)
	defer testDB.Close()

	clearDailyTurnHistory(t, testDB.DB)

	productId := "00111111"

	var tags = []tag.Tag{
		// Facility config present and computed daily turn present
		{
			FacilityID: "Test",
			LastRead:   helper.UnixMilli(time.Now().AddDate(0, 0, -3)),
			Epc:        t.Name() + "_epc1",
			ProductID:  productId,
		},
		// Facility config present and computed daily turn present
		{
			FacilityID: "Test",
			LastRead:   helper.UnixMilli(time.Now().AddDate(0, 0, -6)),
			Epc:        t.Name() + "_epc2",
			ProductID:  productId,
		},
		// Facility config NOT present and computed daily turn present
		{
			FacilityID: t.Name() + "_NoFacility",
			LastRead:   helper.UnixMilli(time.Now().AddDate(0, 0, -5)),
			Epc:        t.Name() + "_epc3",
			ProductID:  productId,
		},
		// Facility config NOT present and computed daily turn NOT present (ie. defaults)
		{
			FacilityID: t.Name() + "_NoFacility",
			LastRead:   helper.UnixMilli(time.Now().AddDate(0, 0, -7)),
			Epc:        t.Name() + "_epc4",
			ProductID:  "NotFound",
		},
	}

	computedDailyTurn := 0.5
	insertSampleDailyTurnHistory(t, testDB.DB, productId, computedDailyTurn)

	dailyInvPercConfig := config.AppConfig.DailyInventoryPercentage
	probUnreadToReadConfig := config.AppConfig.ProbUnreadToRead
	probInStoreConfig := config.AppConfig.ProbInStoreRead
	probExitErrorConfig := config.AppConfig.ProbExitError

	insertFacilitiesHelper(t, testDB.DB)

	if confErr := ApplyConfidence(testDB.DB, tags, testServer.URL+"/skus"); confErr != nil {
		t.Fatalf("Error returned from applyConfidence %v", confErr)
	}

	facilities, err := facility.CreateFacilityMap(testDB.DB)
	if err != nil {
		t.Fatalf("Couldn't create facility map %v", err)
	}

	for _, val := range tags {
		if !isProbabilisticPluginFound {
			if val.Confidence != 0 {
				t.Errorf("Confidence not set correctly when probabilistic plugin doesn't exit")
			}
		} else {
			fac, foundFacility := facilities[val.FacilityID]
			if foundFacility {
				facilityConf := confidenceCalc(
					fac.Coefficients.DailyInventoryPercentage,
					fac.Coefficients.ProbUnreadToRead,
					fac.Coefficients.ProbInStoreRead,
					fac.Coefficients.ProbExitError,
					val.LastRead)

				if val.Confidence == facilityConf {
					t.Error("Confidence not set correctly when computed daily turn is present and facility is found")
				}

				expectedConf := confidenceCalc(
					computedDailyTurn,
					fac.Coefficients.ProbUnreadToRead,
					fac.Coefficients.ProbInStoreRead,
					fac.Coefficients.ProbExitError,
					val.LastRead)

				if val.Confidence != expectedConf {
					t.Error("Confidence not set correctly when computed daily turn is present and facility is found")
				}
			} else {
				dailyTurnConfidence := confidenceCalc(
					computedDailyTurn,
					probUnreadToReadConfig,
					probInStoreConfig,
					probExitErrorConfig,
					val.LastRead)

				defaultConfidence := confidenceCalc(
					dailyInvPercConfig,
					probUnreadToReadConfig,
					probInStoreConfig,
					probExitErrorConfig,
					val.LastRead)

				if defaultConfidence == dailyTurnConfidence {
					t.Error("Daily turn confidence and default confidence are the same value. This should not happen and means the test is invalid")
				}

				if val.ProductID == productId {
					if val.Confidence != dailyTurnConfidence {
						t.Error("Confidence not set correctly when computed daily turn is present and no facility found")
					}
				} else {
					if val.Confidence != defaultConfidence {
						t.Error("Confidence not set correctly when no facility found and no computed daily turn is present")
					}
				}
			}
		}

		clearDailyTurnHistory(t, testDB.DB)
	}
}

func buildTestServer(t *testing.T, result productdata.Result) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.EscapedPath() != "/skus" {
			t.Errorf("Expected request to be '/skus', received %s",
				request.URL.EscapedPath())
		}
		if request.Method != "GET" {
			t.Errorf("Expected 'GET' request, received '%s", request.Method)
		}
		var jsonData []byte
		if request.URL.EscapedPath() == "/skus" {
			jsonData, _ = json.Marshal(result)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(jsonData)
	}))
}

func insertSampleDailyTurnHistory(t *testing.T, db *sql.DB, productId string, dailyTurn float64) {
	var history dailyturn.History
	history.ProductID = productId
	history.DailyTurn = dailyTurn

	if err := dailyturn.Upsert(db, history); err != nil {
		t.Error("Unable to upsert daily turn history")
	}
}

func clearDailyTurnHistory(t *testing.T, db *sql.DB) {
	selectQuery := fmt.Sprintf(`DELETE FROM %s`,
		pq.QuoteIdentifier(historyTable),
	)

	_, err := db.Exec(selectQuery)
	if err != nil {
		t.Errorf("Unable to delete data from %s table: %s", historyTable, err)
	}
}

func buildProductData(becomingReadable float64, beingRead float64, dailyTurn float64, exitError float64, gtinSku string) productdata.Result {

	productIDMetadata := productdata.ProductMetadata{
		ProductID:        gtinSku,
		BecomingReadable: becomingReadable,
		BeingRead:        beingRead,
		DailyTurn:        dailyTurn,
		ExitError:        exitError,
	}

	productIDList := []productdata.ProductMetadata{productIDMetadata}

	var data = productdata.ProdData{
		Sku:         gtinSku,
		ProductList: productIDList,
	}

	dataList := []productdata.ProdData{data}

	var result = productdata.Result{
		ProdData: dataList,
	}
	return result
}
