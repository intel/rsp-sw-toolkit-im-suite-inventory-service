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

package config

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/encodingscheme"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/configuration"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
)

const (
	maxServerReadTimeoutSeconds   = 1800
	maxServerWriteTimeoutSeconds  = 1800
	maxCloudConnectorRetrySeconds = 60
)

type (
	variables struct {
		ServiceName, ConnectionString, DatabaseName, LoggingLevel, ZeroMQ, Port                        string
		TelemetryEndpoint, TelemetryDataStoreName                                                      string
		DailyInventoryPercentage, ProbUnreadToRead, ProbInStoreRead, ProbExitError                     float64 // Coefficients
		EndpointConnectionTimedOutSeconds                                                              int
		AgeOuts                                                                                        map[string]int
		EpcFilters                                                                                     []string
		RulesUrl, TriggerRulesEndpoint, CloudConnectorUrl, CloudConnectorApiGatewayEndpoint            string // service endpionts
		RfidAlertURL, RfidAlertMessageEndpoint                                                         string
		ContraEpcPartition                                                                             int
		ContextEventFilterProviderID                                                                   string
		PurgingDays                                                                                    int
		ServerReadTimeOutSeconds                                                                       int
		ServerWriteTimeOutSeconds                                                                      int
		ResponseLimit                                                                                  int
		SecureMode                                                                                     bool
		SkipCertVerify                                                                                 bool
		TriggerRulesOnFixedTags                                                                        bool
		NewerHandheldHavePriority                                                                      bool
		MappingSkuUrl                                                                                  string
		EventDestination, EventDestinationAuthEndpoint, EventDestinationAuthType                       string
		EventDestinationClientID, EventDestinationClientSecret                                         string
		EpcToWrin                                                                                      bool
		DailyInventoryPercentageLabel, ProbUnreadToReadLabel, ProbInStoreReadLabel, ProbExitErrorLabel string
		AdvancedShippingNoticeFacilityID                                                               string
		CloudConnectorRetrySeconds                                                                     int
		DailyTurnMinimumDataPoints, DailyTurnHistoryMaximum                                            int
		DailyTurnComputeUsingMedian                                                                    bool
		UseComputedDailyTurnInConfidence                                                               bool
		ProbPlugin                                                                                     bool
		TagDecoders                                                                                    []encodingscheme.TagDecoder
	}
)

// AppConfig exports all config variables
var AppConfig variables

// InitConfig loads application variables
// nolint :gocyclo
func InitConfig() error {
	AppConfig = variables{}

	config, err := configuration.NewConfiguration()
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.ServiceName, err = config.GetString("serviceName")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.ConnectionString, err = helper.GetSecret("connectionString")
	if err != nil {
		AppConfig.ConnectionString, err = config.GetString("connectionString")
		if err != nil {
			return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
		}
	}

	AppConfig.DatabaseName, err = config.GetString("databaseName")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.ZeroMQ, err = config.GetString("zeroMQ")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	// ageOutString is optional
	ageOutString, err := config.GetString("ageOuts")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}
	if len(ageOutString) > 0 {
		// since we have an ageOutString, try to parse it
		AppConfig.AgeOuts, err = parseAgeOuts(ageOutString)
		if err != nil {
			return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
		}
	}

	AppConfig.EndpointConnectionTimedOutSeconds, err = config.GetInt("endpointConnectionTimedOutSeconds")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}
	if AppConfig.EndpointConnectionTimedOutSeconds < 1 {
		return errors.New("EndpointConnectionTimedOutSeconds cannot be lesser than 1")
	}
	if AppConfig.EndpointConnectionTimedOutSeconds > maxServerReadTimeoutSeconds {
		// limit to max value
		log.Debugf("EndpointConnectionTimedOutSeconds value %d exceeds the max value allowed, set to max value %d",
			AppConfig.EndpointConnectionTimedOutSeconds, maxServerReadTimeoutSeconds)
		AppConfig.EndpointConnectionTimedOutSeconds = maxServerReadTimeoutSeconds
	}

	// get prefix filters; they are not optional, since not having one would mean no tags are saved
	AppConfig.EpcFilters, err = config.GetStringSlice("epcFilters")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	purgingDaysString, err := config.GetString("purgingDays")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}
	AppConfig.PurgingDays, err = strconv.Atoi(purgingDaysString)
	if err != nil {
		return errors.Wrapf(err, "Unable to parse PurgingDays: %s", err.Error())
	}

	AppConfig.ServerReadTimeOutSeconds, err = config.GetInt("serverReadTimeOutSeconds")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}
	if AppConfig.ServerReadTimeOutSeconds < 1 {
		return errors.New("ServerReadTimeOutSeconds cannot be lesser than 1")
	}
	if AppConfig.ServerReadTimeOutSeconds > maxServerReadTimeoutSeconds {
		// limit to max value
		log.Debugf("serverReadTimeOutSeconds value %d exceeds the max value allowed, set to max value %d",
			AppConfig.ServerReadTimeOutSeconds, maxServerReadTimeoutSeconds)
		AppConfig.ServerReadTimeOutSeconds = maxServerReadTimeoutSeconds
	}

	AppConfig.ServerWriteTimeOutSeconds, err = config.GetInt("serverWriteTimeOutSeconds")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}
	if AppConfig.ServerWriteTimeOutSeconds < 1 {
		return errors.New("ServerWriteTimeOutSeconds cannot be lesser than 1")
	}
	if AppConfig.ServerWriteTimeOutSeconds > maxServerWriteTimeoutSeconds {
		// limit to max value
		log.Debugf("serverWriteTimeOutSeconds value %d exceeds the max value allowed, set to max value %d",
			AppConfig.ServerWriteTimeOutSeconds, maxServerWriteTimeoutSeconds)
		AppConfig.ServerWriteTimeOutSeconds = maxServerWriteTimeoutSeconds
	}

	AppConfig.ContraEpcPartition, err = config.GetInt("contraEpcPartition")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	// get the Provider id for the Context Event filter service so events are only received from that service.
	AppConfig.ContextEventFilterProviderID, err = config.GetString("contextEventFilterProviderID")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.Port, err = config.GetString("port")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	// Set "debug" for development purposes. Nil or "" for Production.
	AppConfig.LoggingLevel, err = config.GetString("loggingLevel")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	// Parse coefficients
	if err := parseCoefficients(&AppConfig, config); err != nil {
		return errors.Wrap(err, "unable to parse coefficients")
	}

	AppConfig.RulesUrl, err = config.GetString("rulesUrl")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.TriggerRulesEndpoint, err = config.GetString("triggerRulesEndpoint")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.CloudConnectorUrl, err = config.GetString("cloudConnectorUrl")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.CloudConnectorApiGatewayEndpoint, err = config.GetString("cloudConnectorApiGatewayEndpoint")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.SecureMode, err = config.GetBool("secureMode")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}
	AppConfig.TelemetryEndpoint, err = config.GetString("telemetryEndpoint")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.TelemetryDataStoreName, err = config.GetString("telemetryDataStoreName")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.SkipCertVerify, err = config.GetBool("skipCertVerify")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.TriggerRulesOnFixedTags, err = config.GetBool("triggerRulesOnFixedTags")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.NewerHandheldHavePriority, err = config.GetBool("newerHandheldHavePriority")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	// size limit of RESTFul endpoints
	AppConfig.ResponseLimit, err = config.GetInt("responseLimit")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.EventDestination, err = config.GetString("eventDestination")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.EventDestinationAuthEndpoint, err = helper.GetSecret("eventDestinationAuthEndpoint")
	if err != nil {
		AppConfig.EventDestinationAuthEndpoint, err = config.GetString("eventDestinationAuthEndpoint")
		if err != nil {
			return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
		}
	}

	AppConfig.EventDestinationAuthType, err = helper.GetSecret("eventDestinationAuthType")
	if err != nil {
		AppConfig.EventDestinationAuthType, err = config.GetString("eventDestinationAuthType")
		if err != nil {
			return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
		}
	}

	AppConfig.EventDestinationClientID, err = helper.GetSecret("eventDestinationClientID")
	if err != nil {
		AppConfig.EventDestinationClientID, err = config.GetString("eventDestinationClientID")
		if err != nil {
			return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
		}
	}

	AppConfig.EventDestinationClientSecret, err = helper.GetSecret("eventDestinationClientSecret")
	if err != nil {
		AppConfig.EventDestinationClientSecret, err = config.GetString("eventDestinationClientSecret")
		if err != nil {
			return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
		}
	}

	AppConfig.RfidAlertURL, err = config.GetString("rfidAlertURL")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.RfidAlertMessageEndpoint, err = config.GetString("rfidAlertMessageEndpoint")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.MappingSkuUrl, err = config.GetString("mappingSkuUrl")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.DailyInventoryPercentageLabel, err = config.GetString("dailyInventoryPercentageLabel")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.ProbUnreadToReadLabel, err = config.GetString("probUnreadToReadLabel")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.ProbInStoreReadLabel, err = config.GetString("probInStoreReadLabel")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.ProbExitErrorLabel, err = config.GetString("probExitErrorLabel")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.AdvancedShippingNoticeFacilityID, err = config.GetString("advancedShippingNoticeFacilityID")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.CloudConnectorRetrySeconds, err = config.GetInt("cloudConnectorRetrySeconds")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}
	if AppConfig.CloudConnectorRetrySeconds < 1 {
		return errors.New("CloudConnectorRetrySeconds cannot be lesser than 1")
	}
	if AppConfig.CloudConnectorRetrySeconds > maxCloudConnectorRetrySeconds {
		// limit to max value
		log.Debugf("CloudConnectorRetrySeconds value %d exceeds the max value allowed, set to max value %d",
			AppConfig.CloudConnectorRetrySeconds, maxCloudConnectorRetrySeconds)
		AppConfig.CloudConnectorRetrySeconds = maxCloudConnectorRetrySeconds
	}

	AppConfig.DailyTurnMinimumDataPoints, err = config.GetInt("dailyTurnMinimumDataPoints")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}
	if AppConfig.DailyTurnMinimumDataPoints < 1 {
		return errors.New("dailyTurnMinimumDataPoints must be at least 1")
	}

	AppConfig.DailyTurnHistoryMaximum, err = config.GetInt("dailyTurnHistoryMaximum")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}
	if AppConfig.DailyTurnHistoryMaximum < 1 {
		return errors.New("dailyTurnHistoryMaximum must be at least 1")
	}
	if AppConfig.DailyTurnHistoryMaximum < AppConfig.DailyTurnMinimumDataPoints {
		return errors.Errorf("dailyTurnHistoryMaximum must be greater or equal to dailyTurnMinimumDataPoints. Values: max:%d, min:%d",
			AppConfig.DailyTurnHistoryMaximum, AppConfig.DailyTurnMinimumDataPoints)
	}

	AppConfig.DailyTurnComputeUsingMedian, err = config.GetBool("dailyTurnComputeUsingMedian")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.UseComputedDailyTurnInConfidence, err = config.GetBool("useComputedDailyTurnInConfidence")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.TagDecoders, err = getTagDecoders(config)
	if err != nil {
		return err
	}

	AppConfig.ProbPlugin, err = config.GetBool("probPlugin")
	if err != nil {
		log.Warn("Probabilistic Plugin (probPlugin) variable not set. Default to true.")
		AppConfig.ProbPlugin = true
	}

	return nil
}

func getTagDecoders(config *configuration.Configuration) ([]encodingscheme.TagDecoder, error) {
	// always use SGTIN-96
	decoders := []encodingscheme.TagDecoder{encodingscheme.SGTIN96Decoder()}

	// if configured, also use a proprietary extractor
	var extractor encodingscheme.TagDecoder
	extractor, err := parseProprietary(config)
	if err != nil {
		return decoders, err
	}
	if extractor != nil {
		decoders = append(decoders, extractor)
	}
	return decoders, nil
}

func parseProprietary(config *configuration.Configuration) (encodingscheme.TagDecoder, error) {
	fields, err := config.GetString("proprietaryTagFormat")
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}
	if fields == "" {
		log.Debug("skipping proprietary parsing, because no format is present")
		return nil, nil
	}

	widths, err := config.GetString("proprietaryTagBitBoundary")
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}
	if widths == "" {
		return nil, errors.New("there's a proprietary tag format, but no boundaries")
	}

	authority, err := config.GetString("tagURIAuthorityName")
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}
	if authority == "" {
		return nil, errors.New("there's a proprietary tag format, but no tagURIAuthorityName")
	}

	date, err := config.GetString("tagURIAuthorityDate")
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}
	if date == "" {
		return nil, errors.New("there's a proprietary tag format, but no tagURIAuthorityDate")
	}

	return encodingscheme.NewProprietary(authority, date, fields, widths)
}

func parseCoefficients(AppConfig *variables, config *configuration.Configuration) error {

	var err error

	// Parsing coefficient variables
	dailyString, err := config.GetString("dailyInventoryPercentage")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}
	unReadString, err := config.GetString("probUnreadToRead")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}
	inStoreString, err := config.GetString("probInStoreRead")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}
	exitErrorString, err := config.GetString("probExitError")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	// Parsing string to float64
	AppConfig.DailyInventoryPercentage, err = strconv.ParseFloat(dailyString, 64)
	if err != nil {
		return errors.Wrapf(err, "Unable to parse DailyInventoryPercentage: %s", err.Error())
	}

	AppConfig.ProbUnreadToRead, err = strconv.ParseFloat(unReadString, 64)
	if err != nil {
		return errors.Wrapf(err, "Unable to parse ProbUnreadToRead: %s", err.Error())
	}

	AppConfig.ProbInStoreRead, err = strconv.ParseFloat(inStoreString, 64)
	if err != nil {
		return errors.Wrapf(err, "Unable to parse ProbInStoreRead: %s", err.Error())
	}

	AppConfig.ProbExitError, err = strconv.ParseFloat(exitErrorString, 64)
	if err != nil {
		return errors.Wrapf(err, "Unable to parse ProbExitError: %s", err.Error())
	}

	return nil
}

func parseAgeOuts(ageOutString string) (map[string]int, error) {
	ageOuts := make(map[string]int)
	// an empty string is valid, but should just be an empty map
	if len(ageOutString) == 0 {
		return ageOuts, nil
	}

	// the entire ageOutsString is facility:time,facility:time
	for _, tuple := range strings.Split(ageOutString, ",") {
		// split the tuples at ":"
		parts := strings.Split(tuple, ":")
		if len(parts) != 2 {
			return nil, errors.Errorf("String %s is not a valid facility:time tuple", tuple)
		}

		// make sure the facility isn't empty
		facility := parts[0]
		if facility == "" {
			return nil, errors.Errorf("ageOuts includes a tuple with an empty facility")
		}

		// extract and convert the minutes to an int
		minutes, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, errors.Wrapf(err, "Minutes in %s is not a valid integer: %s", tuple, err.Error())
		}

		if minutes < 0 {
			return nil, errors.Errorf("Minutes in ageOuts string %s should be positive", tuple)
		}

		ageOuts[facility] = minutes
	}

	return ageOuts, nil
}
