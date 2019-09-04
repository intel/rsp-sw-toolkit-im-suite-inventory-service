package jsonrpc

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
	"strings"
)

func errorHandler(message string, err error, errorGauge *metrics.Gauge) {
	if err != nil {
		if errorGauge != nil {
			(*errorGauge).Update(1)
		}
		logrus.WithFields(logrus.Fields{
			"Method": "jsonrpc.Decode",
			"Error":  fmt.Sprintf("%+v", err),
		}).Error(message)
	}
}

func Decode(value string, js Message, errorGauge *metrics.Gauge) error {
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.UseNumber()

	if err := decoder.Decode(js); err != nil {
		errorHandler("error decoding jsonrpc messaage", err, errorGauge)
		return err
	}

	if err := js.Validate(); err != nil {
		errorHandler("error validating jsonrpc messaage", err, errorGauge)
		return err
	}

	return nil
}
