// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"fmt"

	"github.com/linksmart/thing-directory/wot"
)

type ThingDescription = map[string]interface{}

const (
	ResponseContextURL    = "https://linksmart.eu/thing-directory/context.jsonld"
	ResponseType          = "Catalog"
	ResponseMediaType     = "application/ld+json"
	ResponseJSONMediaType = "application/json"
	// DNS-SD
	DNSSDServiceType    = "_wot._tcp"
	DNSSDServiceSubtype = "_directory" // _directory._sub._wot._tcp
	// Storage backend types
	BackendMemory  = "memory"
	BackendLevelDB = "leveldb"
	// TD keys used internally
	_id       = "id"
	_created  = "created"
	_modified = "modified"
	_ttl      = "ttl"
)

func validateThingDescription(td map[string]interface{}) error {
	if td[_ttl] != nil {
		_, ok := td[_ttl].(float64)
		if !ok {
			return fmt.Errorf("ttl is %T instead of float64", td[_ttl])
		}
	}

	return wot.ValidateMap(&td)
}

// Controller interface
type CatalogController interface {
	add(d ThingDescription) (string, error)
	get(id string) (ThingDescription, error)
	update(id string, d ThingDescription) error
	patch(id string, d ThingDescription) error
	delete(id string) error
	list(page, perPage int) ([]ThingDescription, int, error)
	listAllBytes() ([]byte, error)
	// Deprecated
	filterJSONPath(path string, page, perPage int) ([]interface{}, int, error)
	filterJSONPathBytes(query string) ([]byte, error)
	// Deprecated
	filterXPath(path string, page, perPage int) ([]interface{}, int, error)
	filterXPathBytes(query string) ([]byte, error)
	//filterXPathBytes(query string) ([]byte, error)
	total() (int, error)
	cleanExpired()

	Stop()
}

// Storage interface
type Storage interface {
	add(id string, td ThingDescription) error
	update(id string, td ThingDescription) error
	delete(id string) error
	get(id string) (ThingDescription, error)
	list(page, perPage int) ([]ThingDescription, int, error)
	listAllBytes() ([]byte, error)
	total() (int, error)
	iterator() <-chan ThingDescription
	Close()
}
