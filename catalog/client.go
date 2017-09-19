// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

// Catalog client
type CatalogClient interface {
	// CRUD
	Get(id string) (*SimpleDevice, error)
	Add(d *Device) (string, error)
	Update(id string, d *Device) error
	Delete(id string) error

	// Returns a slice of Devices given:
	// page - page in the collection
	// perPage - number of entries per page
	List(page, perPage int) ([]SimpleDevice, int, error)

	// Returns a slice of Devices given: path, operation, value, page, perPage
	Filter(path, op, value string, page, perPage int) ([]SimpleDevice, int, error)

	// Returns a single resource
	GetResource(id string) (*Resource, error)

	// Returns a slice of Resources given:
	// page - page in the collection
	// perPage - number of entries per page
	ListResources(page, perPage int) ([]Resource, int, error)

	// Returns a slice of Resources given: path, operation, value, page, perPage
	FilterResources(path, op, value string, page, perPage int) ([]Resource, int, error)
}
