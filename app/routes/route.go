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

package routes

import (
	"github.com/gorilla/mux"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"

	db "github.impcloud.net/RSP-Inventory-Suite/go-dbWrapper"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/routes/handlers"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/middlewares"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/web"
)

// Route struct holds attributes to declare routes
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc web.Handler
}

// NewRouter creates the routes for GET and POST
func NewRouter(masterDB *db.DB, maxSize int) *mux.Router {

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
		// &#8195"results":[
		// &#8195&#8195&#8195{
		// 	&#9"arrived": 1501863300375,
		// 	&#9"encode_format": "tbd",
		// 	&#9"epc": "30143639F84191AD22900204",
		// 	&#9"epc_state": "",
		// 	&#9"event": "cycle_count",
		// 	&#9"facility_id": "",
		// 	&#9"fixed": 1,
		// 	&#9"gtin": "00888446671424",
		// 	&#9"company_prefix": 36232,
		// 	&#9"item_filter": 3,
		// 	&#9"handheld": 1,
		// 	&#9"last_read": 1501863300375,
		// 	&#9"location_history": [
		// 	&#8195&#8195&#9{
		// 	&#9&#9"location": "RSP-95bd71",
		// 	&#9&#9"source": "fixed",
		// 	&#9&#9"timestamp": 1501863300375
		// 	&#8195&#8195&#9}
		// 	&#9],
		// 	&#9"qualified_state": "unknown",
		// 	&#9"source": "fixed",
		// 	&#9"tid": "",
		// 	&#9"ttl": 1503704119
		// 	&#8195&#8195&#8195	}
		// 	&#8195]
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
		// This API call is used to retrieve facilities that are configured on RRS.<br><br>
		//
		// /inventory/facilities
		// /inventory/facilities?$filter=(name eq 'CH6_Common_Area') - Filter facilities by name
		//
		// Example Result:
		// ```
		// {
		// &#8195"results": [
		// &#8195&#8195{
		// &#8195&#8195&#8195&#8195"coefficients": {
		// &#9"dailyinventorypercentage": 0.01,
		// &#9"probexiterror": 0.1,
		// &#9"probinstoreread": 0.75,
		// &#9"probunreadtoread": 0.2
		// &#8195&#8195&#8195&#8195},
		// &#8195&#8195&#8195"name": "CH6"
		// &#8195&#8195}
		// &#8195]
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
		//           type: int
		//
		//   400:
		//     schema:
		//       "$ref": "#/definitions/schemaValidation"
		//   500:
		//     schema:
		//       "$ref": "#/definitions/internalError"
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
		// &#8195"results": [
		// &#8195&#8195{
		// &#9"_id": "59d2818dd0cb6260bf85e3cf",
		// &#9"timestamp": 1506967944919,
		// &#9"event": "FullScanStart"
		// &#8195&#8195},
		// &#8195&#8195{
		// &#9"_id": "59d28294d0cb6260bf85f70e",
		// &#9"timestamp": 1506968207311,
		// &#9"event": "FullScanComplete"
		// &#8195&#8195},
		// &#8195&#8195{
		// &#9"_id": "59d28294d0cb6260bf85f710",
		// &#9"timestamp": 1506968212265,
		// &#9"event": "Calculate"
		// &#8195&#8195}]
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
		//           type: int
		//   400:
		//     "$ref": "#/definitions/schemaValidation"
		//   500:
		//     "$ref": "#/definitions/internalError"
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
		// Example Input:
		// ```
		// {
		// 	&#8195&#8195"qualified_state":"sold",
		// 	&#8195&#8195"facility_id":"store001"
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
		//swagger:route POST /inventory/query/missingtags missingtags getMissingTags
		//
		// Retrieves missing tag data
		//
		// Returns a list of unique tags that have not been read by a reader since a defined timestamp. Body parameters shall be provided in request body in JSON format.<br><br>
		//
		// Example Input:
		// ```
		// {
		// &#9"facility_id":"store99",
		// &#9"time":1495575432000,
		// &#9"confidence":.5,
		// &#9"cursor":"abcd1023abcd",
		// &#9"size":500,
		// &#9"count_only":false
		// }
		// ```
		//
		// + facility_id  - Return only facilities provided
		// + time  - "Not read since" time in epoch milliseconds
		// + confidence  - Minimum probability items must meet
		// + cursor  - Cursor from previous response used to retrieve next page of results
		// + size  - Number of results per page
		// + count_only  - Return only tag count
		//
		//
		// Example Response:
		// ```
		// {
		// 	&#8195"paging":{
		// 	&#8195&#8195&#8195"cursor":"string"
		// 	&#8195},
		// 	&#8195&#8195"results":[
		// 	&#8195&#8195{
		// 	&#9"epc":"string",
		// 	&#9"facility_id":"string",
		// 	&#9"event":"string",
		// 	&#9"gtin":"string",
		// 	&#9"last_read":0,
		// 	&#9"arrived":0,
		// 	&#9"epc_state":"string",
		// 	&#9"confidence":0,
		// 	&#9"encode_format":"string",
		// 	&#9"tid":"string",
		// 	&#9"qualified_state":"string",
		// 	&#9"epc_context":"string",
		// 	&#9"location_history":[
		// 	&#9&#8195&#8195{
		// 	&#9&#9"location":"string",
		// 	&#9&#9"timestamp":0
		// 	&#9&#8195&#8195}
		// 	&#9]
		// 	&#8195&#8195}
		// 	&#8195]
		// }
		// ```
		//
		//
		// + paging  - Paging object
		//   + cursor  - Cursor used to get next page of results
		// + results  - Array of result objects
		//   + epc  - SGTIN EPC code
		//   + facility_id  - Facility ID
		//   + event  - Last event recorded for tag
		//   + gtin  - GTIN-14 decoded from EPC
		//   + last_read  - Tag last read Time in milliseconds epoch
		//   + arrived  - Arrival time in milliseconds epoch
		//   + epc_state  - Current state of tag, either 'present' or 'departed'
		//   + confidence  - Probability item is in inventory
		//   + encode_format  -
		//   + tid  - Tag manufacturer ID
		//   + qualified_state  - Customer defined state
		//   + epc_context  - Customer defined context
		//   + location_history  - Array of objects showing tag history
		//      + location - Location of tag at below time
		//      + timestamp - Time in milliseconds epoch
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
			"GetMissingTags",
			"POST",
			"/inventory/query/missingtags",
			inventory.GetMissingTags,
		},
		//swagger:route POST /inventory/query/searchbygtin searchbygtin searchByGtin
		//
		// Retrieves EPC data corresponding to specified GTIN(s)
		//
		// Returns a list of unique EPCs matching the GTIN(s) provided. Body parameters shall be provided in request body in JSON format.<br><br>
		//
		// Example Input:
		// ```
		// {
		// &#9"gtin":"00012345678905",
		// &#9"facility_id":"store001",
		// &#9"confidence":.75,
		// &#9"cursor":"aGksIDovMSB0aGlz",
		// &#9"size":500,
		// &#9"count_only":false
		// }
		// ```
		//
		//
		// + gtin  - A valid GTIN-14 to search for
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
		// 	&#8195"paging":{
		// 	&#8195&#8195&#8195"cursor":"string"
		// 	&#8195},
		// 	&#8195&#8195"results":[
		// 	&#8195&#8195{
		// 	&#9"epc":"string",
		// 	&#9"facility_id":"string",
		// 	&#9"event":"string",
		// 	&#9"gtin":"string",
		// 	&#9"last_read":0,
		// 	&#9"arrived":0,
		// 	&#9"epc_state":"string",
		// 	&#9"confidence":0,
		// 	&#9"encode_format":"string",
		// 	&#9"tid":"string",
		// 	&#9"qualified_state":"string",
		// 	&#9"epc_context":"string",
		// 	&#9"location_history":[
		// 	&#9&#8195&#8195{
		// 	&#9&#9"location":"string",
		// 	&#9&#9"timestamp":0
		// 	&#9&#8195&#8195}
		// 	&#9]
		// 	&#8195&#8195}
		// 	&#8195]
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
			"GetSearchByGtin",
			"POST",
			"/inventory/query/searchbygtin",
			inventory.GetSearchByGtin,
		},
		//swagger:route PUT /inventory/update/coefficients update updateCoefficients
		//
		// Update Facility Coefficents
		//
		// This API call is used to retrieve handheld events that have been received. Coefficient variables are used to calculate the confidence of a tag. Default values are set as configuration variables. When a Heartbeat is received from RRS, the inventory-service extracts the one or more facility_id configured in the RSP Controller and if it doesn't exist in the database, applies the default coefficient values to each facility. <br><br>
		//
		//
		// Example Schema:
		// ```
		// 	{
		// 	&#8195&#8195"coefficients": {
		// 	&#8195&#8195&#9"dailyinventorypercentage": 0.01,
		// 	&#8195&#8195&#9"probexiterror": 0.1,
		// 	&#8195&#8195&#9"probinstoreread": 0.75,
		// 	&#8195&#8195&#9"probunreadtoread": 0.2
		// 	&#8195&#8195},
		// 	&#8195&#8195"name": "Facility"
		// }
		// ```
		//
		//
		// + coefficients - The coefficients used in the probabilistic inventory algorithm
		//    +  dailyinventorypercentage - Percent of inventory that is sold daily
		//    +  probexiterror - Probability of an exit error (missed 'departed' event) occurring
		//    +  probinstoreread - Probability of a tag in the store being read by the overhead sensor each day
		//    +  probunreadtoread - Probability of an unreadable tag becoming readable again each day (i.e. moved or retagged)
		// + name - Facility name
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
		// Example Input:
		// ```
		// {
		// &#9"qualified_state":"string",
		// &#9"epc":"string",
		// &#9"facility_id":"string"
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
		// Example Input:
		// ```
		// {
		// &#9"epc":"3038E511C6E9A6400012D687",
		// &#9"facility_id":"store001",
		// &#9"cursor":"aGksIDovMSB0aGlz",
		// &#9"size":500
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
		// 	&#8195"paging":{
		// 	&#8195&#8195&#8195"cursor":"string"
		// 	&#8195},
		// 	&#8195&#8195"results":[
		// 	&#8195&#8195{
		// 	&#9"epc":"string",
		// 	&#9"facility_id":"string",
		// 	&#9"event":"string",
		// 	&#9"gtin":"string",
		// 	&#9"last_read":0,
		// 	&#9"arrived":0,
		// 	&#9"epc_state":"string",
		// 	&#9"confidence":0,
		// 	&#9"encode_format":"string",
		// 	&#9"tid":"string",
		// 	&#9"qualified_state":"string",
		// 	&#9"epc_context":"string",
		// 	&#9"location_history":[
		// 	&#9&#8195&#8195{
		// 	&#9&#9"location":"string",
		// 	&#9&#9"timestamp":0
		// 	&#9&#8195&#8195}
		// 	&#9]
		// 	&#8195&#8195}
		// 	&#8195]
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
		// Example Input:
		// ```
		// {
		// &#9"epc_context":"received",
		// &#9"epc":"3038E511C6E9A6400012D687",
		// &#9"facility_id":"store555"
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
		// Example Input:
		// ```
		// {
		// &#9"epc":"3038E511C6E9A6400012D687",
		// &#9"facility_id":"store100"
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
		// This endpoint allows the customer to delete the context for a particular EPC. Body parameters shall be provided in request body in JSON format.<br><br>
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

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}

	return router
}
