// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"os"
	"testing"
)

const (
	TestApiLocation    = "/rc"
	TestStaticLocation = "/static"
)

var (
	TestSupportedBackends = map[string]bool{
		BackendMemory:  false,
		BackendLevelDB: true,
	}
	TestStorageType string
)

func TestMain(m *testing.M) {
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
