/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

// Package integrationtest exists to help with the current, somewhat unfortunate
// state of the tests in the inventory service. It centralizes the database access
// to ensure:
//     1. database calls from different tests don't interfere, even if their
//     code under test would normally reference the same database
//     2. multiple, parallel instances of the the test suite will not interfere,
//     even if they're hitting the same mongodb instance
//     3. there's a (blurry, in need of work) separation between tests that rely
//     on a database instance, and those that don't, as well as an escape switch
//     to avoid running those tests unless necessary (namely, the -test.short flag)

package integrationtest

import (
	"database/sql"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"log"
	"strings"
	"sync"
	"testing"
)

type TestDB struct {
	DB     *sql.DB
	dbName string
	dbHost *DBHost
	t      *testing.T
}

type DBHost struct {
	Name     string
	masterDB *sql.DB
}

func NewDBHost(name string) DBHost {
	return DBHost{
		Name: name,
	}
}

// InitHost returns a DBHost instance constructed from the given name
func InitHost(name string) DBHost {
	if err := config.InitConfig(); err != nil {
		log.Fatalf("unable to initialize config: %+v", err)
	}
	dbHost := NewDBHost(name)
	dbHost.Connect()
	return dbHost
}

var dbNamesToInstances = map[string]int{}
var dbNameLock = sync.Mutex{}

func (dbHost *DBHost) CreateDB(t *testing.T) TestDB {
	t.Helper()

	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	dbName := dbHost.Name + "_" + t.Name()

	illegalChars := []string{":", "/", "-"}
	for _, illegalChar := range illegalChars {
		dbName = strings.ReplaceAll(dbName, illegalChar, "_")
	}
	if len(dbName) > 63 {
		dbName = dbName[:62]
	}

	dbNameLock.Lock()
	dbNamesToInstances[dbName]++
	dbName = dbName + fmt.Sprintf("%02d", dbNamesToInstances[dbName])
	dbNameLock.Unlock()
	t.Logf("using db %s", dbName)

	var err error
	// Create the new database
	if _, err = dbHost.masterDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS \"%s\";", dbName)); err != nil {
		t.Fatalf("Unable to drop existing db for: %s: %v", dbName, err)
	}
	if _, err = dbHost.masterDB.Exec(fmt.Sprintf("CREATE DATABASE \"%s\" OWNER %s;", dbName, config.AppConfig.DbUser)); err != nil {
		t.Fatalf("Unable to create to db for: %s: %v", dbName, err)
	}

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=disable",
		config.AppConfig.DbHost,
		config.AppConfig.DbPort,
		config.AppConfig.DbUser,
		dbName)
	if config.AppConfig.DbPass != "" {
		psqlInfo += " password=" + config.AppConfig.DbPass
	}
	// Re-connect to the new database
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		t.Fatalf("Unable to connect to db server at %s: %v", dbName, err)
	}
	if err = db.Ping(); err != nil {
		t.Fatalf("Unable to ping db server at %s: %v", dbName, err)
	}

	// Creation of tables and indexes
	if _, err = db.Exec(config.DbSchema); err != nil {
		t.Fatalf("Unable to create to db tables and indexes for: %s: %v", dbName, err)
	}

	return TestDB{dbName: dbName, DB: db, dbHost: dbHost, t: t}
}

func (dbHost *DBHost) Connect() {
	var err error
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=disable",
		config.AppConfig.DbHost,
		config.AppConfig.DbPort,
		config.AppConfig.DbUser,
		config.AppConfig.DbName)
	if config.AppConfig.DbPass != "" {
		psqlInfo += " password=" + config.AppConfig.DbPass
	}

	dbHost.masterDB, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("Unable to connect to db server: %v", err)
	}
	if err = dbHost.masterDB.Ping(); err != nil {
		log.Fatalf("Unable to ping db server: %v", err)
	}
}

func (testDB *TestDB) Close() {
	testDB.t.Logf("dropping db %s", testDB.dbName)
	// Drop the temp database
	if _, err := testDB.dbHost.masterDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", testDB.dbName)); err != nil {
		testDB.t.Errorf("Unable to drop to db for: %s: %v", testDB.dbName, err)
	}
	if err := testDB.DB.Close(); err != nil {
		testDB.t.Errorf("error on db close: %v", err)
	}
}

func (dbHost *DBHost) Close() {
	if err := dbHost.masterDB.Close(); err != nil {
		logrus.Errorf("error on db close: %v", err)
	}
}
