package service

const (
	DNSSDServiceType    = "_linksmart-sc._tcp"
	MaxPerPage          = 100
	ApiVersion          = "1.0.0"
	ApiCollectionType   = "ServiceCatalog"
	ApiRegistrationType = "Service"
	loggerPrefix        = "[sc] "

	// MetaKeyGCExpose is the meta key indicating
	// that the service needs to be tunneled in GC
	MetaKeyGCExpose = "gc_expose"
)
