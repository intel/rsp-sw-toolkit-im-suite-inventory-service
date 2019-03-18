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
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/encodingscheme"
	"testing"

	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
)

const (
	epcLength    = 24
	maxPartition = 5
)

var (
	// These all need to be valid gtin14s, which means the last digit is a valid checksum
	validGtin14s = [...]string{
		"11234567890842",
		"00052177002189",
		"00888446100818",
		"00000000000000",
		"99999999999997",
		"17373737373731",
		"85784784584574",
		"00039307597746",
		"00039345597746",
	}
)

func TestGenerateEPC(t *testing.T) {
	// Test each gtin with each possible partition value
	for i := 0; i <= maxPartition; i++ {
		config.AppConfig.ContraEpcPartition = i

		for _, gtin14 := range validGtin14s {
			epc, err := GenerateContraEPC(gtin14)
			if err != nil {
				t.Errorf("error: %s", err.Error())
				break
			}
			if len(epc) != epcLength {
				t.Errorf("Incorrect epc length: %d", len(epc))
				break
			}
			g, err := encodingscheme.GetGtin14(epc)
			if err != nil {
				t.Errorf("Error converting contra-epc back to gtin14: %s", err.Error())
			} else if g != gtin14 {
				t.Errorf("Mismatch converting contra-epc back to gtin14 -- expected: %s, but got: %s", gtin14, g)
			}
		}

		config.AppConfig.ContraEpcPartition = 5
	}
}

func TestGenerateEPCInvalidPartition(t *testing.T) {
	config.AppConfig.ContraEpcPartition = 100
	_, err := GenerateContraEPC(validGtin14s[0])
	if err == nil {
		t.Errorf("Failed to catch invalid partition value")
	}
	config.AppConfig.ContraEpcPartition = 5
}

func TestGenerateBadGtin14Length(t *testing.T) {
	_, err := GenerateContraEPC("123")
	if err == nil {
		t.Fatalf("Failed to catch invalid gtin14 length")
	}
}

func TestGenerateBadGtin14Digits(t *testing.T) {
	_, err := GenerateContraEPC("123abc")
	if err == nil {
		t.Fatalf("Failed to catch invalid gtin14 digits")
	}
}
