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

package config

import (
	"testing"
)

func TestParseAgeOuts(t *testing.T) {
	ageOutString := "front:60,back:30,another:45,one_more:12"
	expectedMap := map[string]int{
		"front":    60,
		"back":     30,
		"another":  45,
		"one_more": 12,
	}

	result, err := parseAgeOuts(ageOutString)
	if err != nil {
		t.Fatalf("Unexpected error during parsing: %s", err.Error())
	}

	if len(result) != len(expectedMap) {
		t.Errorf("Expected length of result: %d. Actual: %d",
			len(expectedMap), len(result))
	}

	for facility, expectedTime := range expectedMap {
		actualTime, ok := result[facility]
		if !ok {
			t.Errorf("Expected %s in result, but it was not there", facility)
		} else if actualTime != expectedTime {
			t.Errorf("Time for %s does not match. Expected: %d. Actual: %d",
				facility, expectedTime, actualTime)
		}
	}
}

func TestParseAgeOutNonInt(t *testing.T) {
	ageOutString := "front:60,back:asdf"

	_, err := parseAgeOuts(ageOutString)
	if err == nil {
		t.Fatal("Failed to catch non-int error")
	}
}

func TestParseAgeOutNoTime(t *testing.T) {
	ageOutString := "front:60,back: "

	_, err := parseAgeOuts(ageOutString)
	if err == nil {
		t.Fatal("Failed to catch missing time error")
	}
}
func TestParseAgeNoFacility(t *testing.T) {
	ageOutString := "front:60,:15"

	_, err := parseAgeOuts(ageOutString)
	if err == nil {
		t.Fatal("Failed to catch missing facility error")
	}
}
func TestParseAgeEmptyString(t *testing.T) {
	ageOutString := ""

	results, err := parseAgeOuts(ageOutString)
	if err != nil {
		t.Fatalf("Unexpected error during parsing: %s", err.Error())
	}

	if len(results) != 0 {
		t.Errorf("Did not expect any items after parsing, but got back %d",
			len(results))
	}
}

func TestParseAgeOnlyOnePart(t *testing.T) {
	ageOutString := "front:60,facility15"

	_, err := parseAgeOuts(ageOutString)
	if err == nil {
		t.Fatal("Failed to catch parse error")
	}
}

func TestParseAgeTrailingComma(t *testing.T) {
	ageOutString := "front:60,"

	_, err := parseAgeOuts(ageOutString)
	if err == nil {
		t.Fatal("Failed to catch parse error")
	}
}

func TestParseAgeNegativeValue(t *testing.T) {
	ageOutString := "front:-60,back:60"

	_, err := parseAgeOuts(ageOutString)
	if err == nil {
		t.Fatal("Failed to catch parse error")
	}
}
