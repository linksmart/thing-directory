package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"linksmart.eu/lc/core/catalog"
	"linksmart.eu/lc/sec/auth/obtainer"
	"strings"
)

type RemoteCatalogClient struct {
	serverEndpoint *url.URL
	ticket         *obtainer.Client
}

func NewRemoteCatalogClient(serverEndpoint string, ticket *obtainer.Client) (CatalogClient, error) {
	// Check if serverEndpoint is a correct URL
	endpointUrl, err := url.Parse(serverEndpoint)
	if err != nil {
		return nil, fmt.Errorf("Error parsing catalog endpoint url: %s", err)
	}

	return &RemoteCatalogClient{
		serverEndpoint: endpointUrl,
		ticket:         ticket,
	}, nil
}

// Retrieves a service
func (c *RemoteCatalogClient) Get(id string) (*Service, error) {
	res, err := catalog.HTTPRequest("GET",
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
		return nil, &BadRequestError{ErrorMsg(res)}
	case http.StatusConflict:
		return nil, &ConflictError{ErrorMsg(res)}
	case http.StatusNotFound:
		return nil, &NotFoundError{ErrorMsg(res)}
	default:
		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Error getting service: %v", ErrorMsg(res))
		}
	}

	decoder := json.NewDecoder(res.Body)
	var s *Service
	err = decoder.Decode(&s)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// Adds a service
func (c *RemoteCatalogClient) Add(s *Service) (string, error) {
	id := s.Id
	service := *s
	service.Id = ""
	b, _ := json.Marshal(service)

	var (
		res *http.Response
		err error
	)

	if id == "" { // Let the system generate an id
		res, err = catalog.HTTPRequest("POST",
			c.serverEndpoint.String()+"/",
			map[string][]string{"Content-Type": []string{"application/ld+json"}},
			bytes.NewReader(b),
			c.ticket,
		)
		if err != nil {
			return "", err
		}
		defer res.Body.Close()

	} else { // User-defined id

		// Check if id is unique
		res, err = catalog.HTTPRequest("GET",
			fmt.Sprintf("%v/%v", c.serverEndpoint, id),
			nil,
			nil,
			c.ticket,
		)
		if err != nil {
			return "", err
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusNotFound {
			return "", &ConflictError{fmt.Sprintf("Device id %s is not unique.", id)}
		}

		// Now add
		res, err = catalog.HTTPRequest("PUT",
			fmt.Sprintf("%v/%v", c.serverEndpoint, id),
			map[string][]string{"Content-Type": []string{"application/ld+json"}},
			bytes.NewReader(b),
			c.ticket,
		)
		if err != nil {
			return "", err
		}
		defer res.Body.Close()
	}

	switch res.StatusCode {
	case http.StatusBadRequest:
		return "", &BadRequestError{ErrorMsg(res)}
	case http.StatusConflict:
		return "", &ConflictError{ErrorMsg(res)}
	case http.StatusNotFound:
		return "", &NotFoundError{ErrorMsg(res)}
	default:
		if res.StatusCode != http.StatusCreated {
			return "", fmt.Errorf("Error adding service: %v", ErrorMsg(res))
		}
	}

	location, err := res.Location()
	if err != nil {
		return "", err
	}
	id = strings.SplitAfterN(location.String(), "", 2)[1]

	return id, nil
}

// Updates a service
func (c *RemoteCatalogClient) Update(id string, s *Service) error {
	// Check if id is found
	res, err := catalog.HTTPRequest("GET",
		fmt.Sprintf("%v/%v", c.serverEndpoint, id),
		nil,
		nil,
		c.ticket,
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return &NotFoundError{fmt.Sprintf("Service with id %s is not found.", id)}
	}

	b, _ := json.Marshal(s)
	res, err = catalog.HTTPRequest("PUT",
		fmt.Sprintf("%v/%v", c.serverEndpoint, id),
		map[string][]string{"Content-Type": []string{"application/ld+json"}},
		bytes.NewReader(b),
		c.ticket,
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusBadRequest:
		return &BadRequestError{ErrorMsg(res)}
	case http.StatusConflict:
		return &ConflictError{ErrorMsg(res)}
	case http.StatusNotFound:
		return &NotFoundError{ErrorMsg(res)}
	default:
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("Error updating service: %v", ErrorMsg(res))
		}
	}

	return nil
}

// Deletes a service
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
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusBadRequest:
		return &BadRequestError{ErrorMsg(res)}
	case http.StatusConflict:
		return &ConflictError{ErrorMsg(res)}
	case http.StatusNotFound:
		return &NotFoundError{ErrorMsg(res)}
	default:
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("Error deleting service: %v", ErrorMsg(res))
		}
	}

	return nil
}

// Retrieves a page from the service collection
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
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusBadRequest:
		return nil, 0, &BadRequestError{ErrorMsg(res)}
	case http.StatusConflict:
		return nil, 0, &ConflictError{ErrorMsg(res)}
	case http.StatusNotFound:
		return nil, 0, &NotFoundError{ErrorMsg(res)}
	default:
		if res.StatusCode != http.StatusOK {
			return nil, 0, fmt.Errorf("Error listing services: %v", ErrorMsg(res))
		}
	}

	decoder := json.NewDecoder(res.Body)
	var coll Collection
	err = decoder.Decode(&coll)
	if err != nil {
		return nil, 0, err
	}

	return coll.Services, len(coll.Services), nil
}

// Filter services
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
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusBadRequest:
		return nil, 0, &BadRequestError{ErrorMsg(res)}
	case http.StatusConflict:
		return nil, 0, &ConflictError{ErrorMsg(res)}
	case http.StatusNotFound:
		return nil, 0, &NotFoundError{ErrorMsg(res)}
	default:
		if res.StatusCode != http.StatusOK {
			return nil, 0, fmt.Errorf("Error filtering services: %v", ErrorMsg(res))
		}
	}

	decoder := json.NewDecoder(res.Body)
	var coll Collection
	err = decoder.Decode(&coll)
	if err != nil {
		return nil, 0, err
	}

	return coll.Services, len(coll.Services), nil
}

// Returns the message field of a resource.Error response
func ErrorMsg(res *http.Response) string {
	decoder := json.NewDecoder(res.Body)

	var e Error
	err := decoder.Decode(&e)
	if err != nil {
		return res.Status
	}
	return e.Message
}
