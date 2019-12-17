/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package routes

import (
	"database/sql"
	"github.com/gorilla/mux"
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/app/config"

	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/app/routes/handlers"
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/pkg/middlewares"
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/pkg/web"
)

// Route struct holds attributes to declare routes
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc web.Handler
}

// NewRouter creates the routes for GET and POST
func NewRouter(masterDB *sql.DB, maxSize int) *mux.Router {

	inventory := handlers.Inventory{MasterDB: masterDB, MaxSize: maxSize, Url: config.AppConfig.MappingSkuUrl}

	var routes = []Route{
		//swagger:operation GET / default Healthcheck
		//
		// Healthcheck Endpoint
		//
		// Endpoint that is used to determine if the application is ready to take web requests
		//
		// ---
		// consumes:
		// - application/json
		//
		// produces:
		// - application/json
		//
		// schemes:
		// - http
		//
		// responses:
		//   '200':
		//     description: OK
		//
		{
			"Index",
			"GET",
			"/",
			inventory.Index,
		},
		//swagger:route GET /inventory/tags tags getTags
		//
		// Retrieves Tag Data
		//
		// This API call is used to retrieve a list of inventory tags. <br><br>
		//
		// + Search by epc: To search by epc, you would use the filter query parameter like so: /inventory/tags?$filter=(epc eq 'example')
		//
		// /inventory/tags
		// /inventory/tags?$top=10&$select=epc,tid  - Useful for paging data. Grab the top 10 records and only pull back the epc and tid fields
		// /inventory/tags?$count - Shows how many records are in the database
		// /inventory/tags?$filter=(epc eq 'example') and (tid ne '1000030404') - Filters on a particular epc whose tid does not match the one specified
		// /inventory/tags?$filter=startswith(epc,'100') or endswith(epc,'003') or contains(epc,'2') - Allows you to filter based on only certain portions of an epc
		//
		// Example of one object being returned:<br><br>
		// ```
		// {
		// "results":[
		// {
		// 	"arrived": 1501863300375,
		// 	"encode_format": "tbd",
		// 	"epc": "30143639F84191AD22900204",
		// 	"epc_state": "",
		// 	"event": "cycle_count",
		// 	"facility_id": "",
		// 	"fixed": 1,
		// 	"gtin": "00888446671424",
		// 	"company_prefix": 36232,
		// 	"item_filter": 3,
		// 	"handheld": 1,
		// 	"last_read": 1501863300375,
		// 	"location_history": [
		// 	{
		// 	"location": "RSP-95bd71",
		// 	"source": "fixed",
		// 	"timestamp": 1501863300375
		// 	}
		// 	],
		// 	"qualified_state": "unknown",
		// 	"source": "fixed",
		// 	"tid": "",
		// 	"ttl": 1503704119
		// 		}
		// 	]
		// }
		// ```
		//
		// + arrived 		- Arrival time in milliseconds epoch
		// + encode_format 	- TBD
		// + epc 			- SGTIN EPC code
		// + epc_state 		- Current state of tag, either 'present' or 'departed'
		// + event 			- Last event recorded for tag
		// + facility_id 	- Facility ID
		// + fixed 			- Count of how many times tag was read by fixed
		// + gtin 			- GTIN-14 decoded from EPC
		// + company_prefix 	- Part of EPC assigned by GS1
		// + item_filter 	- Part of EPC, denotes packaging level of the item
		// + handheld 		- Count of how many times tag was read by handheld
		// + last_read 		- Tag last read Time in milliseconds epoch
		// + location_history - Array of objects showing tag history
		//    +  location 	- Location of tag at below time
		//    +  source 	- Where tags were read from (fixed or handheld)
		//    +  timestamp 	- Time in milliseconds epoch
		// + qualified_state - Customer defined state
		// + source 			- Where tags were read from (fixed or handheld)
		// + tid 			- Tag manufacturer ID
		// + ttl 			- Time to live, used for db purging - always in sync with last read
		//
		//
		//     Consumes:
		//     - application/json
		//
		//     Produces:
		//     - application/json
		//
		//     Schemes: http
		//
		//     Responses:
		//       200: body:resultsResponse
		//       400: schemaValidation
		//       500: internalError
		//
		{
			"GetTags",
			"GET",
			"/inventory/tags",
			inventory.GetTags,
		},
		//swagger:operation GET /inventory/facilities facilities getFacilities
		//
		// Retrieves Data for Facilities
		//
		// This API call is used to retrieve data for facilities that are configured on RSP.<br><br>
		//
		// /inventory/facilities
		// /inventory/facilities?$filter=(name eq 'CH6_Common_Area') - Filter facilities by name
		//
		// Example Result:
		// ```
		// {
		// "results": [
		// {
		// "coefficients": {
		// "dailyinventorypercentage": 0.01,
		// "probexiterror": 0.1,
		// "probinstoreread": 0.75,
		// "probunreadtoread": 0.2
		// },
		// "name": "CH6"
		// }
		// ]
		// }
		// ```
		//
		// ---
		// consumes:
		// - application/json
		//
		// produces:
		// - application/json
		//
		// schemes:
		// - http
		//
		// responses:
		//   200:
		//     description: OK
		//     schema:
		//       type: object
		//       properties:
		//         results:
		//           type: array
		//           description: Array containing results of query
		//           items:
		//             "$ref": "#/definitions/Facility"
		//         count:
		//           description: Count of records for query
		//           type: integer
		//
		//   400:
		//     description: BadRequest
		//     schema:
		//       "$ref": "#/responses/schemaValidation"
		//   500:
		//     description: InternalError
		//     schema:
		//       "$ref": "#/responses/internalError"
		//
		{
			"GetFacilities",
			"GET",
			"/inventory/facilities",
			inventory.GetFacilities,
		},
		//swagger:operation GET /inventory/handheldevents handheldevents getHandheldevents
		//
		// Retrieves Handheld Event Data
		//
		// This API call is used to retrieve handheld events that have been received.<br><br>
		//
		// + `/inventory/handheldevents`
		// + `/inventory/handheldevents?$filter=(event eq 'FullScanStart')`
		// + `/inventory/handheldevents?$filter=(event eq 'FullScanComplete')`
		// + `/inventory/handheldevents?$filter=(event eq 'Calculate')`
		//
		// Example Result:
		// ```
		// {
		// "results": [
		// {
		// "_id": "59d2818dd0cb6260bf85e3cf",
		// "timestamp": 1506967944919,
		// "event": "FullScanStart"
		// },
		// {
		// "_id": "59d28294d0cb6260bf85f70e",
		// "timestamp": 1506968207311,
		// "event": "FullScanComplete"
		// },
		// {
		// "_id": "59d28294d0cb6260bf85f710",
		// "timestamp": 1506968212265,
		// "event": "Calculate"
		// }]
		// }
		// ```
		//
		// ---
		// consumes:
		// - application/json
		//
		// produces:
		// - application/json
		//
		// schemes:
		// - http
		//
		// responses:
		//   200:
		//     description: OK
		//     schema:
		//       description: Results Response
		//       type: object
		//       properties:
		//         results:
		//           type: array
		//           description: Array containing results of query
		//           items:
		//             "$ref": "#/definitions/HandheldEvent"
		//         count:
		//           description: Count of records for query
		//           type: integer
		//   400:
		//     "$ref": "#/responses/schemaValidation"
		//   500:
		//     "$ref": "#/responses/internalError"
		//
		{
			"GetHandheldEvents",
			"GET",
			"/inventory/handheldevents",
			inventory.GetHandheldEvents,
		},
		//swagger:route POST /inventory/query/current current postCurrentInventory
		//
		// Post current inventory snapshot to the cloud connector
		//
		// Example Request Input:
		// ```
		// {
		// 	"qualified_state":"sold",
		// 	"facility_id":"store001"
		// }
		// ```
		//
		//
		//
		// + __qualified_state__ - User set qualified state for the item
		// + __facility_id__ - Return only facilities provided
		// + __epc_state__ - EPC state of 'present' or 'departed'
		// + __starttime__ - Millisecond epoch start time
		// + __endtime__ - Millisecond epoch stop time
		//
		//
		//
		// ---
		// consumes:
		// - application/json
		//
		// produces:
		// - application/json
		//
		// schemes:
		// - http
		//
		// responses:
		//   200: body:resultsResponse
		//   400: schemaValidation
		//   403: forbidden
		//   500: internalError
		//   502: externalError
		//   503: serviceUnavailable
		//   504: externalServiceTimeout
		//
		{
			"PostCurrentInventory",
			"POST",
			"/inventory/query/current",
			inventory.PostCurrentInventory,
		},
		//swagger:route POST /inventory/query/searchByProductID searchByProductID GetSearchByProductID
		//
		// Retrieves EPC data corresponding to specified ProductID
		//
		// Returns a list of unique EPCs matching the ProductID provided. Body parameters shall be provided in request body in JSON format.<br><br>
		//
		// Example Request Input:
		// ```
		// {
		// "productId":"00012345678905",
		// "facility_id":"store001",
		// "confidence":.75,
		// "cursor":"aGksIDovMSB0aGlz",
		// "size":500,
		// "count_only":false
		// }
		// ```
		//
		//
		// + productId  - A valid productId(GTIN-14) to search for
		// + facility_id  - Return only facilities provided
		// + confidence  - Minimum probability items must meet
		// + cursor  - Cursor from previous response used to retrieve next page of results
		// + size  - Number of results per page
		// + count_only  - Return only tag count
		//
		//
		//
		// Example Response:
		// ```
		// {
		// 	"paging":{
		// 	"cursor":"string"
		// 	},
		// 	"results":[
		// 	{
		// 	"epc":"string",
		// 	"facility_id":"string",
		// 	"event":"string",
		// 	"productId":"string",
		// 	"last_read":0,
		// 	"arrived":0,
		// 	"epc_state":"string",
		// 	"confidence":0,
		// 	"encode_format":"string",
		// 	"tid":"string",
		// 	"qualified_state":"string",
		// 	"epc_context":"string",
		// 	"location_history":[
		// 	{
		// 	"location":"string",
		// 	"timestamp":0
		// 	}
		// 	]
		// 	}
		// 	]
		// }
		// ```
		//
		// + paging  - Paging object
		//    + cursor  - Cursor used to get next page of results
		// + results  - Array of result objects
		//    + epc  - SGTIN EPC code
		//    + facility_id  - Facility ID
		//    + event  - Last event recorded for tag
		//    + productId  - productId(GTIN-14)
		//    + last_read  - Tag last read Time in milliseconds epoch
		//    + arrived  - Arrival time in milliseconds epoch
		//    + epc_state  - Current state of tag, either 'present' or 'departed'
		//    + confidence  - Probability item is in inventory
		//    + encode_format  -
		//    + tid  - Tag manufacturer ID
		//    + qualified_state  - Customer defined state
		//    + epc_context  - Customer defined context
		//    + location_history  - Array of objects showing tag history
		//       + location  	- Location of tag at below time
		//       + timestamp  	- Time in milliseconds epoch
		//
		//     Consumes:
		//     - application/json
		//
		//     Produces:
		//     - application/json
		//
		//     Schemes: http
		//
		// responses:
		//   200: body:resultsResponse
		//   400: schemaValidation
		//   403: forbidden
		//   500: internalError
		//   502: externalError
		//   503: serviceUnavailable
		//   504: externalServiceTimeout
		//
		{
			"GetSearchByProductID",
			"POST",
			"/inventory/query/searchByProductID",
			inventory.GetSearchByProductID,
		},
		//swagger:route PUT /inventory/update/coefficients update updateCoefficients
		//
		// Update Facility Coefficents
		//
		// This API call is used to update probabilistic algorithm coefficients for a particular facility. Coefficient variables are used to calculate the confidence of a tag. Default values are set as configuration variables.<br><br>
		//
		//
		// Example Request Input:
		// ```
		// 	{
		// 	"dailyinventorypercentage": 0.01,
		// 	"probexiterror": 0.1,
		// 	"probinstoreread": 0.75,
		// 	"probunreadtoread": 0.2,
		// 	"facility_id": "Facility"
		// }
		// ```
		//
		//
		// +  dailyinventorypercentage - Percent of inventory that is sold daily
		// +  probexiterror - Probability of an exit error (missed 'departed' event) occurring
		// +  probinstoreread - Probability of a tag in the store being read by the overhead sensor each day
		// +  probunreadtoread - Probability of an unreadable tag becoming readable again each day (i.e. moved or retagged)
		// +  facility_id - Facility name
		//
		//     Consumes:
		//     - application/json
		//
		//     Produces:
		//     - application/json
		//
		//     Schemes: http
		//
		//     Responses:
		//       200: body:resultsResponse
		//       400: schemaValidation
		//       500: internalError
		//
		{
			"UpdateCoefficients",
			"PUT",
			"/inventory/update/coefficients",
			inventory.UpdateCoefficients,
		},
		//swagger:route PUT /inventory/update/qualifiedstate update updateQualifiedState
		//
		// Upload inventory events
		//
		// The update endpoint is for uploading inventory events such as those from a handheld RFID reader.<br><br>
		//
		// Example Request Input:
		// ```
		// {
		// "qualified_state":"string",
		// "epc":"string",
		// "facility_id":"string"
		// }
		// ```
		//
		// + qualified_state  - User-defined state
		// + epc  - SGTIN-96 EPC
		// + facility_id  - Facility code or identifier
		//
		//
		//     Consumes:
		//     - application/json
		//
		//     Produces:
		//     - application/json
		//
		//     Schemes: http
		//
		//     Responses:
		//       200: body:resultsResponse
		//       400: schemaValidation
		//       403: forbidden
		//       500: internalError
		//       502: externalError
		//       503: serviceUnavailable
		//       504: externalServiceTimeout
		//
		{
			"UpdateQualifiedState",
			"PUT",
			"/inventory/update/qualifiedstate",
			inventory.UpdateQualifiedState,
		},
		//swagger:route POST /inventory/search epc getSearchByEpc
		//
		// Retrieves tag data corresponding to specified EPC pattern
		//
		// Returns a list of tags with their EPCs matching a pattern. Body parameters shall be provided in request body in JSON format.<br><br>
		//
		// Example Request Input:
		// ```
		// {
		// "epc":"3038E511C6E9A6400012D687",
		// "facility_id":"store001",
		// "cursor":"aGksIDovMSB0aGlz",
		// "size":500
		// }
		// ```
		//
		// + epc  - EPC search string which can contain a single asterisk at the beginning, middle, or end of EPC string
		// + facility_id  - Facility code or identifier
		// + cursor  - Cursor from previous response used to retrieve next page of results
		// + size  - Number of results per page
		//
		// Example Response:
		// ```
		// {
		// 	"paging":{
		// 	"cursor":"string"
		// 	},
		// 	"results":[
		// 	{
		// 	"epc":"string",
		// 	"facility_id":"string",
		// 	"event":"string",
		// 	"gtin":"string",
		// 	"last_read":0,
		// 	"arrived":0,
		// 	"epc_state":"string",
		// 	"confidence":0,
		// 	"encode_format":"string",
		// 	"tid":"string",
		// 	"qualified_state":"string",
		// 	"epc_context":"string",
		// 	"location_history":[
		// 	{
		// 	"location":"string",
		// 	"timestamp":0
		// 	}
		// 	]
		// 	}
		// 	]
		// }
		// ```
		//
		// + paging  - Paging object
		//    + cursor  - Cursor used to get next page of results
		// + results  - Array of result objects
		//    + epc  - SGTIN EPC code
		//    + facility_id  - Facility ID
		//    + event  - Last event recorded for tag
		//    + gtin  - GTIN-14 decoded from EPC
		//    + last_read  - Tag last read Time in milliseconds epoch
		//    + arrived  - Arrival time in milliseconds epoch
		//    + epc_state  - Current state of tag, either 'present' or 'departed'
		//    + confidence  - Probability item is in inventory
		//    + encode_format  -
		//    + tid  - Tag manufacturer ID
		//    + qualified_state  - Customer defined state
		//    + epc_context  - Customer defined context
		//    + location_history  - Array of objects showing tag history
		//       + location  	- Location of tag at below time
		//       + timestamp  	- Time in milliseconds epoch
		//
		//     Consumes:
		//     - application/json
		//
		//     Produces:
		//     - application/json
		//
		//     Schemes: http
		//
		//     Responses:
		//       200: body:resultsResponse
		//       400: schemaValidation
		//       403: forbidden
		//       500: internalError
		//       502: externalError
		//       503: serviceUnavailable
		//       504: externalServiceTimeout
		//
		{
			"GetSearchByEpc",
			"POST",
			"/inventory/search",
			inventory.GetSearchByEpc,
		},
		//swagger:route PUT /inventory/update/epccontext epc setEpcContext
		//
		// Set EPC context
		//
		// This endpoint allows the customer to arbitrarily set the context for a particular EPC. For example, the customer may want to mark the tag as received, sold, lost, stolen, and anything else the customer decides is appropriate. Body parameters shall be provided in request body in JSON format.<br><br>
		//
		// Example Request Input:
		// ```
		// {
		// "epc_context":"received",
		// "epc":"3038E511C6E9A6400012D687",
		// "facility_id":"store555"
		// }
		// ```
		//
		// + epc_context  - User-defined context
		// + facility_id  - Facility code or identifier
		// + epc  - SGTIN-96 EPC
		//
		//
		//     Consumes:
		//     - application/json
		//
		//     Produces:
		//     - application/json
		//
		//     Schemes: http
		//
		//     Responses:
		//       200: body:resultsResponse
		//       400: schemaValidation
		//       403: forbidden
		//       500: internalError
		//       502: externalError
		//       503: serviceUnavailable
		//       504: externalServiceTimeout
		//
		{
			"SetEpcContext",
			"PUT",
			"/inventory/update/epccontext",
			inventory.SetEpcContext,
		},
		//swagger:route DELETE /inventory/update/epccontext epc deleteEpcContext
		//
		// Delete EPC context
		//
		// This endpoint allows the customer to delete the context for a particular EPC. Body parameters shall be provided in request body in JSON format.<br><br>
		//
		// Example Request Input:
		// ```
		// {
		// "epc":"3038E511C6E9A6400012D687",
		// "facility_id":"store100"
		// }
		// ```
		//
		// + epc  - SGTIN-96 EPC
		// + facility_id  - Facility code or identifier
		//
		//
		//     Consumes:
		//     - application/json
		//
		//     Produces:
		//     - application/json
		//
		//     Schemes: http
		//
		//     Responses:
		//       200: body:resultsResponse
		//       400: schemaValidation
		//       403: forbidden
		//       500: internalError
		//       502: externalError
		//       503: serviceUnavailable
		//       504: externalServiceTimeout
		//
		{
			"DeleteEpcContext",
			"DELETE",
			"/inventory/update/epccontext",
			inventory.DeleteEpcContext,
		},
		//swagger:route DELETE /inventory/tags tags deleteAllTags
		//
		// Delete Tag Collection in database
		//
		// This endpoint allows the customer to delete all the tags in the tags table.<br><br>
		//
		//     Consumes:
		//     - application/json
		//
		//     Produces:
		//     - application/json
		//
		//     Schemes: http
		//
		//     Responses:
		//       204: body:resultsResponse
		//       400: schemaValidation
		//       403: forbidden
		//       500: internalError
		//       502: externalError
		//       503: serviceUnavailable
		//       504: externalServiceTimeout
		//
		{
			"DeleteAllTags",
			"DELETE",
			"/inventory/tags",
			inventory.DeleteAllTags,
		},
	}

	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {

		var handler = route.HandlerFunc
		handler = middlewares.Recover(handler)
		handler = middlewares.Logger(handler)
		handler = middlewares.Bodylimiter(handler)
		if config.AppConfig.EnableCORS {
			handler = middlewares.CORS(config.AppConfig.CORSOrigin, handler)
		}

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}

	return router
}
