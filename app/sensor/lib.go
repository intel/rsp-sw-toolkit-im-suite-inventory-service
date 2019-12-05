package sensor

import (
	"database/sql"
	"fmt"
	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/pkg/jsonrpc"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
	"io/ioutil"
	"net/http"
)

const (
	RspController = "rsp-controller"
	GetBasicInfo  = "sensor_get_basic_info"
	GetDeviceIds  = "sensor_get_device_ids"
)

// refreshSensorBasicInfo forces a call out to the RSP Controller to retrieve the sensor basic info (facility, personality, aliases, etc)
func refreshSensorBasicInfo(dbs *sql.DB, deviceId string, insertDefaultsOnError bool) (*RSP, error) {
	rsp := NewRSP(deviceId)

	// this is a new sensor, try and obtain the actual info from the RSP Controller
	info, err := QueryBasicInfo(deviceId)
	if err != nil {
		wrappedErr := errors.Wrapf(err, "unable to query sensor basic info from RSP Controller for sensor %s", deviceId)
		logrus.Warn(wrappedErr)

		if !insertDefaultsOnError {
			return nil, wrappedErr
		}

		// setting rsp.UpdatedOn to 0 lets us know the sensor info has not been retrieved yet
		rsp.UpdatedOn = 0
		// warn, but continue to upsert code
		logrus.Warnf("inserting default values for sensor %+v", rsp)
	} else {
		// update the info before upserting
		rsp.Personality = Personality(info.Personality)
		rsp.Aliases = info.Aliases
		rsp.FacilityId = info.FacilityId
		rsp.UpdatedOn = helper.UnixMilliNow()
	}

	if err = Upsert(dbs, rsp); err != nil {
		return nil, err
	}

	return rsp, nil
}

// GetOrCreateRSP returns a pointer to an RSP if found in the DB, and if
// not found in the DB, a record will be created and added, then returned to the caller
// error is only non-nil when there is an issue communicating with the DB
func GetOrCreateRSP(dbs *sql.DB, deviceId string) (*RSP, error) {
	rsp, err := FindRSP(dbs, deviceId)
	if err != nil {
		return nil, err
	} else if rsp == nil {
		if _, err = refreshSensorBasicInfo(dbs, deviceId, true); err != nil {
			logrus.Error(err)
			logrus.Warnf("unable to query sensor information for %s, inserting default values", deviceId)
			// still insert into the database
			rsp = NewRSP(deviceId)
			if err = Upsert(dbs, rsp); err != nil {
				return nil, err
			}
		}
	} else if rsp.UpdatedOn == 0 {
		// this will only occur if the sensor exists in the database but we have not received the information about it from the controller yet
		// so we query it in the background as an attempt to keep trying to get the information without causing a huge bottleneck
		// if the controller is offline
		logrus.Debugf("missing sensor basic info from RSP controller for sensor %s. attempting to refresh it in the background", deviceId)
		go refreshSensorBasicInfo(dbs, deviceId, false)
	}

	return rsp, nil
}

// QueryBasicInfoAllSensors retrieves the list of deviceIds from the RSP Controller
// and then queries the basic info for each one
func QueryBasicInfoAllSensors(dbs *sql.DB) error {
	reading, err := ExecuteSensorCommand(RspController, GetDeviceIds)
	if err != nil {
		logrus.Error(err)
		return err
	}

	logrus.Debugf("received: %s", reading.Value)

	deviceIds := new(jsonrpc.SensorDeviceIdsResponse)
	if err := jsonrpc.Decode(reading.Value, deviceIds, nil); err != nil {
		return err
	}

	for _, deviceId := range *deviceIds {
		// ignore error because we want to query all sensors
		go refreshSensorBasicInfo(dbs, deviceId, false)
	}

	return nil
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http response returned: %d %s", resp.StatusCode, resp.Status)
	}

	evt := new(models.Event)
	if err := evt.UnmarshalJSON(body); err != nil {
		return nil, err
	}

	if len(evt.Readings) == 0 {
		return nil, errors.New("response contained no reading values!")
	}

	return &evt.Readings[0], nil
}
