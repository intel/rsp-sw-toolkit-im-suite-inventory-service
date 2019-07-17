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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	golog "log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/globalsign/mgo"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"github.com/edgexfoundry/app-functions-sdk-go/appcontext"
	"github.com/edgexfoundry/app-functions-sdk-go/appsdk"
	db "github.impcloud.net/RSP-Inventory-Suite/go-dbWrapper"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/alert"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/dailyturn"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/facility"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/routes"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/tag"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/statemodel"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
	reporter "github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics-influxdb"
)

const (
	jsonApplication = "application/json;charset=utf-8"
	serviceKey      = "inventory-service"
)

const (
	asnData     = "ASN_data"
	gwEvent     = "gw_events"
	gwAlert     = "gw_alert"
	gwHeartbeat = "gw_heartbeat"
)

type myDB struct {
	masterDB *db.DB
}

func main() {

	mDBIndexesError := metrics.GetOrRegisterGauge("Inventory.Main.DBIndexesError", nil)
	mConfigurationError := metrics.GetOrRegisterGauge("Inventory.Main.ConfigurationError", nil)
	mDatabaseRegisterError := metrics.GetOrRegisterGauge("Inventory.Main.DatabaseRegisterError", nil)

	// Ensure simple text format
	log.SetFormatter(&log.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})

	// Load config variables
	err := config.InitConfig()
	fatalErrorHandler("unable to load configuration variables", err, &mConfigurationError)

	// Start healthCheck
	healthCheck(config.AppConfig.Port)

	// Initialize metrics reporting
	initMetrics()

	setLoggingLevel(config.AppConfig.LoggingLevel)

	log.WithFields(log.Fields{
		"Method": "main",
		"Action": "Start",
	}).Info("Starting inventory service...")

	dbName := config.AppConfig.DatabaseName
	dbHost := config.AppConfig.ConnectionString + "/" + dbName

	// Connect to mongodb
	log.WithFields(log.Fields{
		"Method": "main",
		"Action": "Start",
		"Host":   config.AppConfig.DatabaseName,
	}).Info("Registering a new master db...")

	masterDB, err := db.NewSession(dbHost, 5*time.Second)
	fatalErrorHandler("Unable to register a new master db.", err, &mDatabaseRegisterError)

	// Close master db
	defer masterDB.Close()

	// Prepares database indexes
	prepDBErr := prepareDB(masterDB)
	errorHandler("error creating indexes", prepDBErr, &mDBIndexesError)

	// Verify IA when using Probabilistic Algorithm plugin
	if config.AppConfig.ProbabilisticAlgorithmPlugin {
		verifyProbabilisticPlugin()
	}

	// Connect to EdgeX zeroMQ bus
	receiveZMQEvents(masterDB)

	// Initiate webserver and routes
	startWebServer(masterDB, config.AppConfig.Port, config.AppConfig.ResponseLimit, config.AppConfig.ServiceName)

	log.WithField("Method", "main").Info("Completed.")
}

func startWebServer(masterDB *db.DB, port string, responseLimit int, serviceName string) {

	// Start Webserver and pass additional data
	router := routes.NewRouter(masterDB, responseLimit)

	// Create a new server and set timeout values.
	server := http.Server{
		Addr:           ":" + port,
		Handler:        router,
		ReadTimeout:    900 * time.Second,
		WriteTimeout:   900 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// We want to report the listener is closed.
	var wg sync.WaitGroup
	wg.Add(1)

	// Start the listener.
	go func() {
		log.Infof("%s running!", serviceName)
		log.Infof("Listener closed : %v", server.ListenAndServe())
		wg.Done()
	}()

	// Listen for an interrupt signal from the OS.
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt)

	// Wait for a signal to shutdown.
	<-osSignals

	// Create a context to attempt a graceful 5 second shutdown.
	const timeout = 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Attempt the graceful shutdown by closing the listener and
	// completing all inflight requests.
	if err := server.Shutdown(ctx); err != nil {

		log.WithFields(log.Fields{
			"Method":  "main",
			"Action":  "shutdown",
			"Timeout": timeout,
			"Message": err.Error(),
		}).Error("Graceful shutdown did not complete")

		// Looks like we timedout on the graceful shutdown. Kill it hard.
		if err := server.Close(); err != nil {
			log.WithFields(log.Fields{
				"Method":  "main",
				"Action":  "shutdown",
				"Message": err.Error(),
			}).Error("Error killing server")
		}
	}

	// Wait for the listener to report it is closed.
	wg.Wait()
}

func processHeartBeat(reading *models.Reading, masterDB *db.DB) error {

	log.Debugf("Received Heartbeat:\n%s", reading.Value)

	var data map[string]interface{}
	decoder := json.NewDecoder(strings.NewReader(reading.Value))
	decoder.UseNumber()

	if err := decoder.Decode(&data); err != nil {
		return errors.Wrap(err, "Error decoding HeartBeat")
	}

	facilities, ok := data["facilities"].([]interface{})
	if !ok {
		return errors.New("Not able to cast heartbeat data")
	}
	//noinspection GoPreferNilSlice
	facilityData := []facility.Facility{}

	for _, f := range facilities {
		name := f.(string)
		facilityData = append(facilityData, facility.Facility{Name: name})
	}

	copySession := masterDB.CopySession()

	// Default coefficients
	var coefficients facility.Coefficients
	coefficients.DailyInventoryPercentage = config.AppConfig.DailyInventoryPercentage
	coefficients.ProbUnreadToRead = config.AppConfig.ProbUnreadToRead
	coefficients.ProbInStoreRead = config.AppConfig.ProbInStoreRead
	coefficients.ProbExitError = config.AppConfig.ProbExitError

	// Insert facilities to database and set default coefficients if new facility is inserted
	if err := facility.Insert(copySession, &facilityData, coefficients); err != nil {
		copySession.Close()
		return errors.Wrap(err, "Error replacing facilities")
	}
	copySession.Close()

	return nil
}

// processShippingNotice processes the list of epcs (shipping notice).  If the epc does not exist in the DB
// an entry is created with a default facility config.AppConfig.AdvancedShippingNoticeFacilityID,
// a ttl of now, and epc context of the designated value to identify it as a shipping notice
// config.AppConfig.AdvancedShippingNotice.  If the epc does exist, then only epc context value is updated
// with config.AppConfig.AdvancedShippingNotice
func processShippingNotice(reading *models.Reading, masterDB *db.DB, tagsGauge *metrics.GaugeCollection) error {

	log.Debugf("Received data:\n%s", reading.Value)

	var incomingDataSlice []tag.AdvanceShippingNotice
	decoder := json.NewDecoder(strings.NewReader(reading.Value))
	if err := decoder.Decode(&incomingDataSlice); err != nil {
		return errors.Wrap(err, "unable to Decode data")
	}

	copySession := masterDB.CopySession()
	// do this before inserting the data into the database
	dailyturn.ProcessIncomingASNList(copySession, incomingDataSlice)
	copySession.Close()

	var tagData []tag.Tag
	ttlTime := time.Now()

	for _, asn := range incomingDataSlice {
		if asn.ID == "" || asn.EventTime == "" || asn.SiteID == "" || asn.Items == nil {
			return errors.New("ASN is missing data")
		}
		if tagsGauge != nil {
			(*tagsGauge).Add(int64(len(asn.Items)))
		}

		for _, asnItem := range asn.Items {
			for _, asnEpc := range asnItem.EPCs {
				// create a temporary tag so we can check if it's whitelisted
				tempTag := tag.Tag{}
				tempTag.Epc = asnEpc
				_, tempTag.URI = tag.DecodeTagData(asnEpc)

				if len(config.AppConfig.EpcFilters) > 0 {
					// ignore tags that don't match our filters
					if !statemodel.IsTagWhitelisted(tempTag.Epc, config.AppConfig.EpcFilters) {
						continue
					}
				}

				// marshal the ASNContext
				asnContextBytes, err := json.Marshal(tag.ASNContext{
					ASNID:     asn.ID,
					EventTime: asn.EventTime,
					SiteID:    asn.SiteID,
					ItemGTIN:  asnItem.ItemGTIN,
					ItemID:    asnItem.ItemID,
				})
				if err != nil {
					return errors.Wrap(err, "Unable to marshal ASNContext")
				}

				// If the tag exists, update it with the new EPCContext.
				// If it is new, insert it with default TTL/FacilityID
				// Note: If bottlenecks may need to redesign to eliminate large number
				// of queries to DB currently this will make a call to the DB PER tag
				tagFromDB, err := tag.FindByEpc(masterDB, tempTag.Epc)
				if err != nil {
					if dbErr := errors.Wrap(err, "Error retrieving tag from database"); dbErr != nil {
						log.Debug(dbErr)
					}
				} else {
					if tagFromDB.IsEmpty() {
						// Tag is not in database, add with defaults
						tempTag.TTL = ttlTime
						tempTag.FacilityID = config.AppConfig.AdvancedShippingNoticeFacilityID
						tempTag.EpcContext = string(asnContextBytes)
						tagData = append(tagData, tempTag)
					} else {
						// Found tag, only update the epc context
						tagFromDB.EpcContext = string(asnContextBytes)
						tagData = append(tagData, tagFromDB)
					}
				}
			}
		}
		if len(tagData) > 0 {
			copySession := masterDB.CopySession()
			if err := tag.Replace(copySession, &tagData); err != nil {
				return errors.Wrap(err, "error replacing tags")
			}
			copySession.Close()
		}
	}

	return nil
}

// PrepareDB prepares the database with indexes
func prepareDB(dbs *db.DB) error {

	copySession := dbs.CopySession()
	defer copySession.Close()

	purgingDays := config.AppConfig.PurgingDays
	// Convert days into seconds
	purgingSeconds := purgingDays * 24 * 60 * 60

	indexes := make(map[string][]mgo.Index)

	// tags purging and query indices
	indexes["tags"] = []mgo.Index{
		{
			Key:        []string{"uri"},
			Unique:     true,
			DropDups:   false,
			Background: false,
		},
		{
			Key:        []string{"epc"},
			Unique:     true,
			DropDups:   false,
			Background: false,
		},
		{
			Key:         []string{"ttl"},
			Unique:      false,
			DropDups:    false,
			Background:  false,
			ExpireAfter: time.Duration(purgingSeconds) * time.Second,
		},
		{
			Key:        []string{"productId"},
			Unique:     false,
			DropDups:   false,
			Background: false,
		},
		{
			Key:        []string{"event"},
			Unique:     false,
			DropDups:   false,
			Background: false,
		},
		{
			Key:        []string{"filter_value"},
			Unique:     false,
			DropDups:   false,
			Background: false,
		},
	}
	// handheldevents purging indices
	indexes["handheldevents"] = []mgo.Index{
		{
			Key:        []string{"timestamp"},
			Unique:     false,
			DropDups:   false,
			Background: false,
		},
		{
			Key:         []string{"ttl"},
			Unique:      false,
			DropDups:    false,
			Background:  false,
			ExpireAfter: time.Duration(purgingSeconds) * time.Second,
		},
	}

	for collectionName, indexes := range indexes {

		for _, index := range indexes {
			execFuncAddIndex := func(collection *mgo.Collection) error {
				log.Infof("Adding Index %s to collection %s.", index.Key[0], collection.Name)
				return collection.EnsureIndex(index)
			}

			execFuncDropIndex := func(collection *mgo.Collection) error {
				log.Infof("Dropping Index %s from collection %s in order to recreate it.", index.Key[0], collection.Name)
				return collection.DropIndex(index.Key[0])
			}

			if err := copySession.Execute(collectionName, execFuncAddIndex); err != nil {
				// Couldn't add the index so drop it and try to add it again, if that doesn't work exit.
				log.Errorf("Unable to add Index %v to collection %s %s", index, collectionName, err.Error())

				// try to drop the index
				if err := copySession.Execute(collectionName, execFuncDropIndex); err != nil {
					log.Errorf("Unable to drop Index %v to collection %s %s", index, collectionName, err.Error())
				}

				// try to add the index after it's been dropped
				if err := copySession.Execute(collectionName, execFuncAddIndex); err != nil {
					return errors.Wrapf(err, "Unable to add Index %v to collection %s", index, collectionName)
				}
			}
		}
	}
	log.WithFields(log.Fields{
		"Method": "config.PrepareDB",
		"Action": "Start",
	}).Info("Prepared database indexes...")

	return nil
}

func callDeleteTagCollection(masterDB *db.DB) error {
	log.Debug("received request to delete tag db collection...")
	return tag.DeleteTagCollection(masterDB)
}

func triggerRules(triggerRulesEndpoint string, data interface{}) error {
	timeout := time.Duration(config.AppConfig.EndpointConnectionTimedOutSeconds) * time.Second
	client := &http.Client{
		Timeout: timeout,
	}

	mData, err := json.Marshal(data)
	if err != nil {
		return errors.Wrapf(err, "problem marshalling the data")
	}

	// Make the POST to authenticate
	request, err := http.NewRequest("POST", triggerRulesEndpoint, bytes.NewBuffer(mData))
	if err != nil {
		return errors.Wrapf(err, "unable to create http.NewRquest")
	}
	request.Header.Set("content-type", jsonApplication)

	response, err := client.Do(request)
	if err != nil {
		return errors.Wrapf(err, "unable trigger rules: %s", triggerRulesEndpoint)
	}
	defer func() {
		if respErr := response.Body.Close(); respErr != nil {
			log.WithFields(log.Fields{
				"Method": "triggerRules",
				"Action": "response.Body.Close()",
			}).Warning("Failed to close response.")
		}
	}()

	if response.StatusCode != http.StatusOK {
		responseData, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return errors.Wrapf(err, "unable to ReadALL response.Body")
		}
		return errors.Wrapf(errors.New("execution error"), "StatusCode %d , Response %s",
			response.StatusCode, string(responseData))
	}
	return nil
}

// POC only implementation
func markDepartedIfUnseen(tag *tag.TagEvent, ageOuts map[string]int, currentTimeMillis int64) {
	if tag.EventType == "cycle_count" {
		if minutes, ok := ageOuts[tag.FacilityID]; ok {
			if tag.Timestamp+int64(minutes*60*1000) <= currentTimeMillis {
				tag.EventType = "departed"
			}
		}
	}
}

func initMetrics() {
	// setup metrics reporting
	if config.AppConfig.TelemetryEndpoint != "" {
		go reporter.InfluxDBWithTags(
			metrics.DefaultRegistry,
			time.Second*10, //cfg.ReportingInterval,
			config.AppConfig.TelemetryEndpoint,
			config.AppConfig.TelemetryDataStoreName,
			"",
			"",
			nil,
		)
	}
}

func receiveZMQEvents(masterDB *db.DB) {

	db := myDB{masterDB: masterDB}

	go func() {

		//Initialized EdgeX apps functionSDK
		edgexSdk := &appsdk.AppFunctionsSDK{ServiceKey: serviceKey}
		if err := edgexSdk.Initialize(); err != nil {
			edgexSdk.LoggingClient.Error(fmt.Sprintf("SDK initialization failed: %v\n", err))
			os.Exit(-1)
		}

		// Filter data by value descriptors
		valueDescriptors := []string{"ASN_data", "gw_event", "gw_alert", "gw_heartbeat"}

		edgexSdk.SetFunctionsPipeline(
			edgexSdk.ValueDescriptorFilter(valueDescriptors),
			db.processEvents,
		)

		err := edgexSdk.MakeItRun()
		if err != nil {
			edgexSdk.LoggingClient.Error("MakeItRun returned error: ", err.Error())
			os.Exit(-1)
		}

	}()
}

func (db myDB) processEvents(edgexcontext *appcontext.Context, params ...interface{}) (bool, interface{}) {
	if len(params) < 1 {
		return false, nil
	}

	event := params[0].(models.Event)
	if len(event.Readings) < 1 {
		return false, nil
	}

	mRRSHeartbeatReceived := metrics.GetOrRegisterGauge("Inventory.receiveZMQEvents.RRSHeartbeatReceived", nil)
	mRRSHeartbeatProcessingError := metrics.GetOrRegisterGauge("Inventory.receiveZMQEvents.RRSHeartbeatError", nil)
	mRRSEventsProcessingError := metrics.GetOrRegisterGauge("Inventory.receiveZMQEvents.RRSEventsError", nil)
	mRRSEventsTags := metrics.GetOrRegisterGaugeCollection("Inventory.receiveZMQEvents.RRSTags", nil)
	mRRSAlertError := metrics.GetOrRegisterGauge("Inventory.receiveZMQEvents.RRSAlertError", nil)
	mRRSResetEventReceived := metrics.GetOrRegisterGaugeCollection("Inventory.receiveZMQEvents.RRSResetEventReceived", nil)
	mRRSASNEpcs := metrics.GetOrRegisterGaugeCollection("Inventory.processShippingNotice.RRSASNEpcs", nil)

	skuMapping := NewSkuMapping(config.AppConfig.MappingSkuUrl)

	for _, reading := range event.Readings {
		switch reading.Name {

		case asnData:
			logrus.Debugf(fmt.Sprintf("ASN data received: %#v", event))

			// todo: why is it decoding base64?????

			//data, err := base64.StdEncoding.DecodeString(reading.Value)
			//if err != nil {
			//	log.WithFields(log.Fields{
			//		"Method": "receiveZMQEvents",
			//		"Action": "ASN data ingestion",
			//		"Error":  err.Error(),
			//	}).Error("error decoding base64 value")
			//	return false, err
			//}
			// todo: &reading appropriate here?
			if err := processShippingNotice(&reading, db.masterDB, &mRRSASNEpcs); err != nil {
				log.WithFields(log.Fields{
					"Method": "processShippingNotice",
					"Action": "ASN data ingestion",
					"Error":  err.Error(),
				}).Error("error processing ASN data")
				return false, err
			}
			mRRSASNEpcs.Add(1)

			break

		case gwHeartbeat:
			mRRSHeartbeatReceived.Update(1)
			if err := handleMessage("heartbeat", &reading, &mRRSHeartbeatProcessingError,
				func(reading *models.Reading) error {
					return processHeartBeat(reading, db.masterDB)
				}); err != nil {
				return false, err
			}
			break

		case gwEvent:
			go func(reading *models.Reading) {
				_ = handleMessage("fixed", reading, &mRRSEventsProcessingError,
					func(reading *models.Reading) error {
						return skuMapping.processTagData(reading, db.masterDB, "fixed", &mRRSEventsTags)
					})
			}(&reading)
			break

		case gwAlert:
			if err := handleMessage("RRS Alert data", &reading, &mRRSAlertError,
				func(reading *models.Reading) error {
					rrsAlert, err := alert.ProcessAlert(reading)
					if err != nil {
						return err
					}

					if rrsAlert.IsInventoryUnloadAlert() {
						mRRSResetEventReceived.Add(1)
						go func() {
							err := callDeleteTagCollection(db.masterDB)
							errorHandler("error calling delete tag collection",
								err, &mRRSEventsProcessingError)

							if err == nil {
								alertMessage := new(alert.MessagePayload)
								if sendErr := alertMessage.SendDeleteTagCompletionAlertMessage(); sendErr != nil {
									errorHandler("error sending alert message for delete tag collection", sendErr, &mRRSEventsProcessingError)
								}
							}
						}()
					}

					return nil
				}); err != nil {
				return false, err
			}
			break
		}
	}

	return false, nil
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
	default:
		log.SetLevel(log.InfoLevel)
	}

	// Not using filtered func (Info, etc ) so that message is always logged
	golog.Printf("Logging level set to %s\n", loggingLevel)
}
