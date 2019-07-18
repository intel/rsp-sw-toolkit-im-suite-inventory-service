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
	"encoding/json"
	"fmt"
	"plugin"
	"time"

	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	metrics "github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
)

func errorHandler(message string, err error, errorGauge *metrics.Gauge) {
	if err != nil {
		(*errorGauge).Update(1)
		log.WithFields(log.Fields{
			"Method": "main",
			"Error":  fmt.Sprintf("%+v", err),
		}).Error(message)
	}
}

func fatalErrorHandler(message string, err error, errorGauge *metrics.Gauge) {
	if err != nil {
		(*errorGauge).Update(1)
		log.WithFields(log.Fields{
			"Method": "main",
			"Error":  fmt.Sprintf("%+v", err),
		}).Fatal(message)
	}
}

func handleMessage(dataType string, data *map[string]interface{}, errGauge *metrics.Gauge, handler func([]byte) error) {
	if data == nil {
		errorHandler(fmt.Sprintf("unable to marshal %s data", dataType),
			errors.New("ItemData was nil"), errGauge)
		return
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		errorHandler(fmt.Sprintf("unable to marshal %s data", dataType),
			err, errGauge)
		return
	}

	if err := handler(jsonBytes); err != nil {
		errorHandler(fmt.Sprintf("error processing %s data", dataType),
			err, errGauge)
	}
}

func verifyProbabilisticPlugin() {
	retry := 1
	pluginFound := false

	for retry < 10 {

		log.Infof("Loading proprietary Intel Probabilistic Algorithm plugin (Retry %d)", retry)
		probPlugin, err := plugin.Open("/plugin/inventory-probabilistic-algo")
		if err == nil {
			pluginFound = true
			checkIA, err := probPlugin.Lookup("CheckIA")
			if err != nil {
				log.Errorf("Unable to find checkIA function in probabilistic algorithm plugin")
				break
			}

			if err := checkIA.(func() error)(); err != nil {
				log.Warnf("Unable to verify Intel Architecture, Confidence value will be set to 0. Error: %s", err.Error())
				break
			}

			log.Info("Intel Probabilistic Algorithm plugin loaded.")
			break
		}
		log.Warn(err)
		time.Sleep(1 * time.Second)
		retry++
	}

	if !pluginFound {
		log.Warn("Unable to verify Intel Architecture, Confidence value will be set to 0")
	}
}

func parseReadingValue(read *models.Reading) (*reading, error) {

	readingObj := reading{}

	if err := json.Unmarshal([]byte(read.Value), &readingObj); err != nil {
		return nil, err
	}

	return &readingObj, nil

}
