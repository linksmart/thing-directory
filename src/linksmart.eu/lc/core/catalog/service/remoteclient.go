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

func (c *RemoteCatalogClient) Get(id string) (*Service, error) {
	res, err := catalog.HTTPRequest("GET",
		fmt.Sprintf("%v/%v", c.serverEndpoint, id),
		map[string][]string{"Content-Type": []string{"application/ld+json"}},
		nil,
		c.ticket,
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

	var s *Service
	err = decoder.Decode(&s)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (c *RemoteCatalogClient) Add(s *Service) error {
	b, _ := json.Marshal(s)
	res, err := catalog.HTTPRequest("POST",
		c.serverEndpoint.String()+"/",
		nil,
		bytes.NewReader(b),
		c.ticket,
	)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusCreated {
		return fmt.Errorf("Cannot add registration: %v", res.StatusCode)
	}
	return nil
}

func (c *RemoteCatalogClient) Update(id string, s *Service) error {
	b, _ := json.Marshal(s)
	res, err := catalog.HTTPRequest("PUT",
		fmt.Sprintf("%v/%v", c.serverEndpoint, id),
		nil,
		bytes.NewReader(b),
		c.ticket,
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

func (c *RemoteCatalogClient) Delete(id string) error {
	res, err := catalog.HTTPRequest("DELETE",
		fmt.Sprintf("%v/%v", c.serverEndpoint, id),
		nil,
		bytes.NewReader([]byte{}),
		c.ticket,
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

func (c *RemoteCatalogClient) List(page, perPage int) ([]Service, int, error) {
	res, err := catalog.HTTPRequest("GET",
		fmt.Sprintf("%v?%v=%v&%v=%v",
			c.serverEndpoint, catalog.GetParamPage, page, catalog.GetParamPerPage, perPage),
		nil,
		nil,
		c.ticket,
	)
	if err != nil {
		return nil, 0, err
	}

	decoder := json.NewDecoder(res.Body)
	defer res.Body.Close()

	var coll Collection
	err = decoder.Decode(&coll)
	if err != nil {
		return nil, 0, err
	}

	return coll.Services, len(coll.Services), nil
}

func (c *RemoteCatalogClient) Filter(path, op, value string, page, perPage int) ([]Service, int, error) {
	res, err := catalog.HTTPRequest("GET",
		fmt.Sprintf("%v/%v/%v/%v?%v=%v&%v=%v",
			c.serverEndpoint, path, op, value, catalog.GetParamPage, page, catalog.GetParamPerPage, perPage),
		nil,
		nil,
		c.ticket,
	)
	if err != nil {
		return nil, 0, err
	}

	decoder := json.NewDecoder(res.Body)
	defer res.Body.Close()

	var coll Collection
	err = decoder.Decode(&coll)
	if err != nil {
		return nil, 0, err
	}

	return coll.Services, len(coll.Services), nil
}
