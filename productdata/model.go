/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package productdata

// Result
type Result struct {
	ProdData []ProdData `json:"results"`
}

// ProdData represents the product data schema in the database
type ProdData struct {
	Sku         string            `json:"sku"`
	ProductList []ProductMetadata `json:"productList"`
}

// ProductMetadata represents the ProductList schema attribute in the database
type ProductMetadata struct {
	ProductID        string                 `json:"productId"`
	BeingRead        float64                `json:"beingRead"`
	BecomingReadable float64                `json:"becomingReadable"`
	ExitError        float64                `json:"exitError"`
	DailyTurn        float64                `json:"dailyTurn"`
	Metadata         map[string]interface{} `json:"metadata"`
}
