package sensor

import (
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
	db "github.impcloud.net/RSP-Inventory-Suite/go-dbWrapper"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
	"time"
)

const (
	rspCollection = "rspconfig"
	deviceIdField = "device_id"
)

// FindRSP searches DB for RSP based on the device_id value
// Returns the RSP if found or empty RSP if it does not exist
func FindRSP(dbs *db.DB, deviceId string) (*RSP, error) {
	retrieveTimer := time.Now()

	// Metrics
	metrics.GetOrRegisterGauge(`Sensor.FindRSP.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Sensor.FindRSP.Success`, nil)
	mFindErr := metrics.GetOrRegisterGauge("Sensor.FindRSP.Find-Error", nil)
	defer metrics.GetOrRegisterTimer(`Sensor.FindRSP.Find-Latency`, nil).Update(time.Since(retrieveTimer))

	rsp := new(RSP)

	execFunc := func(collection *mgo.Collection) error {
		return collection.Find(bson.M{deviceIdField: deviceId}).One(rsp)
	}
	if err := dbs.Execute(rspCollection, execFunc); err != nil {
		// If the error was because item does not exist, return empty struct and no error
		if err == mgo.ErrNotFound {
			return nil, nil
		}
		mFindErr.Update(1)
		return nil, errors.Wrapf(err, "db.%s.find()", rspCollection)
	}

	mSuccess.Update(1)
	return rsp, nil
}

// Upsert takes a pointer to an rsp config and either adds it to the DB if it is new,
// or updates its values if it is existing
func Upsert(dbs *db.DB, rsp *RSP) error {
	upsertTimer := time.Now()

	// Metrics
	metrics.GetOrRegisterGaugeCollection(`Sensor.RSP.Upsert.Attempt`, nil).Add(1)
	mSuccess := metrics.GetOrRegisterGaugeCollection(`Sensor.RSP.Upsert.Success`, nil)
	mUpsertErr := metrics.GetOrRegisterGaugeCollection(`Sensor.RSP.Upsert.Error`, nil)
	defer metrics.GetOrRegisterTimer(`Sensor.RSP.Upsert.Latency`, nil).Update(time.Since(upsertTimer))

	execFunc := func(collection *mgo.Collection) error {
		_, err := collection.Upsert(bson.M{deviceIdField: rsp.DeviceId}, rsp)
		return err
	}

	err := dbs.Execute(rspCollection, execFunc)

	if err != nil {
		mUpsertErr.Add(1)
		return errors.Wrapf(err, "db.%s.Upsert()", rspCollection)
	}

	mSuccess.Add(1)
	return nil
}
