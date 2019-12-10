/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
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
