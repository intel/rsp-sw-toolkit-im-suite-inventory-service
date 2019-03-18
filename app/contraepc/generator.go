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
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
)

/*
 REFERENCES
 http://www.epc-rfid.info/sgtin
 http://www.epc-rfid.info/epc-binary-headers
 http://www.epc-rfid.info/sgtin-filter-values
 https://www.gs1.at/fileadmin/user_upload/RFIDBarcodeInterop-Guideline-i1-final-Publication.pdf
*/
const (
	gtin14Length          = 14
	binaryBase            = 2
	decimalBase           = 10
	hexBase               = 16
	header                = "00110000" // Specifies tag is using SGTIN-96 encoding
	filter                = "101"      // Reserved value that is being used for Contra EPC
	partitionBits         = 3
	serialNumberBits      = 38
	combinedReferenceBits = 44 // Company Prefix Bits + Item Reference Bits

	// MaxTries is the number of times to attempt to generate a unique contra-epc before giving up
	MaxTries = 25
)

type partitionTableItem struct {
	bits   int
	digits int
}

var (
	isValidGtin14 = regexp.MustCompile(`^[0-9]{14}$`).MatchString

	// Reference: http://www.epc-rfid.info/sgtin-partition-values
	companyPrefixMetadataPartitionTable = map[int]partitionTableItem{
		0: {40, 12},
		1: {37, 11},
		2: {34, 10},
		3: {30, 9},
		4: {27, 8},
		5: {24, 7},
		6: {20, 6},
	}
)

// GenerateContraEPC takes a valid gtin14 value and generates an SGTIN-96 EPC code
func GenerateContraEPC(gtin14 string) (string, error) {
	companyPrefixMetadata, ok := companyPrefixMetadataPartitionTable[config.AppConfig.ContraEpcPartition]
	if !ok {
		return "", errors.New("invalid partition value")
	}
	if !isValidGtin14(gtin14) {
		return "", errors.New("invalid gtin value, unable to generate contra-epc")
	}

	companyPrefixInt, err := getCompanyPrefix(gtin14, companyPrefixMetadata.digits)
	if err != nil {
		return "", errors.Wrap(err, "unable to parse company prefix")
	}
	companyPrefix := padZerosLeft(toBinaryString(companyPrefixInt), companyPrefixMetadata.bits)

	itemReferenceInt, err := getItemReference(gtin14, companyPrefixMetadata.digits)
	if err != nil {
		return "", errors.Wrap(err, "unable to parse item reference")
	}
	itemReference := padZerosLeft(toBinaryString(itemReferenceInt),
		getItemReferenceBits(companyPrefixMetadata.bits))

	partitionBinary := padZerosLeft(toBinaryString(int64(config.AppConfig.ContraEpcPartition)), partitionBits)
	serialNumber, err := generateSerialNumber()
	if err != nil {
		return "", err
	}

	epcInBinary := header + filter + partitionBinary + companyPrefix + itemReference + serialNumber
	return binaryToHexString(epcInBinary), nil
}

func getCompanyPrefix(gtin14 string, companyPrefixDigits int) (int64, error) {
	prefix := gtin14[1 : companyPrefixDigits+1]
	return strconv.ParseInt(prefix, decimalBase, 64)
}

func getItemReference(gtin14 string, companyPrefixDigits int) (int64, error) {
	itemRef := gtin14[:1] + gtin14[companyPrefixDigits+1:gtin14Length-1]
	return strconv.ParseInt(itemRef, decimalBase, 64)
}

func getItemReferenceBits(companyPrefixBits int) int {
	return combinedReferenceBits - companyPrefixBits
}

func padZerosLeft(value string, length int) string {
	return fmt.Sprintf("%0"+strconv.Itoa(length)+"s", value)
}

// generateSerialNumber generates a random binary string serialNumberBits in length
func generateSerialNumber() (string, error) {
	// (1 << serialNumberBits) is equal to the exclusive maximum decimal integer that can fit into
	// a binary string with length of 'serialNumberBits'. i.e. (1 << 4) - 1 == "0b1111" aka 4 1's
	maxNum := big.NewInt(1 << uint(serialNumberBits))
	n, err := rand.Int(rand.Reader, maxNum)
	if err != nil {
		return "", err
	}
	// We still need to pad with zeroes as the random number may not fill all bits
	return padZerosLeft(toBinaryString(n.Int64()), serialNumberBits), nil
}

// binaryToHexString converts a binary string to a hex string
func binaryToHexString(binary string) string {
	i := new(big.Int)
	i.SetString(binary, binaryBase)
	return strings.ToUpper(i.Text(hexBase))
}

// toBinaryString converts an int64 to a binary string
func toBinaryString(i int64) string {
	return strconv.FormatInt(i, binaryBase)
}
