package service

import (
	"os"
	"testing"

	utils "linksmart.eu/lc/core/catalog"
)

const (
	TestApiLocation    = "/sc"
	TestStaticLocation = "/static"
)

var (
	TestSupportedBackends = map[string]bool{
		utils.CatalogBackendMemory:  true,
		utils.CatalogBackendLevelDB: true,
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
