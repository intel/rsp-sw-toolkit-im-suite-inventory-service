/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
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
