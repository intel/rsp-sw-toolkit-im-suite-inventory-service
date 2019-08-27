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
	"encoding/json"
	"fmt"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/cloudconnector/event"
	"io"
	"net/http"
	"plugin"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/dailyturn"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/statemodel"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	db "github.impcloud.net/RSP-Inventory-Suite/go-dbWrapper"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/contraepc"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/facility"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/routes/schemas"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/tag"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/web"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/productdata"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
)

// ApplyConfidence calculates the confidence to each tag using the facility coefficients
// this function can be reused by RRS endpoint and RRP endpoints with odata for multiple facilities
func ApplyConfidence(session *db.DB, tags *[]tag.Tag, url string) error {

	if len(*tags) == 0 {
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

	for i := 0; i < len(*tags); i++ {
		// Get coefficients
		facilityID := (*tags)[i].FacilityID
		var confidence float64
		tagFacility, foundFacility := facilities[facilityID]
		lastRead := (*tags)[i].LastRead
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
		gtin := (*tags)[i].ProductID

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

		log.Tracef("DailyInvPerc = %f, probUnreadToRead = %f, probInStore = %f, probExitError = %f", dailyInvPerc, probUnreadToRead, probInStore, probExitError)

		// Load proprietary Intel probabilistic confidence algorithm
		confidencePlugin, err := plugin.Open("/plugin/inventory-probabilistic-algo")
		if err != nil {
			log.Warn("Intel Probabilistic Algorithm plugin not found. Setting Confidence to 0.")
			(*tags)[i].Confidence = 0
			return nil
		}

		calculateConfidence, err := confidencePlugin.Lookup("CalculateConfidence")
		if err != nil {
			log.Errorf("Unable to find calculate confidence function")
		}

		confidence = calculateConfidence.(func(float64, float64, float64, float64, int64, bool) float64)(dailyInvPerc, probUnreadToRead, probInStore, probExitError,
			lastRead, contraepc.IsContraEpc((*tags)[i]))

		(*tags)[i].Confidence = confidence
	}
	return nil
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
func processGetRequest(ctx context.Context, schema string, MasterDB *db.DB, request *http.Request, writer http.ResponseWriter, url string) error {

	// Metrics
	metrics.GetOrRegisterGauge("Inventory.processGetRequest.Attempt", nil).Update(1)

	startTime := time.Now()
	defer metrics.GetOrRegisterTimer("Inventory.processGetRequest.Latency", nil).Update(time.Since(startTime))

	mSuccess := metrics.GetOrRegisterGauge("Inventory.GetTags.Success", nil)
	mRetrieveErr := metrics.GetOrRegisterGauge("Inventory.GetTags.Retrieve-Error", nil)

	copySession := MasterDB.CopySession()
	defer copySession.Close()

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
	tags, count, cursor, err := tag.Retrieve(copySession, odataMap, 250) // Per RRS documentation, size limit of 250
	if err != nil {
		mRetrieveErr.Update(1)
		return errors.Wrap(err, "Error retrieving Tag")
	}

	//If count only is requested
	if count != nil && tags != nil {
		mSuccess.Update(1)
		web.Respond(ctx, writer, count, http.StatusOK)
		return nil
	}

	tagSlice, err := unmarshallTagsInterface(tags)
	if err != nil {
		return err
	}

	if len(tagSlice) > 0 {
		if err := ApplyConfidence(copySession, &tagSlice, url); err != nil {
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

	if cursor != nil && cursor.Cursor != "" {
		results.PagingType = cursor
	}

	web.Respond(ctx, writer, results, http.StatusOK)
	mSuccess.Update(1)
	return nil
}

// processPostRequest handles POST requests for sending inventory snapshot to the cloud connector
func processPostRequest(ctx context.Context, schema string, MasterDB *db.DB, request *http.Request, writer http.ResponseWriter, url string) error {

	// Metrics
	metrics.GetOrRegisterGauge("Inventory.processPostRequest.Attempt", nil).Update(1)
	startTime := time.Now()
	defer metrics.GetOrRegisterTimer("Inventory.processPostRequest.Latency", nil).Update(time.Since(startTime))
	mSuccess := metrics.GetOrRegisterGauge("Inventory.PostCurrentInventory.Success", nil)
	mRetrieveErr := metrics.GetOrRegisterGauge("Inventory.PostCurrentInventory.Retrieve-Error", nil)

	copySession := MasterDB.CopySession()
	defer copySession.Close()

	var tags []tag.Tag
	var err error

	// if there is a request body
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

		odataMap := make(map[string][]string)
		odataMap = mapRequestToOdata(odataMap, &mapping)

		tags, err = tag.RetrieveOdataAll(copySession, odataMap)
		if err != nil {
			mRetrieveErr.Update(1)
			return err
		}
	} else {
		tags, err = tag.RetrieveAll(copySession)
		if err != nil {
			mRetrieveErr.Update(1)
			return err
		}
	}

	if len(tags) > 0 {
		if err := ApplyConfidence(copySession, &tags, url); err != nil {
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
		for {
			if range2 < len(tags) {
				payload := event.DataPayload{
					TagEvent: tags[range1:range2],
				}
				if err := event.TriggerCloudConnector(payload.ControllerId, payload.SentOn, payload.TotalEventSegments, payload.EventSegmentNumber, payload.TagEvent, triggerCloudConnectorEndpoint); err != nil {
					return errors.Wrap(err, "Error sending tags to cloud connector")
				}
			} else {
				payload := event.DataPayload{
					TagEvent: tags[range1:],
				}
				if err := event.TriggerCloudConnector(payload.ControllerId, payload.SentOn, payload.TotalEventSegments, payload.EventSegmentNumber, payload.TagEvent, triggerCloudConnectorEndpoint); err != nil {
					return errors.Wrap(err, "Error sending tags to cloud connector")
				}
				lastBatch = true
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

func unmarshallTagsInterface(tags interface{}) ([]tag.Tag, error) {

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

	// Unmarshal and validate request only if its body is not empty
	if err = json.Unmarshal(body, &v); err != nil {
		log.Info("Unmarshalling failed")
		return nil, errors.Wrap(web.ErrValidation, err.Error())
	}

	// Validate json against schema
	schemaValidatorResult, err := schemas.ValidateSchemaRequest(body, schema)
	if err != nil {
		log.Info("ValidateSchemaRequest failed")
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
func processUpdateRequest(ctx context.Context, MasterDB *db.DB, writer http.ResponseWriter, selectorMap map[string]interface{},
	objectMap map[string]interface{}) error {

	copySession := MasterDB.CopySession()
	defer copySession.Close()

	err := tag.Update(copySession, selectorMap, objectMap)
	if err != nil {
		return errors.Wrap(err, "Error updating Tag")
	}

	web.Respond(ctx, writer, nil, http.StatusOK)

	return nil
}

//nolint :gocyclo
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
		//nolint :lll
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

//nolint:gocyclo
// cyclymatic complexity of 13, no need for further refactor.
func generateContraEPC(data []contraepc.CreateContraEpcItem, copySession *db.DB) ([]tag.Tag, error) {

	mInvalidInputErr := metrics.GetOrRegisterGauge("Inventory.generateContraEPC.InvalidInput-Error", nil)
	mFindByEpcErr := metrics.GetOrRegisterGauge("Inventory.generateContraEPC.FindByEpc-Error", nil)
	mGenerateContraEpcErr := metrics.GetOrRegisterGauge("Inventory.generateContraEPC.GenerateContraEpc-Error", nil)
	mGenerateContraEpcTriesErr := metrics.GetOrRegisterGauge("Inventory.generateContraEPC.GenerateContraEpcTries-Error", nil)
	mGenerateContraEpcLatency := metrics.GetOrRegisterTimer("Inventory.CreateContraEPC.GenerateContraEpc-Latency", nil)

	tagData := make([]tag.Tag, len(data))
	// A set of epcs we have already generated (to track uniqueness)
	uniqueEpcs := make(map[string]bool)
	for i, item := range data {
		if item.Epc != "" {
			if uniqueEpcs[item.Epc] {
				mInvalidInputErr.Update(1)
				return nil, errors.Wrap(web.ErrInvalidInput, "found multiple request items with the same epc")
			}
			// Make sure the epc is not already in the database
			foundTag, err := tag.FindByEpc(copySession, item.Epc)
			if err != nil {
				mFindByEpcErr.Update(1)
				return nil, errors.Wrap(err, "error checking database for existing tag")
			}
			if foundTag.IsTagReadByRspController() {
				mInvalidInputErr.Update(1)
				return nil, errors.Wrap(web.ErrInvalidInput, "tag with that epc already exists in the database")
			}

			// If this epc does not already exist, convert it to a Tag to insert into the db
			tagData[i] = item.AsNewTag()
			uniqueEpcs[item.Epc] = true
		} else if item.Gtin != "" {
			// How many times to attempt to generate a unique epc
			tries := contraepc.MaxTries
			generateContreEpcTimer := time.Now()
			for tries > 0 {
				// Generate a contra-epc based on the gtin provided
				epc, err := contraepc.GenerateContraEPC(item.Gtin)
				if err != nil {
					mGenerateContraEpcErr.Update(1)
					return nil, errors.Wrapf(err, "error generating contra-epc for gtin %s", item.Gtin)
				}

				// If the epc we generated is already pending use, try again
				if uniqueEpcs[epc] {
					tries--
					continue
				}

				// If the epc we generated is already in the database, try again
				foundTag, err := tag.FindByEpc(copySession, item.Epc)
				if err != nil || foundTag.IsTagReadByRspController() {
					tries--
					continue
				}

				// Convert it to a Tag to insert into the db
				item.Epc = epc
				tagData[i] = item.AsNewTag()
				uniqueEpcs[item.Epc] = true
				break
			}
			mGenerateContraEpcLatency.Update(time.Since(generateContreEpcTimer))
			if tries == 0 {
				// internal server error
				mGenerateContraEpcTriesErr.Update(1)
				return nil, fmt.Errorf("unable to generate a unique contra-epc after %d tries for gtin %s",
					contraepc.MaxTries, item.Gtin)
			}
		} else {
			mInvalidInputErr.Update(1)
			return nil, errors.Wrap(web.ErrInvalidInput, "missing epc or gtin")
		}
	}

	return tagData, nil

}
