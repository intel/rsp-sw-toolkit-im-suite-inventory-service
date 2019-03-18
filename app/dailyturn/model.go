/*
 * INTEL CONFIDENTIAL
 * Copyright (2018) Intel Corporation.
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

package dailyturn

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"math"
	"sort"
	"time"
)

const millisecondsInDay = float64(24 * time.Hour / time.Millisecond)

var (
	// ErrTimeTooShort is when the time between two ASNs is not long enough to compute the daily turn
	ErrTimeTooShort = errors.New("time between asn is too short, daily turn will not be computed")
	// ErrNoInventory is when an ASN comes in but there is no existing inventory for that product
	ErrNoInventory = errors.New("no inventory found for product, daily turn will not be computed")
)

// History is the model of the history of daily turn computations for a product
type History struct { //nolint :golint
	ProductID string   `json:"product_id" bson:"product_id"`
	DailyTurn float64  `json:"daily_turn" bson:"daily_turn"`
	Records   []Record `json:"records" bson:"records"`
	Timestamp int64    `json:"last_timestamp" bson:"last_timestamp"`
}

// Record is the model for each daily turn data point
type Record struct { //nolint :golint
	Present           int     `json:"present" bson:"present"`
	Departed          int     `json:"departed" bson:"departed"`
	DailyTurn         float64 `json:"daily_turn" bson:"daily_turn"`
	PreviousTimestamp int64   `json:"previous_timestamp" bson:"previous_timestamp"`
	Timestamp         int64   `json:"timestamp" bson:"timestamp"`
}

func (record *Record) ComputeDailyTurn() error {
	log.Debugf("Compute Daily Turn: %d", record.Timestamp)

	if record.Present+record.Departed == 0 {
		return ErrNoInventory
	}

	daysSinceLastTimestamp := float64(record.Timestamp-record.PreviousTimestamp) / millisecondsInDay
	if daysSinceLastTimestamp < 1.0 {
		return ErrTimeTooShort
	}

	record.DailyTurn = math.Abs(float64(record.Departed) / float64(record.Present+record.Departed) / daysSinceLastTimestamp)
	return nil
}

func (history *History) ComputeAverageDailyTurn() {
	if len(history.Records) < config.AppConfig.DailyTurnMinimumDataPoints {
		history.DailyTurn = 0
		return
	}

	if config.AppConfig.DailyTurnComputeUsingMedian {
		history.DailyTurn = computeMedian(history.Records)
	} else {
		history.DailyTurn = computeMean(history.Records)
	}
}

func computeMedian(records []Record) float64 {
	// make a copy to avoid modifying input data
	copyRecords := append([]Record(nil), records...)
	sort.Slice(copyRecords, func(i, j int) bool { return copyRecords[i].DailyTurn < copyRecords[j].DailyTurn })
	middle := len(copyRecords) / 2

	if len(copyRecords)%2 == 0 {
		return (copyRecords[middle-1].DailyTurn + copyRecords[middle].DailyTurn) / 2.0
	}

	return copyRecords[middle].DailyTurn
}

func computeMean(records []Record) float64 {
	var total float64
	for _, record := range records {
		total += record.DailyTurn
	}
	return total / float64(len(records))
}
