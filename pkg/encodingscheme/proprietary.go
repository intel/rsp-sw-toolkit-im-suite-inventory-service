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
	"fmt"
	"github.com/pkg/errors"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/slices"
	"math/big"
	"strconv"
	"strings"
	"time"
)

const (
	// ProductID is the label used for the product id field
	ProductID = "productID"
	// tagAuthorityReferenceYear is used to parse tag URI authority dates
	tagAuthorityReferenceYear = "2006-01-02"
)

type ProprietaryExtractor struct {
	// RFC-4151: authorityName + "," + date
	taggingEntity string
	fields        []string
	widths        []int
	productIdx    int
	bitLength     int
}

func (pe *ProprietaryExtractor) Type() string {
	return "ProprietaryTag"
}

// NewProprietary returns a new ProprietaryExtractor based on configuration
// strings for the field names and bit widths. In both cases, values should be
// separated with "."; surrounding whitespace is stripped.
//
// This validates the following:
// - there are the same number of items in fields and widths
// - all widths are integers >= 1
// - no fields are empty or duplicated
// - one field is named "productID"
func NewProprietary(authority, date, fields, widths string) (TagDecoder, error) {
	pe := ProprietaryExtractor{productIdx: -1}
	if err := pe.setTaggingEntity(authority, date); err != nil {
		return nil, err
	}

	splitFields := strings.Split(fields, ".")
	splitWidths := strings.Split(widths, ".")
	if len(splitFields) != len(splitWidths) {
		return nil, errors.Errorf("fields and widths should have the same length, "+
			"but there are %d fields and %d widths", len(splitFields), len(splitWidths))
	}

	for i := 0; i < len(splitFields); i++ {
		if err := pe.addField(strings.TrimSpace(splitFields[i])); err != nil {
			return nil, errors.Wrapf(err, "bad field at index %d", i)
		}

		if err := pe.addWidth(strings.TrimSpace(splitWidths[i])); err != nil {
			return nil, errors.Wrapf(err, "bad width at index %d", i)
		}
	}

	if pe.productIdx == -1 {
		return nil, errors.Errorf("missing field %s", ProductID)
	}

	return &pe, nil
}

// Decode decodes the given tag data, expected as a hex string and returns a URI
// representing this tag, along with its extracted productID field. If decoding
// fails, this returns an error, and productID and URI are unspecified.
//
// This uri scheme is derived from the Tag URI defined by www.taguri.org and
// published as RFC 4151. The general syntax of a tag URI, in ABNF [2], is:
// tagURI = "tag:" taggingEntity ":" specific [ "#" fragment ]
// taggingEntity = authorityName "," date
// authorityName = DNSname / emailAddress
// date = year ["-" month ["-" day]]
func (pe *ProprietaryExtractor) Decode(data string) (productID, URI string, err error) {
	if len(data)*4 != pe.bitLength {
		err = errors.Errorf("invalid data length %d; expected %d bits",
			len(data)*4, pe.bitLength)
	}

	bigInt := new(big.Int)
	if _, ok := bigInt.SetString(data, 16); !ok {
		err = errors.New("unable to decode tag data as hex")
		return
	}
	bitStr := fmt.Sprintf("%0[1]*b", pe.bitLength, bigInt)

	var bitIdx = 0
	base10Fields := make([]string, len(pe.fields))
	for fieldIdx, width := range pe.widths {
		bits := bitStr[bitIdx:(bitIdx + width)]
		if width <= 64 {
			// common-case, faster path for value that fit in a 8-byte word
			decVal, convErr := strconv.ParseUint(bits, 2, 64)
			if convErr != nil {
				err = errors.Wrap(convErr, "unable to convert binary extraction to decimal")
				return
			}
			base10Fields[fieldIdx] = strconv.FormatUint(decVal, 10)

			if fieldIdx == pe.productIdx {
				productID = fmt.Sprintf("%X", decVal)
			}
		} else {
			// slower case to handle more than 64 bits
			if _, ok := bigInt.SetString(bits, 2); !ok {
				err = errors.New("unable to convert binary extraction to decimal")
				return
			}

			base10Fields[fieldIdx] = fmt.Sprintf("%d", bigInt)
			if fieldIdx == pe.productIdx {
				hexLen := ((width - 1) / 4) + 1
				productID = fmt.Sprintf("%0[1]*X", hexLen, bigInt)
			}
		}
		bitIdx += width
	}

	// the tag URI's specific part is simply the decimal fields, joined with "."
	URI = fmt.Sprintf("tag:%s:%s", pe.taggingEntity, strings.Join(base10Fields, "."))
	return
}

func (pe *ProprietaryExtractor) setTaggingEntity(authority string, date string) error {
	if authority == "" || date == "" {
		return errors.New("authority and date must be set")
	}

	// TODO: validate URI

	_, err := time.Parse(tagAuthorityReferenceYear, date)
	if err != nil {
		return errors.Wrapf(err, "invalid authority date")
	}

	pe.taggingEntity = authority + "," + date
	return nil
}

func (pe *ProprietaryExtractor) addField(f string) error {
	if f == "" {
		return errors.Errorf("found an empty field")
	}

	if slices.Contains(pe.fields, f) {
		return errors.Errorf("found a duplicate field: '%s'", f)
	}

	pe.fields = append(pe.fields, f)

	if f == ProductID {
		pe.productIdx = len(pe.fields) - 1
	}

	return nil
}

func (pe *ProprietaryExtractor) addWidth(w string) error {
	if w == "" {
		return errors.New("empty bit width")
	}

	width, err := strconv.Atoi(w)
	if err != nil {
		return errors.Wrapf(err, "bit width '%s' is not an int", w)
	}

	if width <= 0 {
		return errors.Errorf("illegal bit width is %d: must be >0", width)
	}

	pe.widths = append(pe.widths, width)
	pe.bitLength += width
	return nil
}
