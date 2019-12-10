/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package schemas

// SearchByGtinSchema required for request body validation
const SearchByGtinSchema = `{
	"type": "object",
	"required": ["gtin", "facility_id"],
	"properties": {
		"gtin": {
			"type": "string",
			"pattern": "^\\d{14}$"
		},
		"count_only": {
			"type": "boolean"
		},
		"size": {
			"type": "integer"
		},
		"cursor": {
			"type": "string"
		},
		"facility_id": {
			"type": "string"
		},
		"confidence": {
			"type": "number"
		}
	},
	"definitions": {
		"facilities": {
			"type": "array",
			"items": {
				"type": "string"
			},
			"minItems": 1
		}
	},
	"additionalProperties": false
}`
