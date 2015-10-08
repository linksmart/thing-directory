package validator

import (
	"errors"
	"net/url"

	"linksmart.eu/lc/sec/authz"
)

// Validator Config
type Conf struct {
	// Auth switch
	Enabled bool `json:"enabled"`
	// Authentication server address
	ServerAddr string `json:"serverAddr"`
	// Service ID
	ServiceID string `json:"serviceID"`
	// Authorization config
	Authz authz.Conf `json:"authorization"`
}

func (c Conf) Validate() error {

	// Validate ServerAddr
	if c.ServerAddr == "" {
		return errors.New("Ticket Validator: Server address (serverAddr) is not specified.")
	}
	_, err := url.Parse(c.ServerAddr)
	if err != nil {
		return errors.New("Ticket Validator: Server address (serverAddr) is invalid: " + err.Error())
	}

	// Validate ServiceID
	if c.ServiceID == "" {
		return errors.New("Ticket Validator: Service ID (serviceID) is not specified.")
	}

	// Validate Authorization
	if c.Authz.Enabled {
		if err := c.Authz.Validate(); err != nil {
			return err
		}
	}

	return nil
}
