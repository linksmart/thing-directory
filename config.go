// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"

	"github.com/kelseyhightower/envconfig"
	"github.com/linksmart/go-sec/authz"
)

const (
	DNSSDServiceType = "_linksmart-rc._tcp"
	BackendMemory    = "memory"
	BackendLevelDB   = "leveldb"
)

type Config struct {
	ServiceID      string         `json:"serviceID"`
	Description    string         `json:"description"`
	PublicEndpoint string         `json:"publicEndpoint"`
	BindAddr       string         `json:"bindAddr"`
	BindPort       int            `json:"bindPort"`
	DnssdEnabled   bool           `json:"dnssdEnabled"`
	Storage        StorageConfig  `json:"storage"`
	ServiceCatalog ServiceCatalog `json:"serviceCatalog"`
	Auth           ValidatorConf  `json:"auth"`
}

type ServiceCatalog struct {
	Enabled  bool         `json:"enabled"`
	Discover bool         `json:"discover"`
	Endpoint string       `json:"endpoint"`
	Ttl      int          `json:"ttl"`
	Auth     ObtainerConf `json:"auth"`
}

type StorageConfig struct {
	Type string `json:"type"`
	DSN  string `json:"dsn"`
}

var supportedBackends = map[string]bool{
	BackendMemory:  false,
	BackendLevelDB: true,
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

	if c.ServiceCatalog.Enabled {
		if c.ServiceCatalog.Endpoint == "" && c.ServiceCatalog.Discover == false {
			err = fmt.Errorf("All ServiceCatalog entries must have either endpoint or a discovery flag defined")
		}
		if c.ServiceCatalog.Ttl <= 0 {
			err = fmt.Errorf("All ServiceCatalog entries must have TTL >= 0")
		}
		if c.ServiceCatalog.Auth.Enabled {
			// Validate ticket obtainer config
			err = c.ServiceCatalog.Auth.Validate()
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

	var config Config
	err = json.Unmarshal(file, &config)
	if err != nil {
		return nil, err
	}

	// Override loaded values with environment variables
	err = envconfig.Process("td", &config)
	if err != nil {
		return nil, err
	}

	if err = config.Validate(); err != nil {
		return nil, err
	}
	return &config, nil
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
	// Auth switch
	Enabled bool `json:"enabled"`
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
