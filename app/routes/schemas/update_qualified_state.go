/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package schemas

// UpdateQualifiedStateSchema required for request body validation
const UpdateQualifiedStateSchema = `{
	"type": "object",
	"required": ["epc", "facility_id", "qualified_state"],
	"properties": {
		"epc": {
			"type": "string",
			"pattern": "^[a-fA-F0-9]{1,}$"
		},
		"facility_id": {
			"type": "string"
		},
		"qualified_state": {
			"type": "string",
			"pattern": "^[-a-zA-Z0-9_ ]{1,}$"
		}
	},
	"additionalProperties": false
}`
