/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"fmt"
	golog "log"
	"plugin"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
)

func errorHandler(message string, err error, errorGauge *metrics.Gauge) {
	if err != nil {
		if errorGauge != nil {
			(*errorGauge).Update(1)
		}
		log.WithFields(log.Fields{
			"Method": "main",
			"Error":  fmt.Sprintf("%+v", err),
		}).Error(message)
	}
}

func fatalErrorHandler(message string, err error, errorGauge *metrics.Gauge) {
	if err != nil {
		if errorGauge != nil {
			(*errorGauge).Update(1)
		}
		log.WithFields(log.Fields{
			"Method": "main",
			"Error":  fmt.Sprintf("%+v", err),
		}).Fatal(message)
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

func setLoggingLevel(loggingLevel string) {
	switch strings.ToLower(loggingLevel) {
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "trace":
		log.SetLevel(log.TraceLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}

	// Not using filtered func (Info, etc ) so that message is always logged
	golog.Printf("Logging level set to %s\n", loggingLevel)
}
