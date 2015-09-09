package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"

	"linksmart.eu/auth/validator"
	utils "linksmart.eu/lc/core/catalog"
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
	// Auth config
	Auth validator.Conf `json:"auth"`
}

type StorageConfig struct {
	Type string `json:"type"`
}

var supportedBackends = map[string]bool{
	utils.CatalogBackendMemory: true,
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

	if err = config.Validate(); err != nil {
		return nil, err
	}
	return config, nil
}
