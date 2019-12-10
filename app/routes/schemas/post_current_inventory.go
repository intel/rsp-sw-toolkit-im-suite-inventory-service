/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package schemas

const PostCurrentInventorySchema = `{
	"type": "object",
	"properties": {
		"facility_id": {
			"type": "string"
		},
		"qualified_state": {
			"type": "string"
		},
		"epc_state": {
			"type": "string"
		},
		"starttime": {
			"type": "integer"
		},
		"endtime": {
			"type": "integer"
		}
	},
	"additionalProperties": false
}`
