package sensor

import (
	"database/sql"
	"fmt"
	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/jsonrpc"
	"io/ioutil"
	"net/http"
)

const (
	GetBasicInfo = "sensor_get_basic_info"
)

// GetOrCreateRSP returns a pointer to an RSP if found in the DB, and if
// not found in the DB, a record will be created and added, then returned to the caller
// error is only non-nil when there is an issue communicating with the DB
func GetOrCreateRSP(dbs *sql.DB, deviceId string) (*RSP, error) {
	rsp, err := FindRSP(dbs, deviceId)
	if err != nil {
		return nil, err
	} else if rsp == nil {
		rsp = NewRSP(deviceId)

		// this is a new sensor, try and obtain the actual info from the RSP Controller
		info, err := QueryBasicInfo(deviceId)
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

// QueryBasicInfo makes a call to the EdgeX command service to request the RSP-Controller
// to return us more information about a given RSP sensor
func QueryBasicInfo(deviceId string) (*jsonrpc.SensorBasicInfo, error) {
	reading, err := ExecuteSensorCommand(deviceId, GetBasicInfo)
	if err != nil {
		return nil, err
	}

	sensorInfo := new(jsonrpc.SensorBasicInfo)
	if err := jsonrpc.Decode(reading.Value, sensorInfo, nil); err != nil {
		return nil, err
	}

	return sensorInfo, nil
}

// ExecuteSensorCommand makes an HTTP GET call to the EdgeX core command service to execute
// a specified command on a given RSP sensor
func ExecuteSensorCommand(deviceId string, commandName string) (*models.Reading, error) {
	url := fmt.Sprintf("%s/api/v1/device/name/%s/command/%s", config.AppConfig.CoreCommandUrl, deviceId, commandName)
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

	return &evt.Readings[0], nil
}
