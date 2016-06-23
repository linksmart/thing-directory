package service

import (
	"time"
)

// Structs

// Service is a service entry in the catalog
type Service struct {
	Id             string                 `json:"id"`
	URL            string                 `json:"url"`
	Type           string                 `json:"type"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Meta           map[string]interface{} `json:"meta"`
	Protocols      []Protocol             `json:"protocols"`
	Representation map[string]interface{} `json:"representation"`
	Ttl            int                    `json:"ttl"`
	Created        time.Time              `json:"created"`
	Updated        time.Time              `json:"updated"`
	Expires        *time.Time             `json:"expires"`
}

// Validates the Service configuration
func (s *Service) validate() error {
	/*	if s.Id == "" || len(strings.Split(s.Id, "/")) != 2 || s.Name == "" || s.Ttl == 0 {
			return false
		}
		return true*/

	return nil
}

// Checks whether the service can be tunneled in GC
func (s *Service) isGCTunnelable() bool {
	// Until the service discovery in GC is not working properly,
	// we can only tunnel services that never expire (tll == -1)
	if s.Ttl != -1 {
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
	Methods      []string               `json:"methods"`
	ContentTypes []string               `json:"content-types"`
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
