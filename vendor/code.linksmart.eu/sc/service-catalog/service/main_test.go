// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package service

import (
	"os"
	"testing"
)

const (
	TestApiLocation    = "/sc"
	TestStaticLocation = "/static"
)

var (
	TestSupportedBackends = map[string]bool{
		CatalogBackendMemory:  true,
		CatalogBackendLevelDB: true,
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
