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
	"fmt"
	"github.impcloud.net/RSP-Inventory-Suite/go-dbWrapper"
	"github.impcloud.net/RSP-Inventory-Suite/inventory-service/app/config"
	"log"
	"sync"
	"testing"
	"time"
)

type DBHost string

// InitHost returns a DBHost instance constructed from the given name, appended
// with _HH:MM:SS to ensure that if parallel testing instances for the inventory
// service are hitting the same mongodb, they will not use the same database
// names unless they are launched within the same second.
func InitHost(dbName string) DBHost {
	if err := config.InitConfig(); err != nil {
		log.Fatalf("unable to initialize config: %+v", err)
	}
	return DBHost(config.AppConfig.ConnectionString + "/" + dbName +
		time.Now().Format("_15:04:05"))
}

var dbNamesToInstances = map[string]int{}
var dbNameLock = sync.Mutex{}

// CreateDB returns a database session constructed as DBHost_testName, where
// testName is taken from t.Testing.Name().
//
// Because mongodb names can only be 64 characters, longer names are truncated
// to 62 characters, and a monotonically increasing, two-digit identifier is
// appended to the name, ensuring uniqueness (unless you manage to construct
// over 100 very long names with nearly identical prefixes, in which case you
// should really reconsider your identifier habits). In that unlikely case, mongo
// will return an "Invalid database name" error.
//
// Note that mongo also restricts the use of `/\. "$` on all OSes and
// additionally `*<>:|?` on Windows; since none of those are valid Go identifiers
// anyway, this is ignored.
func (dbHost DBHost) CreateDB(t *testing.T) *mongodb.DB {
	t.Helper()

	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	dbName := string(dbHost)+t.Name()
	if len(dbName) > 64 {
		dbName = dbName[:62]
	}

	dbNameLock.Lock()
	dbNamesToInstances[dbName]++
	dbName = dbName + fmt.Sprintf("%02d", dbNamesToInstances[dbName])
	dbNameLock.Unlock()
	t.Logf("using db %s", dbName)

	masterDb, err := mongodb.NewSession(dbName, 10*time.Second)
	if err != nil {
		t.Fatalf("Unable to connect to db server at %s", dbName)
	}

	return masterDb
}
