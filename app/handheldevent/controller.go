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

package handheldevent

import (
	"net/url"
	"reflect"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
	db "github.impcloud.net/RSP-Inventory-Suite/go-dbWrapper"
	odata "github.impcloud.net/RSP-Inventory-Suite/go-odata/mongo"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/web"
)

const collection = "handheldevents"

// Retrieve retrieves All handheld events from database
//nolint:dupl
func Retrieve(dbs *db.DB, query url.Values) (interface{}, *CountType, error) {
	var object []interface{}

	count := query["$count"]

	// If count is true, return count number
	if len(count) > 0 && len(query) < 2 {

		var count int
		var err error

		execFunc := func(collection *mgo.Collection) (int, error) {
			return odata.ODataCount(collection)
		}

		if count, err = dbs.ExecuteCount(collection, execFunc); err != nil {
			return nil, nil, errors.Wrap(err, "db."+collection+".Count()")
		}

		return nil, &CountType{Count: &count}, nil
	}

	// Else, run filter query and return slice of Facilities
	execFunc := func(collection *mgo.Collection) error {
		return odata.ODataQuery(query, &object, collection)
	}

	if err := dbs.Execute(collection, execFunc); err != nil {
		if errors.Cause(err) == odata.ErrInvalidInput {
			return nil, nil, errors.Wrap(web.ErrInvalidInput, err.Error())
		}
		return nil, nil, errors.Wrap(err, "db."+collection+".find()")
	}

	// Check if inlinecount is set
	inlineCount := query["$inlinecount"]
	var inCount int
	if len(inlineCount) > 0 {
		if inlineCount[0] == "allpages" {
			resultSlice := reflect.ValueOf(object)
			inCount = resultSlice.Len()
			return object, &CountType{Count: &inCount}, nil
		}
	}

	return object, nil, nil

}

// Insert to insert handheldEvent into database
func Insert(dbs *db.DB, handheldEvent HandheldEvent) error {

	handheldEvent.TTL = time.Now()
	execFunc := func(collection *mgo.Collection) error {
		return collection.Insert(handheldEvent)
	}

	if err := dbs.Execute(collection, execFunc); err != nil {
		return errors.Wrap(err, "db."+collection+".insert()")
	}

	return nil
}

// UpdateTTLIndexForHandheldEvents updates the expireAfterSeconds value in ttl index
// nolint :dupl
func UpdateTTLIndexForHandheldEvents(dbs *db.DB, purgingSeconds int) error {

	updateCommand := bson.D{{"collMod", collection}, {"index", bson.D{{"keyPattern", bson.D{{"ttl", 1}}}, {"expireAfterSeconds", purgingSeconds}}}}
	var result interface{}

	execFunc := func(collection *mgo.Collection) error {
		return collection.Database.Run(updateCommand, result)
	}

	return dbs.Execute(collection, execFunc)
}
