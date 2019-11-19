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

package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/cloudconnector/event"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
	"io"
	"net/http"
	"plugin"
	"reflect"
	"strconv"
	"strings"
	"time"

	//"time"

	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/dailyturn"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/statemodel"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/facility"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/routes/schemas"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/tag"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/web"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/productdata"
)

// ApplyConfidence calculates the confidence to each tag using the facility coefficients
// this function can be reused by endpoints with odata for multiple facilities
func ApplyConfidence(session *sql.DB, tags []tag.Tag, url string) error {

	if len(tags) == 0 {
		return nil
	}

	// Getting coefficients from database by facilityID
	facilities, err := facility.CreateFacilityMap(session)
	if err != nil {
		return err
	}

	// Getting coefficients for gtin from sku-mapping service
	productDataMap, err := productdata.CreateProductDataMap(url)
	if err != nil {
		return err
	}

	// Create lookup map for computed daily turn values
	var computedDailyTurnMap map[string]dailyturn.History
	if config.AppConfig.UseComputedDailyTurnInConfidence {
		computedDailyTurnMap = dailyturn.CreateHistoryMap(session, tags)
	}

	var dailyInvPerc float64
	var probUnreadToRead float64
	var probInStore float64
	var probExitError float64

	for i := 0; i < len(tags); i++ {
		// Get coefficients
		facilityID := tags[i].FacilityID
		tagFacility, foundFacility := facilities[facilityID]
		lastRead := tags[i].LastRead
		if foundFacility {
			dailyInvPerc = tagFacility.Coefficients.DailyInventoryPercentage
			probUnreadToRead = tagFacility.Coefficients.ProbUnreadToRead
			probInStore = tagFacility.Coefficients.ProbInStoreRead
			probExitError = tagFacility.Coefficients.ProbExitError
		} else {
			dailyInvPerc = config.AppConfig.DailyInventoryPercentage
			probUnreadToRead = config.AppConfig.ProbUnreadToRead
			probInStore = config.AppConfig.ProbInStoreRead
			probExitError = config.AppConfig.ProbExitError
		}
		gtin := tags[i].ProductID

		product, foundProduct := productDataMap[gtin]
		if foundProduct {
			log.Debugf("Found product: %s", product.ProductID)
			// Only override if value isn't 0
			if product.BecomingReadable != 0 {
				probUnreadToRead = product.BecomingReadable
			}
			if product.BeingRead != 0 {
				// Only override if value isn't 0
				probInStore = product.BeingRead
			}
			if product.ExitError != 0 {
				// Only override if value isn't 0
				probExitError = product.ExitError
			}
			if product.DailyTurn != 0 {
				// Only override if value isn't 0
				dailyInvPerc = product.DailyTurn
			}
		}

		// Only override if enabled in config
		if config.AppConfig.UseComputedDailyTurnInConfidence {
			history, foundHistory := computedDailyTurnMap[gtin]
			if foundHistory && history.DailyTurn != 0 {
				// Only override if value isn't 0
				dailyInvPerc = history.DailyTurn
			}
		}

		log.Tracef("DailyInvPerc = %f, probUnreadToRead = %f, probInStore = %f, probExitError = %f",
			dailyInvPerc, probUnreadToRead, probInStore, probExitError)
		tags[i].Confidence = confidenceCalc(dailyInvPerc, probUnreadToRead, probInStore, probExitError, lastRead)
	}
	return nil
}

type confidenceFunc func(float64, float64, float64, float64, int64) float64

var confidenceCalc = confidenceFunc(zeroConfidence)

func zeroConfidence(_, _, _, _ float64, _ int64) float64 {
	return 0.0
}

func loadConfidencePlugin() error {
	confidencePlugin, err := plugin.Open("/plugin/inventory-probabilistic-algo")
	if err != nil {
		return errors.New("Intel Probabilistic Algorithm plugin not found; all Confidence values will be set to 0.")
	}
	calculateConfidence, err := confidencePlugin.Lookup("CalculateConfidence")
	if err != nil {
		return errors.New("Unable to find CalculateConfidence function in plugin")
	}
	// panics if this plugin & function exists but signature doesn't match
	confidenceCalc = calculateConfidence.(func(float64, float64, float64, float64, int64) float64)
	return nil
}

func init() {
	if err := loadConfidencePlugin(); err != nil {
		log.Error(err)
	}
}

func UpdateForCycleCount(tags []tag.Tag) {
	for i := 0; i < len(tags); i++ {
		if (tags)[i].CycleCount {
			(tags)[i].Event = statemodel.CycleCountEvent
		}
	}
}

// processGetRequest handles the request for retrieving tags
//nolint:lll
func processGetRequest(ctx context.Context, schema string, masterDB *sql.DB, request *http.Request, writer http.ResponseWriter, url string) error {

	// Metrics
	metrics.GetOrRegisterGauge("Inventory.processGetRequest.Attempt", nil).Update(1)

	startTime := time.Now()
	defer metrics.GetOrRegisterTimer("Inventory.processGetRequest.Latency", nil).Update(time.Since(startTime))

	mSuccess := metrics.GetOrRegisterGauge("Inventory.GetTags.Success", nil)
	mRetrieveErr := metrics.GetOrRegisterGauge("Inventory.GetTags.Retrieve-Error", nil)

	var mapping tag.RequestBody

	validationErrors, err := readAndValidateRequest(request, schema, &mapping)

	if err != nil {
		return err
	}

	if validationErrors != nil {
		web.Respond(ctx, writer, validationErrors, http.StatusBadRequest)
		return nil
	}

	odataMap := make(map[string][]string)
	odataMap = mapRequestToOdata(odataMap, &mapping)
	tags, count, err := tag.Retrieve(masterDB, odataMap, 250) // Per RRS documentation, size limit of 250
	if err != nil {
		mRetrieveErr.Update(1)
		return errors.Wrap(err, "Error retrieving Tag")
	}

	// If count only is requested
	if count != nil && tags != nil {
		mSuccess.Update(1)
		web.Respond(ctx, writer, count, http.StatusOK)
		return nil
	}

	tagSlice, err := unmarshalTagsInterface(tags)
	if err != nil {
		return err
	}

	if len(tagSlice) > 0 {
		if err := ApplyConfidence(masterDB, tagSlice, url); err != nil {
			return err
		}
	} else {
		tagSlice = []tag.Tag{} // Set empty array
	}

	var results tag.Response
	results.Results = tagSlice

	if count != nil {
		results.Count = count.Count
	}

	// For the upcoming release cursor is not a priority
	/*if cursor != nil && cursor.Cursor != "" {
		results.PagingType = cursor
	}*/

	web.Respond(ctx, writer, results, http.StatusOK)
	mSuccess.Update(1)
	return nil
}

// processPostRequest handles POST requests for sending inventory snapshot to the cloud connector
func processPostRequest(ctx context.Context, schema string, masterDB *sql.DB, request *http.Request, writer http.ResponseWriter, url string) error {

	// Metrics
	metrics.GetOrRegisterGauge("Inventory.processPostRequest.Attempt", nil).Update(1)
	startTime := time.Now()
	defer metrics.GetOrRegisterTimer("Inventory.processPostRequest.Latency", nil).Update(time.Since(startTime))
	mSuccess := metrics.GetOrRegisterGauge("Inventory.PostCurrentInventory.Success", nil)
	mRetrieveErr := metrics.GetOrRegisterGauge("Inventory.PostCurrentInventory.Retrieve-Error", nil)

	var tags []tag.Tag
	var err error
	odataMap := make(map[string][]string)

	// if there is a request body validate the request
	if request.ContentLength > 0 {
		var mapping tag.RequestBody

		validationErrors, err := readAndValidateRequest(request, schema, &mapping)
		if err != nil {
			return err
		}
		if validationErrors != nil {
			web.Respond(ctx, writer, validationErrors, http.StatusBadRequest)
			return nil
		}
		odataMap = mapRequestToOdata(odataMap, &mapping)
	}
	tags, err = tag.RetrieveOdataAll(masterDB, odataMap)
	if err != nil {
		mRetrieveErr.Update(1)
		return err
	}

	if len(tags) > 0 {
		if err := ApplyConfidence(masterDB, tags, url); err != nil {
			return err
		}
		if err := postToCloudInBatches(tags); err != nil {
			return err
		}
	}

	web.Respond(ctx, writer, nil, http.StatusOK)
	mSuccess.Update(1)
	return nil
}

func postToCloudInBatches(tags []tag.Tag) error {
	if config.AppConfig.CloudConnectorUrl != "" {
		triggerCloudConnectorEndpoint := config.AppConfig.CloudConnectorUrl + config.AppConfig.CloudConnectorApiGatewayEndpoint

		batchSize := 500
		range1 := 0
		range2 := batchSize
		lastBatch := false
		var payload event.DataPayload
		for {
			if range2 < len(tags) {
				payload = event.DataPayload{
					TagEvent: tags[range1:range2],
				}
			} else {
				payload = event.DataPayload{
					TagEvent: tags[range1:],
				}
				lastBatch = true
			}
			if err := event.TriggerCloudConnector(payload.ControllerId, payload.SentOn, payload.TotalEventSegments, payload.EventSegmentNumber, payload.TagEvent, triggerCloudConnectorEndpoint); err != nil {
				return errors.Wrap(err, "Error sending tags to cloud connector")
			}
			if lastBatch {
				break
			}
			range1 = range2
			range2 += batchSize
		}
	}
	return nil
}

func unmarshalTagsInterface(tags interface{}) ([]tag.Tag, error) {
	// todo: don't marshal/unmarshal... this is a hack
	tagsBytes, err := json.Marshal(tags)
	if err != nil {
		return nil, errors.Wrap(err, "marshaling []interface{} to []bytes")
	}

	var tagSlice []tag.Tag
	if err := json.Unmarshal(tagsBytes, &tagSlice); err != nil {
		return nil, errors.Wrap(err, "unmarshaling []bytes to []Tags")
	}

	return tagSlice, nil
}

func readAndValidateRequest(request *http.Request, schema string, v interface{}) (interface{}, error) {
	// Reading request
	body := make([]byte, request.ContentLength)
	_, err := io.ReadFull(request.Body, body)
	if err != nil {
		return nil, errors.Wrap(web.ErrValidation, err.Error())
	}

	if err = json.Unmarshal(body, &v); err != nil {
		return nil, errors.Wrap(web.ErrValidation, err.Error())
	}

	// Validate json against schema
	schemaValidatorResult, err := schemas.ValidateSchemaRequest(body, schema)
	if err != nil {
		return nil, err
	}
	if !schemaValidatorResult.Valid() {
		result := schemas.BuildErrorsString(schemaValidatorResult.Errors())
		return result, nil
	}

	return nil, nil
}

// processUpdateRequest handles the request that needs database updating
// nolint :lll
func processUpdateRequest(ctx context.Context, masterDB *sql.DB, writer http.ResponseWriter, epc string,
	facilityId string, object map[string]string) error {

	err := tag.Update(masterDB, epc, facilityId, object)
	if err != nil {
		return errors.Wrap(err, "Error updating Tag")
	}

	web.Respond(ctx, writer, nil, http.StatusOK)

	return nil
}

// nolint :gocyclo
func mapRequestToOdata(odataMap map[string][]string, request *tag.RequestBody) map[string][]string {

	var filterSlice []string
	if request.Cursor != "" {
		filterSlice = append(filterSlice, "_id gt '"+request.Cursor+"'")
	}
	if request.Size > 0 {
		odataMap["$top"] = append(odataMap["$top"], strconv.Itoa(request.Size))
	}
	if request.FacilityID != "" {
		filterSlice = append(filterSlice, "facility_id eq '"+request.FacilityID+"'")
	}
	if request.QualifiedState != "" {
		filterSlice = append(filterSlice, "qualified_state eq '"+request.QualifiedState+"'")
	}
	if request.EpcState != "" {
		filterSlice = append(filterSlice, "epc_state eq '"+request.EpcState+"'")
	}
	if request.Confidence != 0 {
		// nolint :lll
		filterSlice = append(filterSlice, "confidence ge "+strconv.FormatFloat(request.Confidence, 'f', -1, 64))
	}
	if request.ProductID != "" {
		filterSlice = append(filterSlice, "gtin eq '"+request.ProductID+"'")
	}
	if request.StartTime != 0 {
		filterSlice = append(filterSlice, "last_read ge "+strconv.FormatInt(request.StartTime, 10))
	}
	if request.EndTime != 0 {
		filterSlice = append(filterSlice, "last_read le "+strconv.FormatInt(request.EndTime, 10))
	}
	if request.Time != 0 {
		filterSlice = append(filterSlice, "last_read le "+strconv.FormatInt(request.Time, 10))
	}
	if request.Epc != "" {
		epcParts := strings.Split(request.Epc, "*")
		if len(epcParts) == 2 {
			// Note:json schema ensures at most one '*'
			if epcParts[0] != "" {
				filterSlice = append(filterSlice, "startswith(epc, '"+epcParts[0]+"')")
			}
			if epcParts[1] != "" {
				filterSlice = append(filterSlice, "endswith(epc, '"+epcParts[1]+"')")
			}
		} else {
			filterSlice = append(filterSlice, "epc eq '"+request.Epc+"'")
		}
	}

	odataMap["$filter"] = append(odataMap["$filter"], strings.Join(filterSlice, " and "))

	if request.CountOnly {
		odataMap["$inlinecount"] = append(odataMap["$inlinecount"], "allpages")
	}

	return odataMap
}

func getTagForSelectFields(copyOfSelectQuery string, tagSlice *[]tag.Tag) interface{} {
	if len(*tagSlice) < 1 {
		return []tag.Tag{}
	}
	selectFields := strings.Split(copyOfSelectQuery, ",")
	mapSelectFields := make(map[string]bool, len(selectFields))
	for _, field := range selectFields {
		mapSelectFields[strings.TrimSpace(field)] = true
	}
	var filterSlice []map[string]interface{}

	for _, tagSelect := range *tagSlice {
		singleTag := MapOfTags(&tagSelect, mapSelectFields)
		filterSlice = append(filterSlice, singleTag)
	}

	if len(filterSlice) < 1 {
		return []tag.Tag{}
	}
	return filterSlice
}

// MapOfTags creates a map for selected fields in query string
func MapOfTags(tag *tag.Tag, selectFields map[string]bool) map[string]interface{} {
	tagType := reflect.TypeOf(*tag)
	tagValue := reflect.ValueOf(*tag)
	mapTags := make(map[string]interface{})
	for i := 0; i < tagValue.NumField(); i++ {
		key := tagType.Field(i).Tag.Get("json")
		tagKey := strings.Split(key, ",")[0]
		if selectFields[tagKey] {
			mapTags[tagKey] = tagValue.Field(i).Interface()
		}
	}
	return mapTags
}
