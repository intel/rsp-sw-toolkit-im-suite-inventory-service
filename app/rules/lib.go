package rules

import (
	"bytes"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/tag"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	jsonApplication = "application/json;charset=utf-8"
)

func ApplyRules(source string, tagStateChangeList []tag.TagStateChange) error {
	if source == "handheld" || config.AppConfig.TriggerRulesOnFixedTags == false {
		// Run only the StateChanged rule since handheld or not triggering on fixed tags
		return TriggerRules(config.AppConfig.RulesUrl+config.AppConfig.TriggerRulesEndpoint+"?ruletype="+tag.StateChangeEvent, tagStateChangeList)
	} else {
		// Run all rules
		return TriggerRules(config.AppConfig.RulesUrl+config.AppConfig.TriggerRulesEndpoint, tagStateChangeList)
	}
}

func TriggerRules(triggerRulesEndpoint string, data interface{}) error {
	timeout := time.Duration(config.AppConfig.EndpointConnectionTimedOutSeconds) * time.Second
	client := &http.Client{
		Timeout: timeout,
	}

	mData, err := json.Marshal(data)
	if err != nil {
		return errors.Wrapf(err, "problem marshalling the data")
	}

	// Make the POST to authenticate
	request, err := http.NewRequest("POST", triggerRulesEndpoint, bytes.NewBuffer(mData))
	if err != nil {
		return errors.Wrapf(err, "unable to create http.NewRquest")
	}
	request.Header.Set("content-type", jsonApplication)

	response, err := client.Do(request)
	if err != nil {
		return errors.Wrapf(err, "unable trigger rules: %s", triggerRulesEndpoint)
	}
	defer func() {
		if respErr := response.Body.Close(); respErr != nil {
			logrus.WithFields(logrus.Fields{
				"Method": "triggerRules",
				"Action": "response.Body.Close()",
			}).Warning("Failed to close response.")
		}
	}()

	if response.StatusCode != http.StatusOK {
		responseData, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return errors.Wrapf(err, "unable to ReadALL response.Body")
		}
		return errors.Wrapf(errors.New("execution error"), "StatusCode %d , Response %s",
			response.StatusCode, string(responseData))
	}
	return nil
}
