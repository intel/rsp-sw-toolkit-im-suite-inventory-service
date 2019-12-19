/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package productdata

import (
	"encoding/json"
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/app/config"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNoDataSkuMapping(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.EscapedPath() != "/skus" {
			t.Errorf("Expected request to be '/skus', received %s",
				request.URL.EscapedPath())
		}
		if request.Method != "GET" {
			t.Errorf("Expected 'GET' request, received '%s", request.Method)
		}
		var jsonData []byte
		if request.URL.EscapedPath() == "/skus" {

			var result = Result{}

			jsonData, _ = json.Marshal(result)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(jsonData)
	}))

	defer testServer.Close()

	data, err := MakeGetCallToSkuMapping(testServer.URL + "/skus")
	if len(data) > 0 {
		t.Fatalf(" %v", err)
	}
}

func TestSkuMapping_TimeOut(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timeout test")
	}
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(time.Duration(config.AppConfig.EndpointConnectionTimedOutSeconds)*time.Second + 10*time.Millisecond)
	}))

	defer testServer.Close()

	config.AppConfig.EndpointConnectionTimedOutSeconds = 1
	_, err := MakeGetCallToSkuMapping(testServer.URL + "/skus")

	if err == nil {
		t.Error("Expecting timeout")
	}
}

func TestCreateProductDataMap(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.EscapedPath() != "/skus" {
			t.Errorf("Expected request to be '/skus', received %s",
				request.URL.EscapedPath())
		}
		if request.Method != "GET" {
			t.Errorf("Expected 'GET' request, received '%s", request.Method)
		}
		var jsonData []byte
		if request.URL.EscapedPath() == "/skus" {
			productIDMetadata := ProductMetadata{
				ProductID:        "00111111",
				BecomingReadable: 0.2,
				BeingRead:        0.75,
				DailyTurn:        0.2,
				ExitError:        0.1,
			}

			productIDList := []ProductMetadata{productIDMetadata}

			var data = ProdData{
				Sku:         "00111111",
				ProductList: productIDList,
			}

			dataList := []ProdData{data}

			var result = Result{
				ProdData: dataList,
			}

			jsonData, _ = json.Marshal(result)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(jsonData)
	}))

	defer testServer.Close()

	dataMap, err := CreateProductDataMap(testServer.URL + "/skus")
	if err != nil {
		t.Errorf("Error creating product data map: %s", err)
	}
	metaData := dataMap["00111111"]
	if metaData.ProductID != "00111111" {
		t.Errorf("Error creating product data map: Expected %s", "00111111")
	}
}
