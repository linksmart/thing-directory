// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

type LocalCatalogClient struct {
	controller CatalogController
}

func NewLocalCatalogClient(controller CatalogController) CatalogClient {
	return &LocalCatalogClient{
		controller: controller,
	}
}

// Adds a device and returns its id
func (self *LocalCatalogClient) Add(r *Device) (string, error) {
	return self.controller.add(*r)
}

func (self *LocalCatalogClient) Update(id string, r *Device) error {
	return self.controller.update(id, *r)
}

func (self *LocalCatalogClient) Delete(id string) error {
	return self.controller.delete(id)
}

func (self *LocalCatalogClient) Get(id string) (*SimpleDevice, error) {
	return self.controller.get(id)
}

func (self *LocalCatalogClient) List(page int, perPage int) ([]SimpleDevice, int, error) {
	return self.controller.list(page, perPage)
}

func (self *LocalCatalogClient) GetResource(id string) (*Resource, error) {
	return self.controller.getResource(id)
}

func (self *LocalCatalogClient) ListResources(page int, perPage int) ([]Resource, int, error) {
	return self.controller.listResources(page, perPage)
}

func (self *LocalCatalogClient) Filter(path, op, value string, page, perPage int) ([]SimpleDevice, int, error) {
	return self.controller.filter(path, op, value, page, perPage)
}

func (self *LocalCatalogClient) FilterResources(path, op, value string, page, perPage int) ([]Resource, int, error) {
	return self.controller.filterResources(path, op, value, page, perPage)
}
