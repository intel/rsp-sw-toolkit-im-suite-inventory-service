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
	"github.impcloud.net/RSP-Inventory-Suite/expect"
	"testing"
)

func TestGetProprietaryURI(t *testing.T) {
	w := expect.WrapT(t)

	decoder := w.ShouldHaveResult(
		NewProprietary("test.com", "2019-01-01",
			"header.serialNumber.productID", "8.48.40")).(TagDecoder)

	productID, URI, err := decoder.Decode("0F00000000000C00000014D2")

	w.As("decoding").ShouldSucceed(err)
	w.As("productID").ShouldBeEqual(productID, "14D2")
	w.As("URI").ShouldBeEqual(URI, "tag:test.com,2019-01-01:15.12.5330")
}

func TestGetProprietaryURI_invalidBitBoundary(t *testing.T) {
	w := expect.WrapT(t)

	invalidWidths := []string{
		"8..40",
		"",
		"  ",
		"8.88.",
	}
	for _, widths := range invalidWidths {
		w.As(widths).ShouldHaveError(
			NewProprietary("test.com", "2019-01-01",
				"header.serialNumber.productID", widths))
	}
}
