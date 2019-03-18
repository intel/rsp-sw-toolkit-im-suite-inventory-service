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
	"fmt"
	"testing"
	"time"

	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
)

type inputTest struct {
	lastRead           int64
	contra             bool
	expectedConfidence float64
}

func TestCalculateConfidence(t *testing.T) {

	var testCases = []inputTest{
		// Contra false
		{
			lastRead:           helper.UnixMilliNow(),
			contra:             false,
			expectedConfidence: 1,
		},
		// Contra true
		{
			lastRead:           helper.UnixMilliNow(),
			contra:             true,
			expectedConfidence: -1,
		},
		// Longtime, contra true
		{
			lastRead:           int64(0),
			contra:             true,
			expectedConfidence: 0,
		},
		// Longtime, contra false
		{
			lastRead:           int64(0),
			contra:             false,
			expectedConfidence: 0,
		},
	}

	for _, test := range testCases {
		confidence := CalculateConfidence(0.01, 0.20, 0.75, 0.10, test.lastRead, test.contra)
		// Testing with current timestamp, confidence should be 1
		if confidence != test.expectedConfidence {
			t.Fatalf("coeficient calculation wrong. Expected %f, calculated %f", test.expectedConfidence, confidence)
		}

	}

}

func TestCalculateHighishConfidence(t *testing.T) {
	timestamp := helper.UnixMilliNow() - int64(8*time.Hour/time.Millisecond)
	coeficient := CalculateConfidence(0.01, 0.20, 0.75, 0.10, timestamp, false)

	fmt.Println(coeficient)

	if coeficient == 1.0 {
		t.Errorf("wrong value %v", coeficient)
	}
}
