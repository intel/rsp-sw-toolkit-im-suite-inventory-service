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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/edgexfoundry/go-mod-core-contracts/clients"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/coredata"
	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	db "github.impcloud.net/RSP-Inventory-Suite/go-dbWrapper"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/cloudconnector"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/cloudconnector/event"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/routes/handlers"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/rules"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/tag"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/jsonrpc"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/statemodel"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
	"reflect"
	"time"
	"unsafe"
)

// SkuMapping struct for the sku-mapping service
type SkuMapping struct {
	url string
}

// NewSkuMapping initialize new SkuMapping
func NewSkuMapping(url string) SkuMapping {
	return SkuMapping{
		url: url,
	}
}

// processTagData inserts data from context sensing broker into database
//nolint :gocyclo
func (skuMapping SkuMapping) processTagData(invEvent *jsonrpc.InventoryEvent, masterDB *db.DB, source string, tagsGauge *metrics.GaugeCollection) error {

	numberOfTags := len(invEvent.Params.Data)
	if numberOfTags == 0 {
		return nil
	}

	mProcessTagLatency := metrics.GetOrRegisterTimer(`Inventory.ProcessTagData-Latency`, nil)
	processTagTimer := time.Now()

	var tagData []tag.Tag
	var tagStateChangeList []tag.TagStateChange

	// todo: is below comment still valid?
	// POC only implementation
	currentTimeMillis := helper.UnixMilliNow()

	if tagsGauge != nil {
		(*tagsGauge).Add(int64(numberOfTags))
	}
	log.Debugf("Processing %d Tag Events", numberOfTags)
	tagsFiltered := 0
	for _, tempTag := range invEvent.Params.Data {
		if len(config.AppConfig.EpcFilters) > 0 {
			// ignore tags that don't match our filters
			if !statemodel.IsTagWhitelisted(tempTag.EpcCode, config.AppConfig.EpcFilters) {
				continue
			}
		}

		// todo: is below comment still valid?
		// POC only implementation
		markDepartedIfUnseen(&tempTag, config.AppConfig.AgeOuts, currentTimeMillis)

		// Add source & event
		if source == "handheld" {
			tempTag.EventType = statemodel.ArrivalEvent
		}

		// Note: If bottlenecks may need to redesign to eliminate large number
		// of queries to DB currently this will make a call to the DB PER tag
		tagFromDB, err := tag.FindByEpc(masterDB, tempTag.EpcCode)

		if err != nil {
			return errors.Wrap(err, "Error retrieving tag from database")
		}

		updatedTag := statemodel.UpdateTag(tagFromDB, tempTag, source)

		tagData = append(tagData, updatedTag)

		var tagStateChange tag.TagStateChange
		tagStateChange.PreviousState = tagFromDB
		tagStateChange.CurrentState = updatedTag

		if tagStateChange.PreviousState.IsEqual(tag.Tag{}) != true &&
			tagStateChange.CurrentState.IsEqual(tag.Tag{}) != true {
			tagStateChangeList = append(tagStateChangeList, tagStateChange)
		}

		log.Trace("Previous and Current Tag State:\n")
		log.Trace(tagStateChange)
	}

	log.Debugf("Filtered %d Tags.", tagsFiltered)

	// If at least 1 tag passed the whitelist, then insert
	if len(tagData) > 0 {
		copySession := masterDB.CopySession()
		defer copySession.Close()

		if err := tag.Replace(copySession, &tagData); err != nil {
			return errors.Wrap(err, "error replacing tags")
		}

		if err := handlers.ApplyConfidence(copySession, &tagData, skuMapping.url); err != nil {
			return err
		}

		handlers.UpdateForCycleCount(tagData)

		if config.AppConfig.CloudConnectorUrl != "" {
			go func() {
				if err := cloudconnector.SendEvent(invEvent, tagData); err != nil {
					log.WithFields(log.Fields{
						"Method": "processTagData",
						"Action": "Trigger Cloud Connector",
						"Error":  err.Error(),
					}).Error(err)
				}
			}()
		}

		if config.AppConfig.RulesUrl != "" {
			go func() {
				if err := rules.ApplyRules(source, tagStateChangeList); err != nil {
					log.WithFields(log.Fields{
						"Method": "processTagData",
						"Action": "Apply Rules",
						"Error":  fmt.Sprintf("%+v", err),
					}).Error(err)
				}
			}()
		}

		go func() {
			//tagEvents := make([]tag.Tag, len(tagStateChangeList))
			//for _, tagChange := range tagStateChangeList {
			//	tagEvents = append(tagEvents, tagChange.CurrentState)
			//}

			tagEvents := tagData

			if len(tagEvents) > 0 {
				log.Debugf("%+v", tagEvents)

				correlation := uuid.New().String()
				ctx := context.WithValue(context.Background(), clients.CorrelationHeader, correlation)
				ctx = context.WithValue(ctx, clients.ContentType, clients.ContentTypeJSON)

				payload, err := json.Marshal(event.DataPayload{
					SentOn:             currentTimeMillis,
					ControllerId:       invEvent.Params.ControllerId,
					EventSegmentNumber: 1,
					TotalEventSegments: 1,
					TagEvent:           tagEvents,
				})
				if err != nil {
					log.WithFields(log.Fields{
						"Method": "processTagData",
						"Action": "Publish to Core Data",
						"Error":  fmt.Sprintf("%+v", err),
					}).Error(err)
					return
				}

				evt := models.Event{
					Device: invEvent.Params.ControllerId,
					Origin: currentTimeMillis,
					Readings: []models.Reading{
						{
							Device: invEvent.Params.ControllerId,
							Name:   inventoryServiceEvent,
							Value:  string(payload),
							Origin: currentTimeMillis,
						},
					},
				}

				eventClientValue := reflect.ValueOf(edgexSdk).Elem().FieldByName("edgexClients").FieldByName("EventClient")
				eventClientValue = reflect.NewAt(eventClientValue.Type(), unsafe.Pointer(eventClientValue.UnsafeAddr())).Elem()
				eventClient, ok := eventClientValue.Interface().(coredata.EventClient)
				if !ok {
					log.Error("Unable to cast to EventClient")
					return
				}
				_, _ = eventClient.Add(&evt, ctx)
			}
		}()
	}

	mProcessTagLatency.Update(time.Since(processTagTimer))

	return nil
}
