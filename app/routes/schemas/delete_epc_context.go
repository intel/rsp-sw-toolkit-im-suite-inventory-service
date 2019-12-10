/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package schemas

// DeleteEpcContextSchema defines the body which deletes the epc context for the specific tag
const DeleteEpcContextSchema = `{
  "type": "object",
  "required": [
    "epc", "facility_id"
  ],
  "properties": {
    "epc": {
      "type": "string",
      "pattern": "^[a-fA-F0-9]{1,}$"
    },
    "facility_id": {
      "type": "string"
    }
  },
  "additionalProperties": false
}`
