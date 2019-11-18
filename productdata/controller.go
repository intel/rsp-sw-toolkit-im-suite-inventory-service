package productdata

import (
	"encoding/json"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"io/ioutil"
	"net/http"
	"time"

	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
	"strconv"
)

func MakeGetCallToSkuMapping(url string) ([]ProdData, error) {
	log.Debugf("Making GET call to: %s", url)
	// Metrics
	metrics.GetOrRegisterMeter(`InventoryService.makePostCall.Attempt`, nil).Mark(1)
	mSuccess := metrics.GetOrRegisterGauge(`InventoryService.makePostCall.Success`, nil)
	mGetErr := metrics.GetOrRegisterGauge(`InventoryService.makePostCall.makePostCall-Error`, nil)
	mStatusErr := metrics.GetOrRegisterGauge(`InventoryService.makePostCall.requestStatusCode-Error`, nil)
	mGetLatency := metrics.GetOrRegisterTimer(`InventoryService.makePostCall.makePostCall-Latency`, nil)

	timeout := time.Duration(config.AppConfig.EndpointConnectionTimedOutSeconds) * time.Second
	client := &http.Client{
		Timeout: timeout,
	}

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		mGetErr.Update(1)
		log.WithFields(log.Fields{
			"Method": "MakeGetCallToSkuMapping",
			"Action": "Make New HTTP GET request",
			"Error":  err.Error(),
		}).Error(err)
		return nil, errors.Wrapf(err, "unable to create a new GET request")
	}

	getTimer := time.Now()
	response, err := client.Do(request)
	if err != nil {
		mGetErr.Update(1)
		log.WithFields(log.Fields{
			"Method": "MakeGetCallToSkuMapping",
			"Action": "Make HTTP GET request",
			"Error":  err.Error(),
		}).Error(err)
		return nil, errors.Wrapf(err, "unable to get description from mapping service")
	}
	defer func() {
		if respErr := response.Body.Close(); respErr != nil {
			log.WithFields(log.Fields{
				"Method": "makeGetCall",
			}).Warning("Failed to close response.")
		}
	}()

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read the response body")
	}

	if response.StatusCode != http.StatusOK {
		mStatusErr.Update(1)
		log.WithFields(log.Fields{
			"Method": "MakeGetCallToSkuMapping",
			"Action": "Response code: " + strconv.Itoa(response.StatusCode),
			"Error":  fmt.Errorf("response code: %d", response.StatusCode),
		}).Error(err)
		return nil, errors.Wrapf(errors.New("execution error"), "StatusCode %d , Response %s",
			response.StatusCode, string(responseData))
	}
	mGetLatency.UpdateSince(getTimer)
	mSuccess.Update(1)

	var result Result
	if unmarshalErr := json.Unmarshal(responseData, &result); unmarshalErr != nil {
		return nil, errors.Wrapf(err, "failed to Unmarshal response data")
	}

	return result.ProdData, nil
}

// CreateProductDataMap builds a map[string] based of array of gtin for search efficiency
func CreateProductDataMap(url string) (map[string]ProductMetadata, error) {

	metrics.GetOrRegisterGauge(`Inventory.CreateProductDataMap.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Inventory.CreateProductDataMap.Success`, nil)
	mFindAllErr := metrics.GetOrRegisterGauge(`Inventory.CreateProductDataMap.FindAll-Error`, nil)
	mFindALlLatency := metrics.GetOrRegisterTimer(`Inventory.CreateProductDataMap.FindAll-Latency`, nil)

	findAllTimer := time.Now()

	productData, err := MakeGetCallToSkuMapping(url)
	if err != nil {
		mFindAllErr.Update(1)
		return nil, err
	}
	mFindALlLatency.Update(time.Since(findAllTimer))

	productDataMap := make(map[string]ProductMetadata)

	for _, item := range productData {
		for _, data := range item.ProductList {
			productDataMap[data.ProductID] = data
		}
	}

	mSuccess.Update(1)
	return productDataMap, nil

}
