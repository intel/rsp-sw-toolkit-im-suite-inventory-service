/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package schemas

// CoefficientsSchema gets the json schema to update coefficients
const CoefficientsSchema = `{
	"type": "object",
	"required": [
		"dailyinventorypercentage",
		"probunreadtoread",
		"probinstoreread",
		"probexiterror",
		"facility_id"
	],
	"properties": {	
		"facility_id": {
			"type": "string"
		},
		"dailyinventorypercentage": {
			"type": "number"
		},
		"probunreadtoread": {
			"type": "number"
		},
		"probinstoreread": {
			"type": "number"
		},
		"probexiterror": {
			"type": "number"
		}
	},
	"additionalProperties": false
}`
