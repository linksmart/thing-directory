// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"context"
	"fmt"

	"github.com/linksmart/thing-directory/wot"
)

type ThingDescription = map[string]interface{}

const (
	ResponseContextURL = "https://linksmart.eu/thing-directory/context.jsonld"
	ResponseType       = "Catalog"
	// Storage backend types
	BackendMemory  = "memory"
	BackendLevelDB = "leveldb"
)

func validateThingDescription(td map[string]interface{}) ([]wot.ValidationError, error) {
	result, err := wot.ValidateTD(&td)
	if err != nil {
		return nil, fmt.Errorf("error validating with JSON Schemas: %s", err)
	}
	return result, nil
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
	iterateBytes(ctx context.Context) <-chan []byte
	cleanExpired()

	Stop()

	AddSubscriber(listener EventListener)
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
	iterateBytes(ctx context.Context) <-chan []byte
	Close()
}
