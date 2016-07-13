// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package service

// ServiceConfig is a wrapper for Service to be used by
// clients to configure a Service (e.g., read from file)
type ServiceConfig struct {
	*Service
	Host string
}

// Returns a Service object from the ServiceConfig
func (sc *ServiceConfig) GetService() (*Service, error) {
	sc.Id = sc.Host + "/" + sc.Name
	if err := sc.Service.validate(); err != nil {
		return nil, err
	}
	return sc.Service, nil
}

// Catalog client
type CatalogClient interface {
	// CRUD
	Get(id string) (*Service, error)
	Add(s *Service) (string, error)
	Update(id string, s *Service) error
	Delete(id string) error

	// Returns a slice of Services given:
	// page - page in the collection
	// perPage - number of entries per page
	List(page, perPage int) ([]Service, int, error)

	// Returns a slice of Services given: path, operation, value, page, perPage
	Filter(path, op, value string, page, perPage int) ([]Service, int, error)
}
