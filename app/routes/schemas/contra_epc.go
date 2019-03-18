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

package schemas

// DeleteContraEpcSchema is the json-schema for DeleteContraEpc requests
const DeleteContraEpcSchema = `{
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

// CreateContraEpcSchema is the json-schema for CreateContraEpc requests
const CreateContraEpcSchema = `{
	"type": "object",
	"required": [
		"data"
	],
	"properties": {
		"data": {
			"type": "array",
			"minItems": 1,
			"maxItems": 1000,
			"items": {
				"type": "object",
				"oneOf": [{
						"required": [
							"facility_id",
							"gtin"
						],
						"properties": {
							"facility_id": {
								"type": "string"
							},
							"gtin": {
								"type": "string",
								"pattern": "^\\d{14}$"
							},
							"qualified_state": {
								"type": "string",
								"pattern": "^[-a-zA-Z0-9_ ]{1,}$"
							},
							"epc_context": {
								"type": "string"
							}
						},
						"additionalProperties": false
					},
					{
						"required": [
							"facility_id",
							"epc"
						],
						"properties": {
							"facility_id": {
								"type": "string"
							},
							"epc": {
								"type": "string",
								"pattern": "^[a-fA-F0-9]{1,}$"
							},
							"qualified_state": {
								"type": "string",
								"pattern": "^[-a-zA-Z0-9_ ]{1,}$"
							},
							"epc_context": {
								"type": "string"
							}
						},
						"additionalProperties": false
					}
				]
			}
		}
	},
	"additionalProperties": false
}`
