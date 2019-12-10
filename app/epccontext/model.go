/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package epccontext

// PutBody is the struct for the request body to create/update epc context for a given epc
type PutBody struct {
	Epc        string `json:"epc"`
	EpcContext string `json:"epc_context"`
	FacilityID string `json:"facility_id"`
}

// DeleteBody is the struct for the request body to delete epc context for a given epc
type DeleteBody struct {
	Epc        string `json:"epc"`
	FacilityID string `json:"facility_id"`
}
