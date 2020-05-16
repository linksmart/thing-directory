// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/linksmart/thing-directory/wot"
)

const (
	envTestSchemaPath = "TEST_SCHEMA_PATH"
	defaultSchemaPath = "../wot/wot_td_schema.json"
)

type any = interface{}

var (
	TestSupportedBackends = map[string]bool{
		BackendMemory:  false,
		BackendLevelDB: true,
	}
	TestStorageType string
)

func loadSchema() error {
	path := os.Getenv(envTestSchemaPath)
	if path == "" {
		path = defaultSchemaPath
	}
	return wot.LoadSchema(path)
}

func serializedEqual(td1 ThingDescription, td2 ThingDescription) bool {
	// serialize to ease comparison of interfaces and concrete types
	tdBytes, _ := json.Marshal(td1)
	storedTDBytes, _ := json.Marshal(td2)

	return reflect.DeepEqual(tdBytes, storedTDBytes)
}

func TestMain(m *testing.M) {
	// run tests for each storage backend
	for b, supported := range TestSupportedBackends {
		if supported {
			TestStorageType = b
			if m.Run() == 1 {
				os.Exit(1)
			}
		}
	}
	os.Exit(0)
}
