// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/linksmart/go-sec/auth/obtainer"
	"github.com/linksmart/service-catalog/v3/catalog"
	"github.com/linksmart/service-catalog/v3/utils"
)

// HTTPClient is the http client struct
type HTTPClient struct {
	serverEndpoint *url.URL
	ticket         *obtainer.Client
}

// FilterArgs are the filtering arguments
type FilterArgs struct {
	Path, Op, Value string
}

// NewHTTPClient creates a new HTTP client for SC's REST API
func NewHTTPClient(serverEndpoint string, ticket *obtainer.Client) (*HTTPClient, error) {

	endpointUrl, err := url.Parse(serverEndpoint)
	if err != nil {
		return nil, fmt.Errorf("error parsing endpoint: %s", err)
	}

	return &HTTPClient{
		serverEndpoint: endpointUrl,
		ticket:         ticket,
	}, nil
}

// Ping returns true if health endpoint responds OK
func (c *HTTPClient) Ping() (bool, error) {
	res, err := utils.HTTPRequest("GET",
		fmt.Sprintf("%v/health", c.serverEndpoint),
		nil,
		nil,
		c.ticket,
	)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		return true, nil
	} else {
		return false, fmt.Errorf(ErrorMsg(res))
	}
}

// Get gets a service
func (c *HTTPClient) Get(id string) (*catalog.Service, error) {
	res, err := utils.HTTPRequest("GET",
		fmt.Sprintf("%v/%v", c.serverEndpoint, id),
		nil,
		nil,
		c.ticket,
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusBadRequest:
		return nil, &catalog.BadRequestError{ErrorMsg(res)}
	case http.StatusConflict:
		return nil, &catalog.ConflictError{ErrorMsg(res)}
	case http.StatusNotFound:
		return nil, &catalog.NotFoundError{ErrorMsg(res)}
	default:
		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf(ErrorMsg(res))
		}
	}

	decoder := json.NewDecoder(res.Body)
	var s *catalog.Service
	err = decoder.Decode(&s)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// Post posts a service
func (c *HTTPClient) Post(service *catalog.Service) (*catalog.Service, error) {
	if service.ID != "" {
		return nil, fmt.Errorf("cannot POST a service with pre-defined ID. Use PUT method instead")
	}

	b, err := json.Marshal(service)
	if err != nil {
		return nil, err
	}

	res, err := utils.HTTPRequest("POST",
		c.serverEndpoint.String()+"/",
		map[string][]string{"Content-Type": {"application/json"}},
		bytes.NewReader(b),
		c.ticket,
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusBadRequest:
		return nil, &catalog.BadRequestError{ErrorMsg(res)}
	case http.StatusConflict:
		return nil, &catalog.ConflictError{ErrorMsg(res)}
	case http.StatusNotFound:
		return nil, &catalog.NotFoundError{ErrorMsg(res)}
	default:
		if res.StatusCode != http.StatusCreated {
			return nil, fmt.Errorf(ErrorMsg(res))
		}
	}

	decoder := json.NewDecoder(res.Body)
	var s *catalog.Service
	err = decoder.Decode(&s)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// Put puts a service
func (c *HTTPClient) Put(service *catalog.Service) (*catalog.Service, error) {
	if service.ID == "" {
		return nil, fmt.Errorf("cannot PUT a service without ID")
	}

	b, _ := json.Marshal(service)
	res, err := utils.HTTPRequest("PUT",
		fmt.Sprintf("%v/%v", c.serverEndpoint, service.ID),
		map[string][]string{"Content-Type": {"application/ld+json"}},
		bytes.NewReader(b),
		c.ticket,
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusBadRequest:
		return nil, &catalog.BadRequestError{ErrorMsg(res)}
	case http.StatusConflict:
		return nil, &catalog.ConflictError{ErrorMsg(res)}
	case http.StatusNotFound:
		return nil, &catalog.NotFoundError{ErrorMsg(res)}
	default:
		if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
			return nil, fmt.Errorf(ErrorMsg(res))
		}
	}

	decoder := json.NewDecoder(res.Body)
	var s *catalog.Service
	err = decoder.Decode(&s)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// Delete deletes a service
func (c *HTTPClient) Delete(id string) error {
	res, err := utils.HTTPRequest("DELETE",
		fmt.Sprintf("%v/%v", c.serverEndpoint, id),
		nil,
		bytes.NewReader([]byte{}),
		c.ticket,
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusBadRequest:
		return &catalog.BadRequestError{ErrorMsg(res)}
	case http.StatusConflict:
		return &catalog.ConflictError{ErrorMsg(res)}
	case http.StatusNotFound:
		return &catalog.NotFoundError{ErrorMsg(res)}
	default:
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf(ErrorMsg(res))
		}
	}

	return nil
}

// GetMany retrieves a page from the service collection
func (c *HTTPClient) GetMany(page, perPage int, filter *FilterArgs) ([]catalog.Service, int, error) {
	var err error
	var res *http.Response
	if filter == nil {
		res, err = utils.HTTPRequest("GET",
			fmt.Sprintf("%v?%v=%v&%v=%v",
				c.serverEndpoint, utils.GetParamPage, page, utils.GetParamPerPage, perPage),
			nil,
			nil,
			c.ticket,
		)
	} else {
		res, err = utils.HTTPRequest("GET",
			fmt.Sprintf("%v/%v/%v/%v?%v=%v&%v=%v",
				c.serverEndpoint, filter.Path, filter.Op, filter.Value, utils.GetParamPage, page, utils.GetParamPerPage, perPage),
			nil,
			nil,
			c.ticket,
		)
	}
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusBadRequest:
		return nil, 0, &catalog.BadRequestError{ErrorMsg(res)}
	case http.StatusConflict:
		return nil, 0, &catalog.ConflictError{ErrorMsg(res)}
	case http.StatusNotFound:
		return nil, 0, &catalog.NotFoundError{ErrorMsg(res)}
	default:
		if res.StatusCode != http.StatusOK {
			return nil, 0, fmt.Errorf(ErrorMsg(res))
		}
	}

	decoder := json.NewDecoder(res.Body)
	var coll catalog.Collection
	err = decoder.Decode(&coll)
	if err != nil {
		return nil, 0, err
	}

	return coll.Services, len(coll.Services), nil
}

// ErrorMsg extracts the message field of a resource.Error response
func ErrorMsg(res *http.Response) string {

	var e catalog.Error
	err := json.NewDecoder(res.Body).Decode(&e)
	if err != nil {
		return fmt.Sprintf("error decoding: %s", err)
	}
	return fmt.Sprintf("(%d) %s", res.StatusCode, e.Message)
}
