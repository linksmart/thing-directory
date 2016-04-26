package resource

import (
	"fmt"
	"time"
)

// Structs

type Resources []Resource

// Device entry in the catalog
type Device struct {
	Id          string                 `json:"id"`
	URL         string                 `json:"url"`
	Type        string                 `json:"type"`
	Name        string                 `json:"name,omitempty"`
	Meta        map[string]interface{} `json:"meta,omitempty"`
	Description string                 `json:"description,omitempty"`
	Ttl         int                    `json:"ttl,omitempty"`
	Created     time.Time              `json:"created"`
	Updated     time.Time              `json:"updated"`
	Expires     *time.Time             `json:"expires,omitempty"`
	Resources   Resources              `json:"resources"`
}

type SimpleDevice struct {
	*Device
	Resources []string `json:"resources"`
}

// Resource exposed by a device
type Resource struct {
	Id             string                 `json:"id"`
	URL            string                 `json:"url"`
	Type           string                 `json:"type"`
	Name           string                 `json:"name,omitempty"`
	Meta           map[string]interface{} `json:"meta,omitempty"`
	Protocols      []Protocol             `json:"protocols"`
	Representation map[string]interface{} `json:"representation,omitempty"`
	Device         string                 `json:"device"` // URL of device
}

// Protocol describes the resource API
type Protocol struct {
	Type         string                 `json:"type"`
	Endpoint     map[string]interface{} `json:"endpoint"`
	Methods      []string               `json:"methods"`
	ContentTypes []string               `json:"content-types"`
}

// Validates the Device configuration
func (d *Device) validate() error {

	if d.Ttl == 0 {
		return fmt.Errorf("Device TTL must not be zero")
	}

	// validate all resources
	for _, r := range d.Resources {
		if err := r.validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validates the Resource configuration
func (r *Resource) validate() error {

	return nil
}

// Interfaces

type CatalogController interface {
	list(page, perPage int) ([]SimpleDevice, int, error)
	add(d *Device) error
	get(id string) (*SimpleDevice, error)
	update(id string, d *Device) error
	delete(id string) error
	filter(path, op, value string, page, perPage int) ([]SimpleDevice, int, error)
	total() (int, error)
	cleanExpired(d time.Duration)

	listResources(page, perPage int) ([]Resource, int, error)
	getResource(id string) (*Resource, error)
	filterResources(path, op, value string, page, perPage int) ([]Resource, int, error)
	totalResources() (int, error)
}

// Storage interface
type CatalogStorage interface {
	list(page, perPage int) ([]Device, int, error)
	add(d *Device) error
	update(id string, d *Device) error
	delete(id string) error
	get(id string) (*Device, error)
	total() (int, error)
	Close() error
}

// Sorted-map data structure based on AVL Tree (go-avltree)
type SortedMap struct {
	key   interface{}
	value interface{}
}
// Operator for string-type key
func stringKeys(a interface{}, b interface{}) int {
	if a.(SortedMap).key.(string) < b.(SortedMap).key.(string) {
		return -1
	} else if a.(SortedMap).key.(string) > b.(SortedMap).key.(string) {
		return 1
	}
	return 0
}
// Operator for Time-type key
func timeKeys(a interface{}, b interface{}) int {
	if a.(SortedMap).key.(time.Time).Before(b.(SortedMap).key.(time.Time)) {
		return -1
	} else if a.(SortedMap).key.(time.Time).After(b.(SortedMap).key.(time.Time)) {
		return 1
	}
	return 0
}
