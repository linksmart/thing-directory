package resource

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"linksmart.eu/lc/core/catalog"
	"linksmart.eu/lc/sec/auth/obtainer"
)

type RemoteCatalogClient struct {
	serverEndpoint *url.URL
	ticket         *obtainer.Client
}

func NewRemoteCatalogClient(serverEndpoint string, ticket *obtainer.Client) CatalogClient {
	// Check if serverEndpoint is a correct URL
	endpointUrl, err := url.Parse(serverEndpoint)
	if err != nil {
		return &RemoteCatalogClient{}
	}

	return &RemoteCatalogClient{
		serverEndpoint: endpointUrl,
		ticket:         ticket,
	}
}

func (self *RemoteCatalogClient) Get(id string) (*SimpleDevice, error) {
	res, err := catalog.HTTPRequest("GET",
		fmt.Sprintf("%v/%v", self.serverEndpoint, id),
		nil,
		nil,
		self.ticket,
	)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == http.StatusNotFound {
		return nil, &NotFoundError{res.Status}
	} else if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%v", res.StatusCode)
	}

	decoder := json.NewDecoder(res.Body)
	defer res.Body.Close()

	var d SimpleDevice
	err = decoder.Decode(&d)
	if err != nil {
		return nil, err
	}

	return &d, nil
}

func (self *RemoteCatalogClient) Add(d *Device) (*SimpleDevice, error) {
	b, _ := json.Marshal(d)
	res, err := catalog.HTTPRequest("POST",
		fmt.Sprintf("%v/%v/", self.serverEndpoint.String(), FTypeDevices),
		map[string][]string{"Content-Type": []string{"application/ld+json"}},
		bytes.NewReader(b),
		self.ticket,
	)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("Cannot add registration: %v", res.StatusCode)
	}

	location, err := res.Location()
	if err != nil {
		return nil, err
	}

	return self.Get(location.String())
}

func (self *RemoteCatalogClient) Update(id string, d *Device) (*SimpleDevice, error) {
	b, _ := json.Marshal(d)
	res, err := catalog.HTTPRequest("PUT",
		fmt.Sprintf("%v/%v/%v", self.serverEndpoint, FTypeDevices, id),
		nil,
		bytes.NewReader(b),
		self.ticket,
	)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == http.StatusNotFound {
		return nil, &NotFoundError{res.Status}
	} else if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%v", res.StatusCode)
	}

	location, err := res.Location()
	if err != nil {
		return nil, err
	}

	return self.Get(location.String())
}

func (self *RemoteCatalogClient) Delete(id string) error {
	res, err := catalog.HTTPRequest("DELETE",
		fmt.Sprintf("%v/%v/%v", self.serverEndpoint, FTypeDevices, id),
		nil,
		bytes.NewReader([]byte{}),
		self.ticket,
	)
	if err != nil {
		return err
	}

	if res.StatusCode == http.StatusNotFound {
		return &NotFoundError{res.Status}
	} else if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%v", res.StatusCode)
	}

	return nil
}

func (self *RemoteCatalogClient) List(page int, perPage int) ([]SimpleDevice, int, error) {
	res, err := catalog.HTTPRequest("GET",
		fmt.Sprintf("%v/%v?%v=%v&%v=%v", self.serverEndpoint, FTypeDevices, GetParamPage, page, GetParamPerPage, perPage),
		nil,
		nil,
		self.ticket,
	)
	if err != nil {
		return nil, 0, err
	}

	decoder := json.NewDecoder(res.Body)
	defer res.Body.Close()

	var coll DeviceCollection
	err = decoder.Decode(&coll)
	if err != nil {
		return nil, 0, err
	}

	return coll.Devices, coll.Total, nil
}

func (self *RemoteCatalogClient) Filter(path, op, value string, page, perPage int) ([]SimpleDevice, int, error) {
	res, err := catalog.HTTPRequest("GET",
		fmt.Sprintf("%v/%v/%v/%v/%v?%v=%v&%v=%v",
			self.serverEndpoint, FTypeDevices, path, op, value, GetParamPage, page, GetParamPerPage, perPage),
		nil,
		nil,
		self.ticket,
	)
	if err != nil {
		return nil, 0, err
	}

	decoder := json.NewDecoder(res.Body)
	defer res.Body.Close()

	var coll DeviceCollection
	err = decoder.Decode(&coll)
	if err != nil {
		return nil, 0, err
	}

	return coll.Devices, coll.Total, nil
}

func (self *RemoteCatalogClient) GetResource(id string) (*Resource, error) {
	res, err := catalog.HTTPRequest("GET",
		fmt.Sprintf("%v/%v/%v", self.serverEndpoint, FTypeResources, id),
		nil,
		nil,
		self.ticket,
	)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == http.StatusNotFound {
		return nil, &NotFoundError{res.Status}
	} else if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%v", res.StatusCode)
	}

	decoder := json.NewDecoder(res.Body)
	defer res.Body.Close()

	var r Resource
	err = decoder.Decode(&r)
	if err != nil {
		return nil, err
	}

	return &r, nil
}

func (self *RemoteCatalogClient) ListResources(page int, perPage int) ([]Resource, int, error) {
	res, err := catalog.HTTPRequest("GET",
		fmt.Sprintf("%v/%v?%v=%v&%v=%v", self.serverEndpoint, FTypeResources, GetParamPage, page, GetParamPerPage, perPage),
		nil,
		nil,
		self.ticket,
	)
	if err != nil {
		return nil, 0, err
	}

	if res.StatusCode == http.StatusNotFound {
		return nil, 0, &NotFoundError{res.Status}
	} else if res.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("%v", res.StatusCode)
	}

	decoder := json.NewDecoder(res.Body)
	defer res.Body.Close()

	var coll ResourceCollection
	err = decoder.Decode(&coll)
	if err != nil {
		return nil, 0, err
	}

	return coll.Resources, coll.Total, nil
}

func (self *RemoteCatalogClient) FilterResources(path, op, value string, page, perPage int) ([]Resource, int, error) {
	res, err := catalog.HTTPRequest("GET",
		fmt.Sprintf("%v/%v/%v/%v/%v?%v=%v&%v=%v",
			self.serverEndpoint, FTypeResources, path, op, value, GetParamPage, page, GetParamPerPage, perPage),
		nil,
		nil,
		self.ticket,
	)
	if err != nil {
		return nil, 0, err
	}

	decoder := json.NewDecoder(res.Body)
	defer res.Body.Close()

	var coll ResourceCollection
	err = decoder.Decode(&coll)
	if err != nil {
		return nil, 0, err
	}

	return coll.Resources, coll.Total, nil
}
