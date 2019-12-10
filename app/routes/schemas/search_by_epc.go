/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package schemas

// SearchByEpcSchema defines the request body for searchbyepc endpoint
// the pattern only allows at most one '*'
const SearchByEpcSchema = `{
	"type": "object",
	"required": ["epc", "facility_id"],
	"properties": {
		"epc": {
			"type": "string",
			"pattern": "^(?:[a-fA-F0-9]+\\*?[a-fA-F0-9]*|[a-fA-F0-9]*\\*?[a-fA-F0-9]+|\\*)$"
		},
		"size": {
			"type": "integer"
		},
		"facility_id": {
			"type": "string"
		},
		"cursor": {
			"type": "string"
		},
		"confidence": {
			"type": "number"
		}
	},
	"additionalProperties": false
}`
