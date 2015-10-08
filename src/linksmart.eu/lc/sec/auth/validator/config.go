package validator

import (
	"errors"
	"net/url"
)

// Validator Config
type Conf struct {
	// Auth switch
	Enabled bool `json:"enabled"`
	// Authentication server address
	ServerAddr string `json:"serverAddr"`
	// Service ID
	ServiceID string `json:"serviceID"`
	// Authorization rules, if any
	AuthorizationRules []Rule `json:"authorization"`
}

// Validator Config Rule
type Rule struct {
	Resources []string `json:"resources"`
	Methods   []string `json:"methods"`
	Users     []string `json:"users"`
	Groups    []string `json:"groups"`
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

	// Validate AuthorizationRules
	if len(c.AuthorizationRules) == 0 {
		return errors.New("Ticket Validator: At least one authorization rule must be defined.")
	}
	for _, rule := range c.AuthorizationRules {
		if len(rule.Resources) == 0 {
			return errors.New("Ticket Validator: No resources in an authorization rule.")
		}
		if len(rule.Methods) == 0 {
			return errors.New("Ticket Validator: No methods in an authorization rule.")
		}
		if len(rule.Users) == 0 && len(rule.Groups) == 0 {
			return errors.New(
				"Ticket Validator: At least one user or group must be assigned to each authorization rule.")
		}
	}

	return nil
}
