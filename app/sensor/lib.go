package sensor

import (
	"fmt"
	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	db "github.impcloud.net/RSP-Inventory-Suite/go-dbWrapper"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/jsonrpc"
	"io/ioutil"
	"net/http"
)

func GetOrCreateRSP(dbs *db.DB, deviceId string) (*RSP, error) {
	rsp, err := FindRSP(dbs, deviceId)
	if err != nil {
		return nil, err
	} else if rsp == nil {
		rsp = NewRSP(deviceId)

		// this is a new sensor, try and get the info from the RSP Controller
		info, err := QuerySensorBasicInfo(deviceId)
		if err != nil {
			// just warn, we still want to put it in the database
			logrus.Warn(errors.Wrapf(err, "unable to query sensor basic info for device %s", deviceId))
		} else {
			// update the info before upserting
			rsp.Personality = Personality(info.Personality)
			rsp.Aliases = info.Aliases
			rsp.FacilityId = info.FacilityId
		}

		if err = Upsert(dbs, rsp); err != nil {
			return nil, err
		}
	}

	return rsp, nil
}

func QuerySensorBasicInfo(deviceId string) (*jsonrpc.SensorBasicInfo, error) {
	url := fmt.Sprintf("http://edgex-core-command:48082/api/v1/device/name/%s/command/sensor_get_basic_info", deviceId)
	logrus.Infof("Making GET call to %s", url)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	logrus.Info(string(body))

	evt := new(models.Event)
	if err := evt.UnmarshalJSON(body); err != nil {
		return nil, err
	}

	if len(evt.Readings) == 0 {
		return nil, errors.New("response contained no reading values!")
	}

	sensorInfo := new(jsonrpc.SensorBasicInfo)
	if err := jsonrpc.Decode(evt.Readings[0].Value, sensorInfo, nil); err != nil {
		return nil, err
	}

	return sensorInfo, nil
}
