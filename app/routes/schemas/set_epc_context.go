/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package schemas

// SetEpcContextSchema defines the body which sets the epc context for the specific tag
const SetEpcContextSchema = `{
  "type": "object",
  "required": [
    "epc", "facility_id", "epc_context"
  ],
  "properties": {
    "epc": {
      "type": "string",
      "pattern": "^[a-fA-F0-9]{1,}$"
    },
    "facility_id": {
      "type": "string"
    },
    "epc_context": {
      "type": "string"
    }
  },
  "additionalProperties": false
}`
