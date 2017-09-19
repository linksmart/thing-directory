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
	"code.linksmart.eu/sc/service-catalog/service"
)

type Config struct {
	Description  string        `json:"description"`
	DnssdEnabled bool          `json:"dnssdEnabled"`
	BindAddr     string        `json:"bindAddr"`
	BindPort     int           `json:"bindPort"`
	ApiLocation  string        `json:"apiLocation"`
	StaticDir    string        `json:"staticDir"`
	Storage      StorageConfig `json:"storage"`
	GC           GCConfig      `js:"gc"`
	Auth         ValidatorConf `json:"auth"`
}

type StorageConfig struct {
	Type string `json:"type"`
	DSN  string `json:"dsn"`
}

var supportedBackends = map[string]bool{
	service.CatalogBackendMemory:  true,
	service.CatalogBackendLevelDB: true,
}

// GCConfig describes configuration of the GlobalConnect
type GCConfig struct {
	// URL of the Tunneling Service endpoint (aka NM REST API)
	TunnelingService string `json:"tunnelingService"`
}

func (c *Config) Validate() error {
	var err error
	if c.BindAddr == "" || c.BindPort == 0 {
		err = fmt.Errorf("Empty host or port")
	}
	if !supportedBackends[c.Storage.Type] {
		err = fmt.Errorf("Unsupported storage backend")
	}
	_, err = url.Parse(c.Storage.DSN)
	if err != nil {
		err = fmt.Errorf("storage DSN should be a valid URL")
	}
	if c.StaticDir == "" {
		err = fmt.Errorf("staticDir must be defined")
	}
	if strings.HasSuffix(c.StaticDir, "/") {
		err = fmt.Errorf("staticDir must not have a trailing slash")
	}
	if c.GC.TunnelingService != "" {
		_, err := url.Parse(c.GC.TunnelingService)
		if err != nil {
			err = fmt.Errorf("gc tunnelingService must be a valid URL")
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

func loadConfig(confPath string) (*Config, error) {
	file, err := ioutil.ReadFile(confPath)
	if err != nil {
		return nil, err
	}

	config := new(Config)
	err = json.Unmarshal(file, config)
	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(config.ApiLocation, "/") {
		config.ApiLocation = strings.TrimSuffix(config.ApiLocation, "/")
	}

	if err = config.Validate(); err != nil {
		return nil, err
	}
	return config, nil
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
