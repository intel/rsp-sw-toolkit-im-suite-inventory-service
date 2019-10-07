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
	"github.com/pkg/errors"
	"github.impcloud.net/RSP-Inventory-Suite/tagcode/bitextract"
	"github.impcloud.net/RSP-Inventory-Suite/tagcode/bittag"
	"github.impcloud.net/RSP-Inventory-Suite/tagcode/epc"
)

type TagDecoder interface {
	Decode(tagData []byte) (productID, URI string, err error)
}

type sgtinDecoder struct {
	strict bool
}

// NewSGTINDecoder returns a new tag decoder for SGTIN encoded tags.
//
// If strict is true, after successful decoding, the decoder checks that the
// tag's ranges fit within the sizes available according to the EPC standard.
//
// The URI returned by the decoder is the EPC Pure Identity URI. The productID
// is the SGTIN's GTIN-14 representation.
func NewSGTINDecoder(strict bool) TagDecoder {
	return &sgtinDecoder{strict: strict}
}

func (d *sgtinDecoder) Decode(tagData []byte) (productID, URI string, err error) {
	var s epc.SGTIN
	s, err = epc.DecodeSGTIN(tagData)
	if d.strict && err == nil {
		err = s.ValidateRanges()
	}
	if err == nil {
		URI = s.URI()
		productID = s.GTIN()
	}
	return
}

type proprietary struct {
	bittag.Decoder
	prodIdx, prodHexWidth int
}

// ProductID is the label used for the product id field
const ProductID = "productID"

// NewProprietary returns a decoder for proprietary tag encodings, i.e., those
// which are simply delimited by field widths. All tags with at least as much
// data as the bittag's size will be decoded successfully.
//
// The URI returned by the decoder is the same as the bittag URI: "." delimited
// base-10 encoded representations of the individual fields, prepended by the
// authority,date string.
//
// The productID returned by the decoder is the hex presentation of the prodIdx
// field, zero-padded to the closest nibble width matching the field width.
func NewProprietary(authority, date, fieldWidthsStr string, prodIdx int) (TagDecoder, error) {
	fieldWidths, err := bitextract.SplitWidths(fieldWidthsStr, ".")
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse field widths")
	}
	btd, err := bittag.NewDecoder(authority, date, fieldWidths)
	if err != nil {
		return nil, err
	}
	if prodIdx < 0 || prodIdx >= btd.NumFields() {
		return nil, errors.Errorf("field index %d out of range [0, %d]",
			prodIdx, btd.NumFields())
	}

	prodHexWidth := ((fieldWidths[prodIdx] - 1) / 4) + 1
	return &proprietary{Decoder: btd, prodIdx: prodIdx, prodHexWidth: prodHexWidth}, nil
}

func (d *proprietary) Decode(tagData []byte) (productID, URI string, err error) {
	var bt bittag.BitTag
	bt, err = d.Decoder.Decode(tagData)
	if err != nil {
		return
	}
	URI = bt.URI()
	productID = bt.HexField(d.prodIdx, d.prodHexWidth)
	return
}
