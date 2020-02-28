// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"fmt"
	"mime"
	"net/url"
	"strings"
	"time"
)

// Structs

// Service is a service entry in the catalog
type Service struct {
	ID          string                 `json:"id"`
	Description string                 `json:"description"`
	Title       string                 `json:"title"`
	Type        string                 `json:"type"`
	APIs        []API                  `json:"apis"`
	Meta        map[string]interface{} `json:"meta"`
	Doc         string                 `json:"doc"`
	TTL         uint32                 `json:"ttl"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	ExpiresAt   time.Time              `json:"expiresAt"` // the time when service will be removed from the system (unless updated within TTL)
}

// API - an API (e.g. REST API, MQTT API, etc.) exposed by the service
type API struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Protocol    string                 `json:"protocol"`
	URL         string                 `json:"url"`
	Spec        Spec                   `json:"spec"`
	Meta        map[string]interface{} `json:"meta"`
}

// API.spec - the complete specification of the interface exposed by the service
// spec in the form of an url to external specification document is preferred, if not present, the 'schema' could be used
// Recommended - Request-response: OpenAPI/Swagger Spec, PubSub: AsyncAPI Spec
type Spec struct {
	MediaType string                 `json:"mediaType"`
	URL       string                 `json:"url"`
	Schema    map[string]interface{} `json:"schema"`
}

// Validates the Service configuration
func (s Service) validate() error {

	if strings.ContainsAny(s.ID, " ") {
		return fmt.Errorf("service id must not contain spaces")
	}
	_, err := url.Parse("http://example.com/" + s.ID)
	if err != nil {
		return fmt.Errorf("service id is invalid: %v", err)
	}

	if s.Type == "" {
		return fmt.Errorf("service type not defined")
	}
	if strings.ContainsAny(s.Type, " ") {
		return fmt.Errorf("service type must not contain spaces")
	}

	// If a service needs to use the TTL functionality, TTL should be between 1 and 2147483647
	// The appropriately value for TTL should be provided by the service provider based on how critical the availability of his/her service is
	if s.TTL == 0 || s.TTL > MaxServiceTTL {
		return fmt.Errorf("service TTL should be between 1 and %v (seconds)", MaxServiceTTL)
	}

	// TODO: request payload validations as described below (create an issue to discuss and finalize):
	// mandatory: type (done), title, apis[x].title, apis[x].protocol?, apis[x].endpoint?, apis[x].spec?

	for i, API := range s.APIs {

		if API.ID == "" {
			return fmt.Errorf("API id not defined")
		}

		if strings.ContainsAny(API.ID, " ") {
			return fmt.Errorf("API id must not contain spaces")
		}

		for _, nextAPI := range s.APIs[i+1:] {
			if API.ID == nextAPI.ID {
				return fmt.Errorf("API id must be unique among the IDs of APIs of this service")
			}
		}

		if _, err := url.Parse(API.URL); err != nil {
			return fmt.Errorf("invalid service API endpoint: %s", API.URL)
		}

		if _, err := url.Parse(API.Spec.URL); err != nil {
			return fmt.Errorf("invalid API spec url: %s", API.Spec.URL)
		}

		if _, _, err := mime.ParseMediaType(API.Spec.MediaType); err != nil {
			return fmt.Errorf("invalid API Spec mediaType: %s: %s", API.Spec.MediaType, err)
		}
	}

	return nil
}

// Error describes an API error (serializable in JSON)
type Error struct {
	// Code is the (http) code of the error
	Code int `json:"code"`
	// Message is the (human-readable) error message
	Message string `json:"message"`
}

// Interfaces

// Storage interface
type Storage interface {
	add(s *Service) error
	get(id string) (*Service, error)
	update(id string, s *Service) error
	delete(id string) error
	list(page, perPage int) ([]Service, int, error)
	total() (int, error)
	iterator() <-chan *Service
	Close() error
}

// Listener interface can be used for notification of the catalog updates
// NOTE: Implementations are expected to be thread safe
type Listener interface {
	added(s Service)
	updated(s Service)
	deleted(s Service)
}
