/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package schemas

// PurgingSchema gets the json schema to update purging days
const PurgingSchema = `{
	 "type": "object",
	 "required": [
		 "days"
	 ],
	 "properties": {	
		 "days": {
			 "type": "integer"
		 }
	 },
	 "additionalProperties": false
 }`
