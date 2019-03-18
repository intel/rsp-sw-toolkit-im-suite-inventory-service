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
	"math"
	"time"

	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
)

const millisecondsInDay = int64(24 * time.Hour / time.Millisecond)

// CalculateConfidence calculates the confidence for each and all the tags when sending for a query
func CalculateConfidence(dailyInventoryPercentage float64,
	probUnreadToRead float64, probInStoreRead float64,
	probExitError float64, lastRead int64, isContraEpc bool) float64 {

	days := daysFromLastRead(lastRead)
	inStore := math.Pow((1 - dailyInventoryPercentage), days)
	notRead := (1 - probInStoreRead) * math.Pow((1-probUnreadToRead), days)
	outOfStore := 1 - inStore
	confidence := (inStore * notRead) / ((inStore * notRead) + probExitError*outOfStore)

	if isContraEpc {
		return setPrecision(confidence) * -1
	}

	return setPrecision(confidence)
}

func daysFromLastRead(lastRead int64) float64 {
	return float64(helper.UnixMilliNow()-lastRead) / float64(millisecondsInDay)
}

func setPrecision(value float64) float64 {
	var round float64
	places := 3
	power := math.Pow(10, float64(places))
	digits := power * value
	_, remainder := math.Modf(digits)
	if remainder >= 0.5 {
		round = math.Ceil(digits)
	} else {
		round = math.Floor(digits)
	}
	newValue := round / power
	return newValue
}
