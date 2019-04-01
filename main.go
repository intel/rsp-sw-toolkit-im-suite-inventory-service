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
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/globalsign/mgo"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	zmq "github.com/pebbe/zmq4"
	db "github.impcloud.net/RSP-Inventory-Suite/go-dbWrapper"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/alert"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/dailyturn"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/facility"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/handheldevent"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/routes"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/tag"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/statemodel"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/saf/core"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/saf/core/sensing"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
	reporter "github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics-influxdb"
)

const (
	eventsUrn         = "urn:x-intel:context:retailsensingplatform:events"
	heartbeatUrn      = "urn:x-intel:context:retailsensingplatform:heartbeat"
	alertsUrn         = "urn:x-intel:context:retailsensingplatform:alerts"
	handheldUrn       = "urn:x-intel:context:handheld:data"
	handheldEventUrn  = "urn:x-intel:context:handheld:event"
	shippingNoticeUrn = "urn:x-intel:context:retailsensingplatform:shippingmasterdata"
	jsonApplication   = "application/json;charset=utf-8"
)

// ZeroMQ implementation of the event publisher
type zeroMQEventPublisher struct {
	publisher *zmq.Socket
	mux       sync.Mutex
}

func main() {
	mDBIndexesError := metrics.GetOrRegisterGauge("Inventory.Main.DBIndexesError", nil)
	mConfigurationError := metrics.GetOrRegisterGauge("Inventory.Main.ConfigurationError", nil)
	mDatabaseRegisterError := metrics.GetOrRegisterGauge("Inventory.Main.DatabaseRegisterError", nil)

	// Load config variables
	err := config.InitConfig()
	fatalErrorHandler("unable to load configuration variables", err, &mConfigurationError)

	// Start healthCheck
	healthCheck(config.AppConfig.Port)

	// Initialize metrics reporting
	initMetrics()

	if config.AppConfig.LoggingLevel == "debug" {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetFormatter(&log.JSONFormatter{})
	}

	log.WithFields(log.Fields{
		"Method": "main",
		"Action": "Start",
	}).Info("Starting inventory management application...")

	dbName := config.AppConfig.DatabaseName
	dbHost := config.AppConfig.ConnectionString + "/" + dbName

	initZmq()

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

	log.WithFields(log.Fields{
		"Method": "main",
		"Action": "Start",
		"Host":   config.AppConfig.ContextSdk,
	}).Infof("Starting Sensing with Secure = %v...", config.AppConfig.SecureMode)

	doContextSensing(masterDB)

	// Initiate webserver and routes
	startWebServer(masterDB, config.AppConfig.Port, config.AppConfig.ResponseLimit, config.AppConfig.ServiceName)

	log.WithField("Method", "main").Info("Completed.")
}

func doContextSensing(masterDB *db.DB) {
	mEventFilterAddListenerError := metrics.GetOrRegisterGauge("Inventory.DoContextSensing.EventFilterAddListenerError", nil)
	mContextBrokerError := metrics.GetOrRegisterGauge("Inventory.DoContextSensing.ContextBrokerError", nil)
	mRRSEventsReceived := metrics.GetOrRegisterGaugeCollection("Inventory.DoContextSensing.RRSEventsReceived", nil)
	mRRSEventsTags := metrics.GetOrRegisterGaugeCollection("Inventory.DoContextSensing.RRSTags", nil)
	mRRSEventsProcessingError := metrics.GetOrRegisterGauge("Inventory.DoContextSensing.RRSEventsError", nil)
	mRRSHeartbeatReceived := metrics.GetOrRegisterGauge("Inventory.DoContextSensing.RRSHeartbeatReceived", nil)
	mRRSHeartbeatProcessingError := metrics.GetOrRegisterGauge("Inventory.DoContextSensing.RRSHeartbeatError", nil)
	mHandheldDataReceived := metrics.GetOrRegisterGaugeCollection("Inventory.DoContextSensing.HandheldDataReceived", nil)
	mHandheldDataTags := metrics.GetOrRegisterGaugeCollection("Inventory.DoContextSensing.HandheldTags", nil)
	mHandheldDataProcessingError := metrics.GetOrRegisterGauge("Inventory.DoContextSensing.HandheldDataError", nil)
	mHandheldEventReceived := metrics.GetOrRegisterGaugeCollection("Inventory.DoContextSensing.HandheldEventReceived", nil)
	mHandheldEventProcessingError := metrics.GetOrRegisterGauge("Inventory.DoContextSensing.HandheldEventError", nil)
	mRRSAlertError := metrics.GetOrRegisterGauge("Inventory.DoContextSensing.RRSAlertError", nil)
	mRRSASNEpcs := metrics.GetOrRegisterGaugeCollection("Inventory.processShippingNotice.RRSASNEpcs", nil)
	mRRSResetEventReceived := metrics.GetOrRegisterGaugeCollection("Inventory.DoContextSensing.RRSResetEventReceived", nil)

	onSensingStarted := make(core.SensingStartedChannel, 1)
	onSensingError := make(core.ErrorChannel, 1)

	options := core.SensingOptions{
		Server:                      config.AppConfig.ContextSdk,
		Publish:                     true,
		Secure:                      config.AppConfig.SecureMode,
		SkipCertificateVerification: config.AppConfig.SkipCertVerify,
		Application:                 config.AppConfig.ServiceName,
		OnStarted:                   onSensingStarted,
		OnError:                     onSensingError,
		Retries:                     10,
		RetryInterval:               1,
	}

	sensingSdk := sensing.NewSensing()
	sensingSdk.Start(options)

	onHeartbeatItem := make(core.ProviderItemChannel, 10)
	onEventsItem := make(core.ProviderItemChannel, 50)
	onAlertsItem := make(core.ProviderItemChannel, 50)
	onHandHeldItem := make(core.ProviderItemChannel, 100)
	onHandHeldEvent := make(core.ProviderItemChannel, 10)
	onShippingNoticeItem := make(core.ProviderItemChannel, 10)

	skuMapping := NewSkuMapping(config.AppConfig.MappingSkuUrl)

	go func(options core.SensingOptions) {

		for {
			select {
			case started := <-options.OnStarted:
				if !started.Started {
					log.WithFields(log.Fields{
						"Method": "main",
						"Action": "connecting to context broker",
						"Host":   config.AppConfig.ContextSdk,
					}).Fatal("sensing has failed to start")
				}

				log.Info("Sensing has started")
				sensingSdk.AddContextTypeListener("*:*", heartbeatUrn, &onHeartbeatItem, &onSensingError)
				sensingSdk.AddContextTypeListener("*:*", eventsUrn, &onEventsItem, &onSensingError)
				sensingSdk.AddContextTypeListener("*:*", alertsUrn, &onAlertsItem, &onSensingError)
				sensingSdk.AddContextTypeListener("*:*", handheldUrn, &onHandHeldItem, &onSensingError)
				sensingSdk.AddContextTypeListener("*:*", shippingNoticeUrn, &onShippingNoticeItem, &onSensingError)

				err := addEventFilterListener(sensingSdk, &onHandHeldEvent, &onSensingError)
				fatalErrorHandler("Exiting due to not able to add listener for Event Filter", err, &mEventFilterAddListenerError)
				log.Info("Waiting for Heartbeat and Events data....")

			case heartbeatItem := <-onHeartbeatItem:
				mRRSHeartbeatReceived.Update(1)
				handleMessage("heartbeat", heartbeatItem, &mRRSHeartbeatProcessingError,
					func(jsonBytes []byte) error { return processHeartBeat(jsonBytes, masterDB) })

			case handheldData := <-onHandHeldItem:
				mHandheldDataReceived.Add(1)

				// Using Go func in case large amount of data so don't starve processing other data
				go func(data *core.ItemData) {
					handleMessage("handheld", handheldData, &mHandheldDataProcessingError,
						func(jsonBytes []byte) error {
							return skuMapping.processTagData(jsonBytes, masterDB,
								"handheld", &mHandheldDataTags)
						})
				}(handheldData)

			case eventsItem := <-onEventsItem:
				mRRSEventsReceived.Add(1)

				// Using Go func in case large amount of data so don't starve processing other data
				go func(events *core.ItemData) {
					handleMessage("fixed", eventsItem, &mRRSEventsProcessingError,
						func(jsonBytes []byte) error {
							return skuMapping.processTagData(jsonBytes, masterDB,
								"fixed", &mRRSEventsTags)
						})
				}(eventsItem)

			case handheldEventItem := <-onHandHeldEvent:
				mHandheldEventReceived.Add(1)
				handleMessage("handheldEvent", handheldEventItem, &mHandheldEventProcessingError,
					func(jsonBytes []byte) error { return processHandheldEvent(jsonBytes, masterDB) })

			case alertsItem := <-onAlertsItem:
				// TODO: this is a mess and needs serious cleanup
				handleMessage("RRS Alert data", alertsItem, &mRRSAlertError,
					func(jsonBytes []byte) error {
						rrsAlert := alert.NewRRSAlert(jsonBytes)
						err := rrsAlert.ProcessAlert()
						if err != nil {
							return err
						}

						if rrsAlert.IsInventoryUnloadAlert() {
							mRRSResetEventReceived.Add(1)
							go func() {
								err := callDeleteTagCollection(masterDB)
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
					})

			case shippingNoticeItem := <-onShippingNoticeItem:
				mRRSASNEpcs.Add(1)

				// Using Go func in case large amount of data so don't starve processing other data
				go func(notices *core.ItemData) {
					// TODO: should this have a different error gauge?
					handleMessage("ASN", shippingNoticeItem, &mRRSEventsProcessingError,
						func(jsonBytes []byte) error {
							return processShippingNotice(jsonBytes, masterDB, &mRRSASNEpcs)
						})
				}(shippingNoticeItem)

			case err := <-options.OnError:
				fatalErrorHandler("Context Sensing Broker Error Received, error exiting...", err.Error, &mContextBrokerError)
			}
		}
	}(options)
}

func handleMessage(dataType string, data *core.ItemData, errGauge *metrics.Gauge, handler func([]byte) error) {
	if data == nil {
		errorHandler(fmt.Sprintf("unable to marshal %s data", dataType),
			errors.New("ItemData was nil"), errGauge)
		return
	}

	jsonBytes, err := json.Marshal(*data)
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

// This function figures out the provider ID of the Event Filter service so we can listen only to handheld events from it.
// This expects the ContextEventFilterProviderID config to be set to the Event Filter provider name that we'll
// use.
func addEventFilterListener(sensingSdk *sensing.Sensing, onHandHeldEvent *core.ProviderItemChannel, onSensingError *core.ErrorChannel) error {
	// If not expecting the Evnet Filter service the ID will be "" and we can skip this.
	if config.AppConfig.ContextEventFilterProviderID == "" {
		return nil
	}

	type Device struct {
		DeviceID   string `json:"deviceID"`
		DeviceName string `json:"deviceName"`
	}

	type Devices struct {
		Devices []Device `json:"devices"`
	}

	type DeviceDiscovery struct {
		Type  string  `json:"type"`
		Value Devices `json:"value"`
	}

	commandReturnChannel := make(chan interface{})
	params := make([]interface{}, 2)
	params[0] = string("0:0:0:0:0:0:sensing")
	params[1] = string("urn:x-intel:context:type:devicediscovery")
	sensingSdk.SendCommand("0:0:0:0:0:0", "sensing", 0, "urn:x-intel:context:command:getitem", params, commandReturnChannel)
	val := <-commandReturnChannel
	errorVal, ok := val.(core.ErrorData)
	if ok {
		return fmt.Errorf("getitem command returned error for finding filter service provider ID: %s", errorVal.Error.Error())
	}

	devicesBytes, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		return errors.Wrap(err, "unable to marshalIdent")
	}
	devices := DeviceDiscovery{}
	if err := json.Unmarshal(devicesBytes, &devices); err != nil {
		return fmt.Errorf("Error unmarshalling devicediscovery json for finding filter service provider ID")
	}

	foundOne := false
	for _, device := range devices.Value.Devices {
		if device.DeviceName == config.AppConfig.ContextEventFilterProviderID {
			foundOne = true
			log.Infof("Adding Event Filter listener for : %s", device.DeviceID)
			sensingSdk.AddContextTypeListener(device.DeviceID, handheldEventUrn, onHandHeldEvent, onSensingError)
			//not breaking here in case there are multiples, and we don't know which one to attach to, so we attach to all
		}
	}

	if !foundOne {
		return fmt.Errorf("No Event Filter service with then name '%s' found in result from Context Broker device discovery", config.AppConfig.ContextEventFilterProviderID)
	}

	return nil
}

func processHandheldEvent(jsonBytes []byte, masterDB *db.DB) error {

	log.Debugf("Received Handheld Event:\n%s", string(jsonBytes))

	var data map[string]interface{}
	decoder := json.NewDecoder(bytes.NewBuffer(jsonBytes))

	if err := decoder.Decode(&data); err != nil {
		return errors.Wrap(err, "Error decoding Handheld Event")
	}

	value := data["value"].(map[string]interface{})
	eventData := handheldevent.HandheldEvent{}

	eventJSONBytes, err := json.Marshal(value)
	if err != nil {
		return errors.Wrap(err, "Error re-encoding Handheld Event value json")
	}

	if err := json.Unmarshal(eventJSONBytes, &eventData); err != nil {
		return errors.Wrap(err, "Error decoding Handheld Event value json into model")
	}

	copySession := masterDB.CopySession()

	// Insert facilities to database and set default coefficients if new facility is inserted
	if err := handheldevent.Insert(copySession, eventData); err != nil {
		copySession.Close()
		return errors.Wrap(err, "Error inserting handheld event")
	}
	copySession.Close()

	if eventData.Event == "Calculate" && config.AppConfig.RulesUrl != "" {
		go func() {
			if err := triggerRules(config.AppConfig.RulesUrl+config.AppConfig.TriggerRulesEndpoint, nil); err != nil {
				// Must log here since in a go function, i.e. can't return the error.
				log.WithFields(log.Fields{
					"Method": "processHandheldEvent",
					"Action": "Trigger rules",
					"Error":  err.Error(),
				}).Error(err)
			}
		}()
	}

	return nil
}

func processHeartBeat(jsonBytes []byte, masterDB *db.DB) error {

	log.Debugf("Received Heartbeat:\n%s", string(jsonBytes))

	var data map[string]interface{}
	decoder := json.NewDecoder(bytes.NewBuffer(jsonBytes))
	decoder.UseNumber()

	if err := decoder.Decode(&data); err != nil {
		return errors.Wrap(err, "Error decoding HeartBeat")
	}

	value := data["value"].(map[string]interface{})
	facilities := value["facilities"].([]interface{})
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
func processShippingNotice(jsonBytes []byte, masterDB *db.DB, tagsGauge *metrics.GaugeCollection) error {

	log.Debugf("Received data:\n%s", string(jsonBytes))

	// SAF says they're going to wrap the ASN in an object under the key "value".
	// They come in an array, though, so there can be multiple that need to be
	// processed together.
	type ASNArrayWrapper struct {
		AsnList []tag.AdvanceShippingNotice `json:"data"`
	}
	type SAFDataWrapper struct {
		WrappedData ASNArrayWrapper `json:"value"`
	}

	var wrapper SAFDataWrapper
	err := json.Unmarshal(jsonBytes, &wrapper)
	if err != nil {
		return errors.Wrap(err, "unable to unmarshal data")
	}

	copySession := masterDB.CopySession()
	// do this before inserting the data into the database
	dailyturn.ProcessIncomingASNList(copySession, wrapper.WrappedData.AsnList)
	copySession.Close()

	var tagData []tag.Tag
	ttlTime := time.Now()

	for _, asn := range wrapper.WrappedData.AsnList {
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

func initZmq() {
	q, _ := zmq.NewSocket(zmq.SUB)
	defer q.Close()

	logrus.Info("Connecting to incoming 0MQ")
	if err := q.Connect("tcp://127.0.0.1:5563"); err != nil {
		logrus.Error(err)
	}
	logrus.Info("Connected to inbound 0MQ")
	q.SetSubscribe("")

	for {
		msg, err := q.RecvMessage(0)
		if err != nil {
			id, _ := q.GetIdentity()
			logrus.Error(fmt.Sprintf("Error getting message %s", id))
		} else {
			for _, str := range msg {
				//event := parseEvent(str)
				logrus.Info(fmt.Sprintf("Event received: %s", str))
				//eventCh <- event
			}
		}
	}
}

func parseEvent(str string) *models.Event {
	event := models.Event{}

	if err := json.Unmarshal([]byte(str), &event); err != nil {
		logrus.Error(err.Error())
		logrus.Warn("Failed to parse event")
		return nil
	}
	return &event
}
