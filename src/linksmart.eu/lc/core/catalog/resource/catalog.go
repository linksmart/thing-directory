package resource

import (
	"fmt"
	"net/url"
	"time"
)

// STRUCTS

type Resources []Resource
type Devices []Device

// Device
type Device struct {
	Id          string                 `json:"id"`
	URL         string                 `json:"url"`
	Type        string                 `json:"type"`
	Name        string                 `json:"name,omitempty"`
	Meta        map[string]interface{} `json:"meta,omitempty"`
	Description string                 `json:"description,omitempty"`
	Ttl         uint                   `json:"ttl,omitempty"`
	Created     time.Time              `json:"created"`
	Updated     time.Time              `json:"updated"`
	Expires     *time.Time             `json:"expires,omitempty"`
	Resources   Resources              `json:"resources"`
}

// Device with only IDs of resources
type SimpleDevice struct {
	Device
	Resources []string `json:"resources"`
}

// Resource
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
	_, err := url.Parse(d.Id)
	if err != nil {
		return fmt.Errorf("Device id %s cannot be used in a URL: %s", d.Id, err)
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
	_, err := url.Parse(r.Id)
	if err != nil {
		return fmt.Errorf("Resource id %s cannot be used in a URL: %s", r.Id, err)
	}

	return nil
}

// Converts a Device into SimpleDevice
func (d *Device) simplify() *SimpleDevice {
	resourceIDs := make([]string, len(d.Resources))
	for i := 0; i < len(d.Resources); i++ {
		resourceIDs[i] = d.Resources[i].URL
	}
	sd := &SimpleDevice{*d, resourceIDs}
	sd.Device.Resources = nil
	return sd
}

// Converts Devices into []SimpleDevice
func (devices Devices) simplify() []SimpleDevice {
	simpleDevices := make([]SimpleDevice, len(devices))
	for i := 0; i < len(devices); i++ {
		simpleDevices[i] = *devices[i].simplify()
	}
	return simpleDevices
}

// INTERFACES

// Controller interface
type CatalogController interface {
	// Devices
	add(d Device) (*SimpleDevice, error)
	get(id string) (*SimpleDevice, error)
	update(id string, d Device) (*SimpleDevice, error)
	delete(id string) error
	list(page, perPage int) ([]SimpleDevice, int, error)
	filter(path, op, value string, page, perPage int) ([]SimpleDevice, int, error)
	total() (int, error)
	cleanExpired()

	// Resources
	getResource(id string) (*Resource, error)
	listResources(page, perPage int) ([]Resource, int, error)
	filterResources(path, op, value string, page, perPage int) ([]Resource, int, error)
	totalResources() (int, error)

	Stop() error
}

// Storage interface
type CatalogStorage interface {
	add(d *Device) error
	update(id string, d *Device) error
	delete(id string) error
	get(id string) (*Device, error)
	list(page, perPage int) (Devices, int, error)
	total() (int, error)
	Close() error
}
