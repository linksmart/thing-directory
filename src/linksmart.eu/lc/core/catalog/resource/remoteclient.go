package resource

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	auth "linksmart.eu/auth/obtainer"
)

type RemoteCatalogClient struct {
	serverEndpoint *url.URL
	ticketClient   *auth.Client
}

func deviceFromResponse(res *http.Response, apiLocation string) (*Device, error) {
	decoder := json.NewDecoder(res.Body)
	defer res.Body.Close()

	var d Device
	err := decoder.Decode(&d)
	if err != nil {
		return nil, err
	}
	d = d.unLdify(apiLocation)
	return &d, nil
}

func devicesFromResponse(res *http.Response, apiLocation string) ([]Device, int, error) {
	decoder := json.NewDecoder(res.Body)
	defer res.Body.Close()

	var coll Collection
	err := decoder.Decode(&coll)
	if err != nil {
		return nil, 0, err
	}

	devs := make([]Device, 0, len(coll.Devices))
	for k, v := range coll.Devices {
		d := *v.Device
		for _, res := range coll.Resources {
			if res.Device == k {
				d.Resources = append(d.Resources, res)
			}
		}
		devs = append(devs, d.unLdify(apiLocation))
	}

	return devs, len(coll.Devices), nil
}

func resourceFromResponse(res *http.Response, apiLocation string) (*Resource, error) {
	decoder := json.NewDecoder(res.Body)
	defer res.Body.Close()

	var r Resource
	err := decoder.Decode(&r)
	if err != nil {
		return nil, err
	}
	r = r.unLdify(apiLocation)
	return &r, nil
}

func resourcesFromResponse(res *http.Response, apiLocation string) ([]Resource, int, error) {
	decoder := json.NewDecoder(res.Body)
	defer res.Body.Close()

	var coll Collection
	err := decoder.Decode(&coll)
	if err != nil {
		return nil, 0, err
	}

	ress := make([]Resource, 0, len(coll.Resources))
	for _, r := range coll.Resources {
		ress = append(ress, r.unLdify(apiLocation))
	}

	return ress, len(coll.Resources), nil
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

func (self *RemoteCatalogClient) Get(id string) (*Device, error) {
	res, err := self.httpClient("GET", fmt.Sprintf("%v/%v", self.serverEndpoint, id), nil, nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == http.StatusNotFound {
		return nil, ErrorNotFound
	} else if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%v", res.StatusCode)
	}
	return deviceFromResponse(res, self.serverEndpoint.Path)
}

func (self *RemoteCatalogClient) Add(d *Device) error {
	b, _ := json.Marshal(d)
	_, err := self.httpClient("POST", self.serverEndpoint.String()+"/",
		bytes.NewReader(b),
		map[string]string{"Content-Type": "application/ld+json"},
	)
	if err != nil {
		return err
	}
	return nil
}

func (self *RemoteCatalogClient) Update(id string, d *Device) error {
	b, _ := json.Marshal(d)
	res, err := self.httpClient("PUT", fmt.Sprintf("%v/%v", self.serverEndpoint, id), bytes.NewReader(b), nil)
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
	res, err := self.httpClient("DELETE", fmt.Sprintf("%v/%v", self.serverEndpoint, id), bytes.NewReader([]byte{}), nil)
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

func (self *RemoteCatalogClient) GetDevices(page int, perPage int) ([]Device, int, error) {
	res, err := self.httpClient("GET",
		fmt.Sprintf("%v?%v=%v&%v=%v",
			self.serverEndpoint, GetParamPage, page, GetParamPerPage, perPage), nil, nil)
	if err != nil {
		return nil, 0, err
	}

	return devicesFromResponse(res, self.serverEndpoint.Path)
}

func (self *RemoteCatalogClient) FindDevice(path, op, value string) (*Device, error) {
	res, err := self.httpClient("GET", fmt.Sprintf("%v/%v/%v/%v/%v", self.serverEndpoint, FTypeDevice, path, op, value), nil, nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == http.StatusNotFound {
		return nil, ErrorNotFound
	} else if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%v", res.StatusCode)
	}
	return deviceFromResponse(res, self.serverEndpoint.Path)
}

func (self *RemoteCatalogClient) FindDevices(path, op, value string, page, perPage int) ([]Device, int, error) {
	res, err := self.httpClient("GET",
		fmt.Sprintf("%v/%v/%v/%v/%v?%v=%v&%v=%v",
			self.serverEndpoint, FTypeDevices, path, op, value, GetParamPage, page, GetParamPerPage, perPage), nil, nil)
	if err != nil {
		return nil, 0, err
	}

	return devicesFromResponse(res, self.serverEndpoint.Path)
}

func (self *RemoteCatalogClient) FindResource(path, op, value string) (*Resource, error) {
	res, err := self.httpClient("GET", fmt.Sprintf("%v/%v/%v/%v/%v", self.serverEndpoint, FTypeResource, path, op, value), nil, nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == http.StatusNotFound {
		return nil, ErrorNotFound
	} else if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%v", res.StatusCode)
	}
	return resourceFromResponse(res, self.serverEndpoint.Path)
}

func (self *RemoteCatalogClient) FindResources(path, op, value string, page, perPage int) ([]Resource, int, error) {
	res, err := self.httpClient("GET",
		fmt.Sprintf("%v/%v/%v/%v/%v?%v=%v&%v=%v",
			self.serverEndpoint, FTypeResources, path, op, value, GetParamPage, page, GetParamPerPage, perPage), nil, nil)
	if err != nil {
		return nil, 0, err
	}

	return resourcesFromResponse(res, self.serverEndpoint.Path)
}
