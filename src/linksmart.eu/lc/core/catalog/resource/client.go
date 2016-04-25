package resource

////DeviceConfig is a wrapper for Device to be used by
////clients to configure a Device (e.g., read from file)
//type DeviceConfig struct {
//	*Device
//	Host string
//}
//
//// Returns a Device object from the DeviceConfig
//func (dc *DeviceConfig) GetDevice() (*Device, error) {
//	dc.Id = dc.Host + "/" + dc.Name
//	if err := dc.Device.validate(); err != nil {
//		return nil, fmt.Errorf("Invalid Device registration: %s", err)
//	}
//	return dc.Device, nil
//}

// Catalog client
type CatalogClient interface {
	// CRUD
	Get(id string) (*SimpleDevice, error)
	Add(d *Device) error
	Update(id string, d *Device) error
	Delete(id string) error

	// Returns a slice of Devices given:
	// page - page in the collection
	// perPage - number of entries per page
	List(page, perPage int) ([]SimpleDevice, int, error)

	// Returns a single resource
	GetResource(id string) (*Resource, error)

	// Returns a slice of Resources given:
	// page - page in the collection
	// perPage - number of entries per page
	ListResources(page, perPage int) ([]Resource, int, error)

	// Returns a slice of Devices given: path, operation, value, page, perPage
	Filter(path, op, value string, page, perPage int) ([]SimpleDevice, int, error)

	// Returns a slice of Resources given: path, operation, value, page, perPage
	FilterResources(path, op, value string, page, perPage int) ([]Resource, int, error)
}
