// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

const (
	DNSSDServiceType = "_linksmart-sc._tcp"
	MaxPerPage       = 100
	LoggerPrefix     = "[sc] "

	CatalogBackendMemory  = "memory"
	CatalogBackendLevelDB = "leveldb"

	APITypeHTTP = "HTTP"
	APITypeMQTT = "MQTT"

	MaxServiceTTL = 24 * 60 * 60 //in seconds
)

var SupportedBackends = map[string]bool{
	CatalogBackendMemory:  true,
	CatalogBackendLevelDB: true,
}
