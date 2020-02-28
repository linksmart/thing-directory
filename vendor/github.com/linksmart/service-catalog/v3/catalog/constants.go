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

	MaxServiceTTL = 2147483647 // in seconds i.e. 2^31 - 1 seconds or approx. 68 years, inspired my max TTL value for a DNS record. See RFC 2181
)

var SupportedBackends = map[string]bool{
	CatalogBackendMemory:  true,
	CatalogBackendLevelDB: true,
}
