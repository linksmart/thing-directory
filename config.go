// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"

	"code.linksmart.eu/com/go-sec/authz"
	utils "code.linksmart.eu/rc/resource-catalog/catalog"
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
	Auth           ValidatorConf    `json:"auth"`
}

type ServiceCatalog struct {
	Discover bool          `json:"discover"`
	Endpoint string        `json:"endpoint"`
	Ttl      int           `json:"ttl"`
	Auth     *ObtainerConf `json:"auth"`
}

type StorageConfig struct {
	Type string `json:"type"`
	DSN  string `json:"dsn"`
}

var supportedBackends = map[string]bool{
	utils.CatalogBackendMemory:  true,
	utils.CatalogBackendLevelDB: true,
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
	_, err = url.Parse(c.Storage.DSN)
	if err != nil {
		err = fmt.Errorf("storage DSN should be a valid URL")
	}
	if !supportedBackends[c.Storage.Type] {
		err = fmt.Errorf("Unsupported storage backend")
	}
	if c.StaticDir == "" {
		err = fmt.Errorf("staticDir must be defined")
	}
	if strings.HasSuffix(c.StaticDir, "/") {
		err = fmt.Errorf("staticDir must not have a trailing slash")
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

	if strings.HasSuffix(c.ApiLocation, "/") {
		c.ApiLocation = strings.TrimSuffix(c.ApiLocation, "/")
	}

	if err = c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

// Ticket Validator Config
type ValidatorConf struct {
	// Auth switch
	Enabled bool `json:"enabled"`
	// Authentication provider name
	Provider string `json:"provider"`
	// Authentication provider URL
	ProviderURL string `json:"providerURL"`
	// Service ID
	ServiceID string `json:"serviceID"`
	// Basic Authentication switch
	BasicEnabled bool `json:"basicEnabled"`
	// Authorization config
	Authz *authz.Conf `json:"authorization"`
}

func (c ValidatorConf) Validate() error {

	// Validate Provider
	if c.Provider == "" {
		return errors.New("Ticket Validator: Auth provider name (provider) is not specified.")
	}

	// Validate ProviderURL
	if c.ProviderURL == "" {
		return errors.New("Ticket Validator: Auth provider URL (providerURL) is not specified.")
	}
	_, err := url.Parse(c.ProviderURL)
	if err != nil {
		return errors.New("Ticket Validator: Auth provider URL (providerURL) is invalid: " + err.Error())
	}

	// Validate ServiceID
	if c.ServiceID == "" {
		return errors.New("Ticket Validator: Auth Service ID (serviceID) is not specified.")
	}

	// Validate Authorization
	if c.Authz != nil {
		if err := c.Authz.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Ticket Obtainer Client Config
type ObtainerConf struct {
	// Authentication provider name
	Provider string `json:"provider"`
	// Authentication provider URL
	ProviderURL string `json:"providerURL"`
	// Service ID
	ServiceID string `json:"serviceID"`
	// User credentials
	Username string `json:"username"`
	Password string `json:"password"`
}

func (c ObtainerConf) Validate() error {

	// Validate Provider
	if c.Provider == "" {
		return errors.New("Ticket Obtainer: Auth provider name (provider) is not specified.")
	}

	// Validate ProviderURL
	if c.ProviderURL == "" {
		return errors.New("Ticket Obtainer: Auth provider URL (ProviderURL) is not specified.")
	}
	_, err := url.Parse(c.ProviderURL)
	if err != nil {
		return errors.New("Ticket Obtainer: Auth provider URL (ProviderURL) is invalid: " + err.Error())
	}

	// Validate Username
	if c.Username == "" {
		return errors.New("Ticket Obtainer: Auth Username (username) is not specified.")
	}

	// Validate ServiceID
	if c.ServiceID == "" {
		return errors.New("Ticket Obtainer: Auth Service ID (serviceID) is not specified.")
	}

	return nil
}
