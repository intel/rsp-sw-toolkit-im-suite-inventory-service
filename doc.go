// Inventory Service API.
//
//  The inventory service subscribes to the context-sensing broker to receive sensor data from RRS and Handheld RFID devices.
//	The service processes the fixed and handheld tag, handheld event, and heartbeat data consumed from the broker.
// __Configuration Values__
// The Inventory configuration is split between values set in a configuration file and those set as environment variables in the compose file. The configuration file is expected to be contained in a docker secret for production deployments, but can be on a docker volume for validation and development.
//
//	## __Example configuration file json__:
//  ```json
//  {
//	&#9&#9"serviceName": "RRP - inventory-service",
//  &#9&#9"databaseName": "inventory",
//  &#9&#9"loggingLevel": "debug",
//  &#9&#9"secureMode" : false,
//  &#9&#9"skipCertVerify" : false,
//  &#9&#9"ageOuts": "front:10,back:60",
//  &#9&#9"epcFilters": ["30"],
//  &#9&#9"dailyInventoryPercentage": "0.01",
//  &#9&#9"probUnreadToRead":"0.20",
//  &#9&#9"probInStoreRead":"0.75",
//  &#9&#9"probExitError":"0.10",
//  &#9&#9"triggerRules": "/triggerrules",
//  &#9&#9"purgingDays": "90",
//  &#9&#9"serverReadTimeOutSeconds" : 5,
//  &#9&#9"serverWriteTimeOutSeconds" : 30,
//  &#9&#9"responseLimit": 10000,
//  &#9&#9"contextEventFilterProviderID" : "rrp_handheld_filter",
//  &#9&#9"telemetryEndpoint": "http://166.130.9.122:8000",
//  &#9&#9"telemetryDataStoreName" : "Store105",
//  &#9&#9"port": "8080",
//  &#9&#9"rrsGatewayToCloudURLHost": "abc123.execute-api.us-west-2.amazonaws.com",
//  &#9&#9"rrsGatewayToCloudURLStage": "/prod",
//  &#9&#9"rrsGatewayToCloudRegion": "us-west-2",
//  &#9&#9"rrsGatewayToCloudURLEventEndpoint": "inventoryappevent",
//  &#9&#9"jwtSignerUrl": "http://jwt-signing:8080",
//  &#9&#9"jwtSignerEndpoint": "/jwt-signing/sign",
//  &#9&#9"cloudConnectorUrl" : "http://cloud-connector:8089",
//  &#9&#9"cloudConnectorApiGatewayEndpoint" : "/callwebhook"
//  }
// ```
//
//	## __Example environment variables in compose File__:
//  ```
//  &#9&#9contextSdk: "127.0.0.1:8888"
//  &#9&#9connectionString: "mongodb://127.0.0.1:27017"
//  &#9&#9rules: "http://rules:8080"
//  &#9&#9runtimeConfigPath: "/run/secrets/configuration.json"
// ```
// ###__Configuration file values__
// + `serviceName`  				 - Runtime name of the service
//
// + `databaseName`  				 - Name of database
//
// + `loggingLevel`  				 - Logging level to use: "info" (default) or "debug" (verbose)
//
// + `secureMode`  					 - Boolean flag indicating if using secure connection to the Context Brokers
//
// + `skipCertVerify`  				 - Boolean flag indicating if secure connection to the Context Brokers should skip certificate validation
//
// + `ageOuts`  					 - Automatically mark tags as departed when their last-read timestamp exceeds a threshold configured for that facility. Only for fixed tags and when received on cycle count
//
// + `epcFilters`  					 - Whitelist of EPC prefixes that should be accepted
//
// + `dailyInventoryPercentage`  	 - Percent of inventory that is sold daily
//
// + `probUnreadToRead`  			 - Probability of an unreadable tag becoming readable again each day (i.e. moved or retagged)
//
// + `probInStoreRead`  			 - Probability of a tag in the store being read by the overhead sensor each day
//
// + `probExitError`  				 - Probability of an exit error (missed 'departed' event) occurring
//
// + `triggerRules`  				 - Endpoint used to trigger rules
//
// + `purgingDays`  				 - Number of days from its last read timestamp before a tag will be purged from the database
//
// + `serverReadTimeOutSeconds`  	 - Seconds until server read timeout
//
// + `serverWriteTimeOutSeconds`  	 - Seconds until server write timeout
//
// + `responseLimit`  				 - Default limit to what can be returned in a GET call - because of this, client must define their own top-skip functionality
//
// + `contextEventFilterProviderID`  - ID of the Event Filter Provider service
//
// + `telemetryEndpoint`  				 - URL of the telemetry service receiving the metrics from the service
//
// + `telemetryDataStoreName`  		 - Name of the data store in the telemetry service to store the metrics
//
// + `port`  						 - Port to run the service/s HTTP Server on
//
// ###__Compose file environment variable values__
//
// + `contextSdk`  			- Host and port number for the Context Broker
//
// + `connectionString`  	- Host and port number for the Database connection
//
// + `rules`  				- Base Rules Service URL
//
// + `runtimeConfigPath`  	- Path to the configuration file to use at runtime - Optional. Only needed if not in the standard /run/secrets/configuration.json location
//
// ##__Age Outs:__
// The ageOuts environment variable/configuration is a feature available to automatically mark tags as departed when their timestamp (last read time) exceeds a threshold configured for that facility. The feature is used in environments that do not have exit readers. Aging out tags only happens during cycle counts; that is, the tag's timestamp is only inspected for age out if its current event type is "cycle_count".
// For example, suppose a store has two facilities, "front" and "back". For tags in the "front", they want to consider them departed if they haven't been read for 60 minutes or more. For tags in the "back", they want to consider them departed if they haven't been read for 24 hours or more. In this scenario, they can configure their ageOuts variable as "ageOuts": "front:60,back:1440" (note that 1440 is the number of minutes in 24 hours). When a cycle count occurs, each tag is inspected - if its timestamp + the facility's ageout is less than or equal to the system's current time, the tag's event is changed from "cycle_count" to "departed".
// If a facility is not configured in the ageOuts list, then no tags for that facility will have its event modified by this feature. Note that the facility name is case sensitive. If ageing out is not required for any facility, the string can be left empty ("ageOuts": ""), or simply not configured.
//
// ##__Filters:__
// The "filters" environment variable/configuration is a comma delimited whitelist of EPC prefixes that should be accepted. When a tag comes in, its epc_code is compared to the filter list. If its epc_code starts with any of the values in the filters list, then it will be added to inventory; if not, then it is ignored.
// For example, "filters": "302,300" will only store tags which begin with 302 or 300. If storing all tags is required, specifying a comma seperated list of valid characters is sufficient. E.g, for SGTIN-96, every tag begins with 1-9, so the filter list "1,2,3,4,5,6,7,8,9" would capture all of them. The variable is required, since no filters means no data is stored; moreover, the value should not be empty, since again, no filters -> no data.
// A future specification may change this methodology to accept regular expressions or specific elements of decomposed EPCs, but for now, the filters specify only prefixes.
//
// ## __Known services this service depends on:__
// + context-broker
// + jwt-signing-service
// These are the topics that this service subscribes to from the Context Sensing SDK Websocket bus. To learn more about the Context Sensing SDK, please visit http://contextsensing.intel.com/
// ```
//		&#9eventsUrn         = "urn:x-intel:context:retailsensingplatform:events"
// 		&#9heartbeatUrn      = "urn:x-intel:context:retailsensingplatform:heartbeat"
// 		&#9handheldUrn       = "urn:x-intel:context:handheld:data"
// 		&#9handheldEventUrn  = "urn:x-intel:context:handheld:event"
// ```
// + rrp-mongo
// + rules-service
// + triggerrules
//
// ## __Known services that depend upon this service:__
// + item-finder
// + rules-service
// + nordstrom tran-service
// + cloud-connector-service
//
//     Schemes: http, https
//     Host: inventory:8080
//	   Contact:  RRP <rrp@intel.com>
//     BasePath: /
//     Version: 0.0.1
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//swagger:meta
package main

// forbidden
//
//swagger:response forbidden
type forbidden struct {
}

// externalError
//
//swagger:response externalError
type externalError struct {
}

// serviceUnavailable
//
//swagger:response serviceUnavailable
type serviceUnavailable struct {
}

// externalServiceTimeout
//
//swagger:response externalServiceTimeout
type externalServiceTimeout struct {
}
