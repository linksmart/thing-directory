package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"

	"linksmart.eu/auth/obtainer"
	"linksmart.eu/auth/validator"
	utils "linksmart.eu/lc/core/catalog"
)

type Config struct {
	Description    string           `json:"description"`
	PublicEndpoint string           `json:"publicEndpoint"`
	BindAddr       string           `json:"bindAddr"`
	BindPort       int              `json:"bindPort"`
	DnssdEnabled   bool             `json:"dnssdEnabled"`
	StaticDir      string           `json:"staticDir"`
	ApiLocation    string           `json:"apiLocation"`
	Storage        StorageConfig    `json:"storage"`
	ServiceCatalog []ServiceCatalog `json:"serviceCatalog"`
	// Auth config
	Auth validator.Conf `json:"auth"`
}

type ServiceCatalog struct {
	Discover bool
	Endpoint string
	Ttl      int
	Auth     *obtainer.Conf `json:"auth"`
}

type StorageConfig struct {
	Type string `json:"type"`
}

var supportedBackends = map[string]bool{
	utils.CatalogBackendMemory: true,
}

func (c *Config) Validate() error {
	var err error
	if c.BindAddr == "" || c.BindPort == 0 || c.PublicEndpoint == "" {
		err = fmt.Errorf("BindAddr, BindPort, and PublicEndpoint have to be defined")
	}
	_, err = url.Parse(c.PublicEndpoint)
	if err != nil {
		err = fmt.Errorf("PublicEndpoint should be a valid URL")
	}
	if !supportedBackends[c.Storage.Type] {
		err = fmt.Errorf("Unsupported storage backend")
	}
	if c.ApiLocation == "" {
		err = fmt.Errorf("apiLocation must be defined")
	}
	if c.StaticDir == "" {
		err = fmt.Errorf("staticDir must be defined")
	}
	if strings.HasSuffix(c.ApiLocation, "/") {
		err = fmt.Errorf("apiLocation must not have a training slash")
	}
	if strings.HasSuffix(c.StaticDir, "/") {
		err = fmt.Errorf("staticDir must not have a training slash")
	}
	for _, cat := range c.ServiceCatalog {
		if cat.Endpoint == "" && cat.Discover == false {
			err = fmt.Errorf("All ServiceCatalog entries must have either endpoint or a discovery flag defined")
		}
		if cat.Ttl <= 0 {
			err = fmt.Errorf("All ServiceCatalog entries must have TTL >= 0")
		}
		if cat.Auth != nil {
			// Validate ticket obtainer config
			err = cat.Auth.Validate()
			if err != nil {
				return err
			}
		}
	}

	if c.Auth.Enabled {
		// Validate ticket validator config
		err = c.Auth.Validate()
		if err != nil {
			return err
		}
	}

	return err
}

func loadConfig(path string) (*Config, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	c := new(Config)
	err = json.Unmarshal(file, c)
	if err != nil {
		return nil, err
	}

	if err = c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}
