// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"fmt"

	"github.com/linksmart/thing-directory/wot"
)

type ThingDescription = map[string]interface{}

const (
	// TD keys
	_id       = "id"
	_created  = "created"
	_modified = "modified"
	_ttl      = "ttl"
)

func validateThingDescription(td map[string]interface{}) error {
	_, ok := td[_ttl].(float64)
	if !ok {
		return fmt.Errorf("ttl is not float64")
	}

	return wot.ValidateAgainstWoTSchema(&td)
}

// Controller interface
type CatalogController interface {
	add(d ThingDescription) (string, error)
	get(id string) (ThingDescription, error)
	update(id string, d ThingDescription) error
	delete(id string) error
	list(page, perPage int) ([]ThingDescription, int, error)
	filter(path, op, value string, page, perPage int) ([]ThingDescription, int, error)
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
	total() (int, error)
	iterator() <-chan ThingDescription
	Close()
}
