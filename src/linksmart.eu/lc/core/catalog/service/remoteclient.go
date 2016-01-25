package service

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

func serviceFromResponse(res *http.Response, apiLocation string) (*Service, error) {
	decoder := json.NewDecoder(res.Body)
	defer res.Body.Close()

	var s *Service
	err := decoder.Decode(&s)
	if err != nil {
		return nil, err
	}
	svc := s.unLdify(apiLocation)
	return &svc, nil
}

func servicesFromResponse(res *http.Response, apiLocation string) ([]Service, int, error) {
	decoder := json.NewDecoder(res.Body)
	defer res.Body.Close()

	var coll Collection
	err := decoder.Decode(&coll)
	if err != nil {
		return nil, 0, err
	}

	svcs := make([]Service, 0, len(coll.Services))
	for _, v := range coll.Services {
		svcs = append(svcs, v.unLdify(apiLocation))
	}

	return svcs, len(svcs), nil
}

func NewRemoteCatalogClient(serverEndpoint string, ticket *obtainer.Client) *RemoteCatalogClient {
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

func (self *RemoteCatalogClient) Get(id string) (*Service, error) {
	res, err := catalog.HTTPRequest("GET",
		fmt.Sprintf("%v/%v", self.serverEndpoint, id),
		map[string][]string{"Content-Type": []string{"application/ld+json"}},
		nil,
		self.ticket,
	)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == http.StatusNotFound {
		return nil, ErrorNotFound
	} else if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%v", res.StatusCode)
	}
	return serviceFromResponse(res, self.serverEndpoint.Path)
}

func (self *RemoteCatalogClient) Add(s *Service) error {
	b, _ := json.Marshal(s)
	res, err := catalog.HTTPRequest("POST",
		self.serverEndpoint.String()+"/",
		nil,
		bytes.NewReader(b),
		self.ticket,
	)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusCreated {
		return fmt.Errorf("Cannot add registration: %v", res.StatusCode)
	}
	return nil
}

func (self *RemoteCatalogClient) Update(id string, s *Service) error {
	b, _ := json.Marshal(s)
	res, err := catalog.HTTPRequest("PUT",
		fmt.Sprintf("%v/%v", self.serverEndpoint, id),
		nil,
		bytes.NewReader(b),
		self.ticket,
	)
	if err != nil {
		return err
	}

	if res.StatusCode == http.StatusNotFound {
		return ErrorNotFound
	} else if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%v", res.StatusCode)
	}
	return nil
}

func (self *RemoteCatalogClient) Delete(id string) error {
	res, err := catalog.HTTPRequest("DELETE",
		fmt.Sprintf("%v/%v", self.serverEndpoint, id),
		nil,
		bytes.NewReader([]byte{}),
		self.ticket,
	)
	if err != nil {
		return err
	}

	if res.StatusCode == http.StatusNotFound {
		return ErrorNotFound
	} else if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%v", res.StatusCode)
	}

	return nil
}

func (self *RemoteCatalogClient) GetServices(page, perPage int) ([]Service, int, error) {
	res, err := catalog.HTTPRequest("GET",
		fmt.Sprintf("%v?%v=%v&%v=%v",
			self.serverEndpoint, GetParamPage, page, GetParamPerPage, perPage),
		nil,
		nil,
		self.ticket,
	)
	if err != nil {
		return nil, 0, err
	}

	return servicesFromResponse(res, self.serverEndpoint.Path)
}

func (self *RemoteCatalogClient) FindService(path, op, value string) (*Service, error) {
	res, err := catalog.HTTPRequest("GET",
		fmt.Sprintf("%v/%v/%v/%v/%v", self.serverEndpoint, FTypeService, path, op, value),
		nil,
		nil,
		self.ticket,
	)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == http.StatusNotFound {
		return nil, ErrorNotFound
	} else if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%v", res.StatusCode)
	}

	return serviceFromResponse(res, self.serverEndpoint.Path)
}

func (self *RemoteCatalogClient) FindServices(path, op, value string, page, perPage int) ([]Service, int, error) {
	res, err := catalog.HTTPRequest("GET",
		fmt.Sprintf("%v/%v/%v/%v/%v?%v=%v&%v=%v",
			self.serverEndpoint, FTypeServices, path, op, value, GetParamPage, page, GetParamPerPage, perPage),
		nil,
		nil,
		self.ticket,
	)
	if err != nil {
		return nil, 0, err
	}

	return servicesFromResponse(res, self.serverEndpoint.Path)
}
