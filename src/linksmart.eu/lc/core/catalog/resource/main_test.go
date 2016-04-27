package resource

import (
	"os"
	"testing"

	utils "linksmart.eu/lc/core/catalog"
)

var TestSupportedBackends = map[string]bool{
	utils.CatalogBackendMemory:  true,
	utils.CatalogBackendLevelDB: true,
}
var TestStorageType string

func TestMain(m *testing.M) {
	for b, supported := range TestSupportedBackends {
		if supported {
			TestStorageType = b
			m.Run()
		}
	}
	os.Exit(0)
}
