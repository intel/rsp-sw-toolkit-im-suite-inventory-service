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

package contraepc

import (
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/tag"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/statemodel"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
)

const (
	// Source is the default source value for a contra-epc
	Source string = "contra-epc"
	// EventType is the default event type for a contra-epc
	EventType string = "arrival"
	// Location is the default location for a contra-epc
	Location string = "contra-epc"
	// EncodeFormat is the default epc encode format for a contra-epc
	EncodeFormat string = "SGTIN-96"
)

// DeleteContraEpcRequest is the object representation for a delete contra-epc request
type DeleteContraEpcRequest struct {
	Epc        string `json:"epc"`
	FacilityID string `json:"facility_id"`
}

// CreateContraEpcRequest is the object representation for a create contra-epc request
type CreateContraEpcRequest struct {
	Data []CreateContraEpcItem `json:"data"`
}

// CreateContraEpcItem is the object representation for each item to
// be created in a create contra-epc request
type CreateContraEpcItem struct {
	Epc            string `json:"epc"`
	FacilityID     string `json:"facility_id"`
	Gtin           string `json:"gtin"`
	QualifiedState string `json:"qualified_state"`
	EpcContext     string `json:"epc_context"`
}

// AsNewTag converts a contra-epc request item to a Tag object
// by passing it through to the inventory state matrix
func (item CreateContraEpcItem) AsNewTag() tag.Tag {
	tagEvent := tag.TagEvent{
		EpcCode:         item.Epc,
		FacilityID:      item.FacilityID,
		EpcEncodeFormat: EncodeFormat,
		EventType:       EventType,
		Location:        Location,
		Tid:             "",
		Timestamp:       helper.UnixMilliNow(),
	}

	// Go through the state model
	t := statemodel.UpdateTag(tag.Tag{}, tagEvent, Source)
	if item.QualifiedState != "" {
		t.QualifiedState = item.QualifiedState
	}
	if item.EpcContext != "" {
		t.EpcContext = item.EpcContext
	}
	return t
}

// IsContraEpc returns true if tag is a contra-epc
func IsContraEpc(t tag.Tag) bool {
	return len(t.LocationHistory) == 1 &&
		t.LocationHistory[0].Location == Location
}
