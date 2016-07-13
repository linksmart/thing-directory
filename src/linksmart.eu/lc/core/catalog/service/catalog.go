// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package service

import (
	"fmt"
	"time"
)

// Structs

// Service is a service entry in the catalog
type Service struct {
	Id             string                 `json:"id"`
	URL            string                 `json:"url"`
	Type           string                 `json:"type"`
	Name           string                 `json:"name,omitempty"`
	Description    string                 `json:"description,omitempty"`
	Meta           map[string]interface{} `json:"meta,omitempty"`
	Protocols      []Protocol             `json:"protocols"`
	Representation map[string]interface{} `json:"representation,omitempty"`
	Ttl            int                    `json:"ttl,omitempty"`
	Created        time.Time              `json:"created"`
	Updated        time.Time              `json:"updated"`
	Expires        *time.Time             `json:"expires,omitempty"`
}

// Validates the Service configuration
func (s *Service) validate() error {

	// Validate protocols
	if len(s.Protocols) == 0 {
		return fmt.Errorf("At least one protocol must be defined")
	}
	for _, protocol := range s.Protocols {
		if protocol.Type == "" {
			return fmt.Errorf("Each protocol must have a type")
		}
		if len(protocol.Endpoint) == 0 {
			return fmt.Errorf("Each protocol must have at least one endpoint")
		}
	}

	return nil
}

// Checks whether the service can be tunneled in GC
func (s *Service) isGCTunnelable() bool {
	// Until the service discovery in GC is not working properly,
	// we can only tunnel services that never expire (tll == 0)
	if s.Ttl != 0 {
		return false
	}

	v, ok := s.Meta[MetaKeyGCExpose]
	if !ok {
		return false
	}

	// flag should be bool
	e := v.(bool)
	if e == true {
		return true
	}
	return false
}

// Protocol describes the service API
type Protocol struct {
	Type         string                 `json:"type"`
	Endpoint     map[string]interface{} `json:"endpoint"`
	Methods      []string               `json:"methods,omitempty"`
	ContentTypes []string               `json:"content-types,omitempty"`
}

// Interfaces

// Controller interface
type CatalogController interface {
	add(s Service) (string, error)
	get(id string) (*Service, error)
	update(id string, s Service) error
	delete(id string) error
	list(page, perPage int) ([]Service, int, error)
	filter(path, op, value string, page, perPage int) ([]Service, int, error)
	total() (int, error)
	cleanExpired()

	Stop() error
}

// Storage interface
type CatalogStorage interface {
	add(s *Service) error
	get(id string) (*Service, error)
	update(id string, s *Service) error
	delete(id string) error
	list(page, perPage int) ([]Service, int, error)
	total() (int, error)
	Close() error
}

// Listener interface can be used for notification of the catalog updates
// NOTE: Implementations are expected to be thread safe
type Listener interface {
	added(s Service)
	updated(s Service)
	deleted(id string)
}
