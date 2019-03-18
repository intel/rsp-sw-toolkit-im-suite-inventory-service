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

package encodingscheme

import (
	"strconv"
)

const (
	EPCPureURIPrefix = "urn:epc:id:sgtin:"
)

type TagDecoder interface {
	Decode(tagData string) (productID, URI string, err error)
	Type() string
}

func GetEpcBytes(epc string) ([numEpcBytes]byte, error) {
	epcBytes := [numEpcBytes]byte{}
	for i := 0; i < len(epcBytes); i++ {
		tempParse, err := strconv.ParseUint(epc[i*2:(i*2)+2], 16, 8)
		if err != nil {
			return epcBytes, err
		}
		epcBytes[i] = byte(tempParse) & 0xFF
	}
	return epcBytes, nil
}

func ZeroFill(data string, num int) string {
	for {
		if len(data) >= num {
			return data[0:num]
		}
		data = "0" + data
	}
}
