package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	auth "linksmart.eu/lc/sec/auth/obtainer"
)

type RemoteCatalogClient struct {
	serverEndpoint *url.URL
	ticketClient   *auth.Client
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

func NewRemoteCatalogClient(serverEndpoint string, ticketClient *auth.Client) *RemoteCatalogClient {
	// Check if serverEndpoint is a correct URL
	endpointUrl, err := url.Parse(serverEndpoint)
	if err != nil {
		return &RemoteCatalogClient{}
	}

	return &RemoteCatalogClient{
		serverEndpoint: endpointUrl,
		ticketClient:   ticketClient,
	}
}

// Manually submit an HTTP request and get the response
func (self *RemoteCatalogClient) httpClient(method string, url string,
	body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	// Set headers
	for key, val := range headers {
		req.Header.Set(key, val)
	}

	// If ticketClient is instantiated, service requires auth
	if self.ticketClient != nil {
		// Set auth header and send the request
		req.Header.Set("X-Auth-Token", self.ticketClient.Ticket())
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		if res != nil {
			if res.StatusCode == http.StatusUnauthorized {
				// Get a new ticket and retry again
				logger.Println("httpClient() Invalid authentication ticket.")
				ticket, err := self.ticketClient.Renew()
				if err != nil {
					return nil, err
				}
				logger.Println("httpClient() Renewed ticket.")

				// Reset the header and try again
				req.Header.Set("X-Auth-Token", ticket)
				res, err := http.DefaultClient.Do(req)
				if err != nil {
					return nil, err
				}
				return res, nil
			}
		}
		return res, nil
	}

	// No auth
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (self *RemoteCatalogClient) Get(id string) (*Service, error) {
	res, err := self.httpClient(
		"GET",
		fmt.Sprintf("%v/%v", self.serverEndpoint, id),
		nil,
		map[string]string{"Content-Type": "application/ld+json"},
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
	_, err := self.httpClient(
		"POST",
		self.serverEndpoint.String()+"/",
		bytes.NewReader(b),
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

func (self *RemoteCatalogClient) Update(id string, s *Service) error {
	b, _ := json.Marshal(s)
	res, err := self.httpClient(
		"PUT",
		fmt.Sprintf("%v/%v", self.serverEndpoint, id),
		bytes.NewReader(b),
		nil,
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
	res, err := self.httpClient(
		"DELETE",
		fmt.Sprintf("%v/%v", self.serverEndpoint, id),
		bytes.NewReader([]byte{}),
		nil,
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
	res, err := self.httpClient(
		"GET",
		fmt.Sprintf("%v?%v=%v&%v=%v",
			self.serverEndpoint, GetParamPage, page, GetParamPerPage, perPage),
		nil,
		nil,
	)
	if err != nil {
		return nil, 0, err
	}

	return servicesFromResponse(res, self.serverEndpoint.Path)
}

func (self *RemoteCatalogClient) FindService(path, op, value string) (*Service, error) {
	res, err := self.httpClient(
		"GET",
		fmt.Sprintf("%v/%v/%v/%v/%v", self.serverEndpoint, FTypeService, path, op, value),
		nil,
		nil,
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
	res, err := self.httpClient(
		"GET",
		fmt.Sprintf("%v/%v/%v/%v/%v?%v=%v&%v=%v",
			self.serverEndpoint, FTypeServices, path, op, value, GetParamPage, page, GetParamPerPage, perPage),
		nil,
		nil,
	)
	if err != nil {
		return nil, 0, err
	}

	return servicesFromResponse(res, self.serverEndpoint.Path)
}
