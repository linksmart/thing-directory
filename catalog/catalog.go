// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"github.com/linksmart/resource-catalog/wot"
)

type ThingDescription struct {
	wot.ThingDescription
	TTL uint `json:"ttl,omitempty"`
}

func (td *ThingDescription) validate() error {
	return wot.ValidateAgainstWoTSchema(&td.ThingDescription)
}

// Controller interface
type CatalogController interface {
	add(d ThingDescription) (string, error)
	get(id string) (*ThingDescription, error)
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
	add(td *ThingDescription) error
	update(id string, td *ThingDescription) error
	delete(id string) error
	get(id string) (*ThingDescription, error)
	list(page, perPage int) ([]ThingDescription, int, error)
	total() (int, error)
	iterator() <-chan *ThingDescription
	Close()
}
