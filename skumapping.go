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
	"fmt"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/jsonrpc"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	db "github.impcloud.net/RSP-Inventory-Suite/go-dbWrapper"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/cloudconnector/event"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/routes/handlers"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/tag"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/statemodel"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
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

	mProcessTagLatency := metrics.GetOrRegisterTimer(`Inventory.ProcessTagData-Latency`, nil)
	processTagTimer := time.Now()

	var tagData []tag.Tag
	var tagStateChangeList []tag.TagStateChange

	// todo: is below comment still valid?
	// POC only implementation
	currentTimeMillis := helper.UnixMilliNow()

	numberOfTags := len(invEvent.Params.Data)

	if tagsGauge != nil {
		(*tagsGauge).Add(int64(numberOfTags))
	}
	log.Debugf("Processing %d Tags.\n", numberOfTags)
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

		log.Debug("Previous and Current Tag State:\n")
		log.Debug(tagStateChange)

	}

	log.Debugf("Filtered %d Tags.", tagsFiltered)

	// If at least 1 tag passed the whitelist, then insert
	if len(tagData) > 0 {
		copySession := masterDB.CopySession()
		if err := tag.Replace(copySession, &tagData); err != nil {
			return errors.Wrap(err, "error replacing tags")
		}

		if err := handlers.ApplyConfidence(copySession, &tagData, skuMapping.url); err != nil {
			return err
		}
		copySession.Close()

		handlers.UpdateForCycleCount(tagData)

		if config.AppConfig.CloudConnectorUrl != "" {
			// todo: what else to put in here? seems like an old SAF bus artifact??
			payload := event.DataPayload{
				TagEvent: tagData,
			}
			triggerCloudConnectorEndpoint := config.AppConfig.CloudConnectorUrl + config.AppConfig.CloudConnectorApiGatewayEndpoint

			if err := event.TriggerCloudConnector(invEvent.Params.GatewayId, payload.SentOn, payload.TotalEventSegments, payload.EventSegmentNumber, tagData, triggerCloudConnectorEndpoint); err != nil {
				// Must log here since in a go function, i.e. can't return the error.
				log.WithFields(log.Fields{
					"Method": "processTagData",
					"Action": "Trigger Cloud Connector",
					"Error":  err.Error(),
				}).Error(err)
			}
		}

		if config.AppConfig.RulesUrl != "" {
			go func() {
				if source == "handheld" || config.AppConfig.TriggerRulesOnFixedTags == false {
					// Run only the StateChanged rule since handheld or not triggering on fixed tags
					if err := triggerRules(config.AppConfig.RulesUrl+config.AppConfig.TriggerRulesEndpoint+"?ruletype="+tag.StateChangeEvent, tagStateChangeList); err != nil {
						// Must log here since in a go function, i.e. can't return the error.
						log.WithFields(log.Fields{
							"Method": "processTagData",
							"Action": "Trigger state change rules",
							"Error":  fmt.Sprintf("%+v", err),
						}).Error(err)
					}
				} else {
					// Run all rules
					if err := triggerRules(config.AppConfig.RulesUrl+config.AppConfig.TriggerRulesEndpoint, tagStateChangeList); err != nil {
						// Must log here since in a go function, i.e. can't return the error.
						log.WithFields(log.Fields{
							"Method": "processTagData",
							"Action": "Trigger rules",
							"Error":  fmt.Sprintf("%+v", err),
						}).Error(err)
					}
				}
			}()
		}
	}

	mProcessTagLatency.Update(time.Since(processTagTimer))

	return nil
}
