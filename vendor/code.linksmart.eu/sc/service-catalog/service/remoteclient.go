// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"code.linksmart.eu/com/go-sec/auth/obtainer"
	"code.linksmart.eu/sc/service-catalog/utils"
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
		return nil, &BadRequestError{ErrorMsg(res)}
	case http.StatusConflict:
		return nil, &ConflictError{ErrorMsg(res)}
	case http.StatusNotFound:
		return nil, &NotFoundError{ErrorMsg(res)}
	default:
		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf(ErrorMsg(res))
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
		res, err = utils.HTTPRequest("POST",
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
		resGet, err := utils.HTTPRequest("GET",
			fmt.Sprintf("%v/%v", c.serverEndpoint, id),
			nil,
			nil,
			c.ticket,
		)
		if err != nil {
			return "", err
		}
		defer resGet.Body.Close()

		// Make sure registration is not found
		// catch every status but http.StatusNotFound
		switch resGet.StatusCode {
		case http.StatusOK:
			return "", &ConflictError{fmt.Sprintf("Device id %s is not unique.", id)}
		case http.StatusBadRequest:
			return "", &BadRequestError{ErrorMsg(resGet)}
		case http.StatusConflict:
			return "", &ConflictError{ErrorMsg(resGet)}
		default:
			if resGet.StatusCode != http.StatusNotFound {
				return "", fmt.Errorf(ErrorMsg(resGet))
			}
		}

		// Now add
		res, err = utils.HTTPRequest("PUT",
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
			return "", fmt.Errorf(ErrorMsg(res))
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
	resGet, err := utils.HTTPRequest("GET",
		fmt.Sprintf("%v/%v", c.serverEndpoint, id),
		nil,
		nil,
		c.ticket,
	)
	if err != nil {
		return err
	}
	defer resGet.Body.Close()

	switch resGet.StatusCode {
	case http.StatusBadRequest:
		return &BadRequestError{ErrorMsg(resGet)}
	case http.StatusConflict:
		return &ConflictError{ErrorMsg(resGet)}
	case http.StatusNotFound:
		return &NotFoundError{ErrorMsg(resGet)}
	default:
		if resGet.StatusCode != http.StatusOK {
			return fmt.Errorf(ErrorMsg(resGet))
		}
	}

	b, _ := json.Marshal(s)
	res, err := utils.HTTPRequest("PUT",
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
			return fmt.Errorf(ErrorMsg(res))
		}
	}

	return nil
}

// Deletes a service
func (c *RemoteCatalogClient) Delete(id string) error {
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
		return &BadRequestError{ErrorMsg(res)}
	case http.StatusConflict:
		return &ConflictError{ErrorMsg(res)}
	case http.StatusNotFound:
		return &NotFoundError{ErrorMsg(res)}
	default:
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf(ErrorMsg(res))
		}
	}

	return nil
}

// Retrieves a page from the service collection
func (c *RemoteCatalogClient) List(page, perPage int) ([]Service, int, error) {
	res, err := utils.HTTPRequest("GET",
		fmt.Sprintf("%v?%v=%v&%v=%v",
			c.serverEndpoint, utils.GetParamPage, page, utils.GetParamPerPage, perPage),
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
			return nil, 0, fmt.Errorf(ErrorMsg(res))
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
	res, err := utils.HTTPRequest("GET",
		fmt.Sprintf("%v/%v/%v/%v?%v=%v&%v=%v",
			c.serverEndpoint, path, op, value, utils.GetParamPage, page, utils.GetParamPerPage, perPage),
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
			return nil, 0, fmt.Errorf(ErrorMsg(res))
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
	return fmt.Sprintf("(%d) %s", res.StatusCode, e.Message)
}
