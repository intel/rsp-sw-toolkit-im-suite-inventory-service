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

package tag

import (
	"reflect"
	"time"

	"encoding/json"

	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
)

// Tag is the model containing items for a Tag
//swagger:model Tag
type Tag struct {
	// URI string representation of tag
	URI string `json:"uri"`
	// SGTIN EPC code
	Epc string `json:"epc"`
	// ProductID
	ProductID string `json:"product_id" bson:"product_id"`
	// Part of EPC, denotes packaging level of the item
	FilterValue int64 `json:"filter_value" bson:"filter_value"`
	// Tag manufacturer ID
	Tid string `json:"tid"`
	// TBD
	EpcEncodeFormat string `json:"encode_format" bson:"encode_format"`
	// Facility ID
	FacilityID string `json:"facility_id" bson:"facility_id"`
	// Last event recorded for tag
	Event string `json:"event"`
	// Arrival time in milliseconds epoch
	Arrived int64 `json:"arrived"`
	// Tag last read time in milliseconds epoch
	LastRead int64 `json:"last_read" bson:"last_read"`
	// Where tags were read from (fixed or handheld)
	Source string `json:"source"`
	// Array of objects showing history of the tag's location
	LocationHistory []LocationHistory `json:"location_history" bson:"location_history"`
	// Current state of tag, either ’present’ or ’departed’
	EpcState string `json:"epc_state" bson:"epc_state"`
	// Customer defined state
	QualifiedState string `json:"qualified_state" bson:"qualified_state"`
	// Time to live, used for db purging - always in sync with last read
	TTL time.Time `json:"ttl"`
	// Customer defined context
	EpcContext string `json:"epc_context" bson:"epc_context"`
	// Probability item is actually present
	Confidence float64 `json:"confidence,omitempty"` //omitempty - confidence is not stored in the db
	// Cycle Count indicator
	CycleCount bool `json:"-"`
}

// LocationHistory is the model to record the whereabouts history of a tag
type LocationHistory struct {
	Location  string `json:"location"`
	Timestamp int64  `json:"timestamp"`
	Source    string `json:"source"`
}

// IsEqual compares 2 tag structures
// nolint :gocyclo
func (source Tag) IsEqual(target Tag) bool {
	if source.URI == target.URI &&
		source.Epc == target.Epc &&
		source.EpcEncodeFormat == target.EpcEncodeFormat &&
		source.Event == target.Event &&
		source.FacilityID == target.FacilityID &&
		source.Tid == target.Tid &&
		source.Arrived == target.Arrived &&
		source.LastRead == target.LastRead &&
		source.Source == target.Source &&
		reflect.DeepEqual(source.LocationHistory, target.LocationHistory) &&
		source.QualifiedState == target.QualifiedState &&
		source.EpcState == target.EpcState &&
		source.EpcContext == target.EpcContext &&
		source.ProductID == target.ProductID {
		return true
	}
	return false
}

// RequestBody is the model for request body used for many data apis.
type RequestBody struct {
	// User set qualified state for the item
	QualifiedState string `json:"qualified_state"`
	// Return only facilities provided
	FacilityID string `json:"facility_id"`
	// EPC state of ‘present’ or ‘departed’
	EpcState string `json:"epc_state"`
	// Millisecond epoch start time
	StartTime int64 `json:"starttime"`
	// Millisecond epoch stop time
	EndTime int64 `json:"endtime"`
	// Millisecond epoch current time
	Time int64 `json:"time"`
	// Minimum probability items must meet
	Confidence float64 `json:"confidence"`
	// Cursor from previous response used to retrieve next page of results.
	Cursor string `json:"cursor"`
	// Number of results per page
	Size int `json:"size"`
	// Return only tag count
	CountOnly bool `json:"count_only"`
	// GTIN-14 decoded from EPC
	ProductID string `json:"productId"`
	// SGTIN EPC code
	Epc string `json:"epc"`
}

// doc_RequsetBody is the swagger doc model
//swagger:parameters getCurrentInventory
//nolint:deadcode
type doc_RequestBody struct {
	//in:body
	Body RequestBody `json:"datadata"`
}

// PagingType is the model used for paging that is returned in the query response
type PagingType struct {
	Cursor string `json:"cursor,omitempty"`
}

// Response is the model used to return the query response
type Response struct {
	PagingType *PagingType `json:"paging,omitempty"`
	Count      *int        `json:"count,omitempty"`
	Results    interface{} `json:"results"`
}

// CountType is the model for returning only the number of tags of a given query
type CountType struct {
	Count *int `json:"count"`
}

// IsEmpty determines if a tag is empty
func (source Tag) IsEmpty() bool {
	return reflect.DeepEqual(source, Tag{})
}

// IsShippingNoticeEntry is function to determine if a tag in the DB was the
// result of an Advance Shipping Notice. This is needed to attempt to distinguish
// between tags inserted by a tag read versus those that resulted from an ASN.
// NOTE: This DOES NOT determine that a tag *has* a shipping notice -- instead,
// it determines that a tag *only exists because* of a shipping notice.
func (source Tag) IsShippingNoticeEntry() bool {
	if source.EpcContext == "" ||
		source.ProductID != "" ||
		source.FilterValue != 0 ||
		source.Tid != "" ||
		source.EpcEncodeFormat != "" ||
		source.FacilityID != config.AppConfig.AdvancedShippingNoticeFacilityID ||
		source.Event != "" ||
		source.Arrived != 0 ||
		source.LastRead != 0 ||
		source.Source != "" ||
		len(source.LocationHistory) != 0 ||
		source.EpcState != "" ||
		source.QualifiedState != "" {
		return false
	}

	// check if it can be deserialized as ASNData
	var asn ASNContext
	if err := json.Unmarshal([]byte(source.EpcContext), &asn); err != nil {
		// ignore unmarshal errors from this; we don't care
		return false
	}
	// if so, does it all the ASN data?
	return asn.ASNID != "" && asn.EventTime != "" &&
		asn.SiteID != "" && asn.ItemGTIN != "" && asn.ItemID != ""
}

// IsTagReadByRspController returns true if a tag was read by the RSP Controller, versus a result of ASN
func (source Tag) IsTagReadByRspController() bool {
	return !source.IsEmpty() && !source.IsShippingNoticeEntry()
}

// TagStateChange is the model to capture the previous and current state of a tag
// nolint :golint
type TagStateChange struct {
	PreviousState Tag `json:"previousState" `
	CurrentState  Tag `json:"currentState" `
}

// ASNContext represents the data to be marshaled into the EPCContext field for
// an Advanced Shipping Notice for each EPC to which the ASN applies.
type ASNContext struct {
	// ASNID is the ID of the shipment copied from the top level of the ASN that added this EPC.
	ASNID string `json:"asnId"`
	// EventTime is a string provided by the ASN indicating when it was updated.
	EventTime string `json:"eventTime"`
	// SiteID indicates the site to which this ASN applies.
	SiteID string `json:"siteId"`
	// ItemGTIN is a company identifier provided with the original ASN data.
	ItemGTIN string `json:"itemGtin"`
	// ItemID is also a company identifier provided with the original ASN data.
	ItemID string `json:"itemId"`
}

// ASNInputItem is a block of metadata and list of EPCs to which the metadata applies.
type ASNInputItem struct {
	// EPCs to which this ASN applies
	EPCs []string `json:"itemEpcs"`
	// ItemGTIN is a company identifier provided with the original ASN data.
	ItemGTIN string `json:"itemGtin"`
	// ItemID is also a company identifier provided with the original ASN data.
	ItemID string `json:"itemId"`
}

// AdvanceShippingNotice models the information meant to be serialized to the
// EPCContext field for all the EPCs in the provided data list.
type AdvanceShippingNotice struct {
	// ID is the ID of this shipment.
	ID string `json:"asnId"`
	// EventTime is a string provided by the ASN indicating when it was updated.
	EventTime string `json:"eventTime"`
	// SiteID indicates the site to which this ASN applies.
	SiteID string `json:"siteId"`
	// Items is the list of ASNInputItems for this ASN.
	Items []ASNInputItem `json:"items"`
}

// PurgingRequest is the model for request body of the api used for purging the collection periodically
type PurgingRequest struct {
	Days int `json:"days"`
}

const (
	// StateChangeEvent is constant for state change trigger rule
	StateChangeEvent = "stateChange"
	// OutOfStockEvent is constant for out of stock trigger rule
	OutOfStockEvent = "outOfStock"
)
