package validator

import (
	"errors"
	"net/url"

	"github.com/linksmart/go-sec/authz"
)

// Conf is a reference configuration struct for Validator
type Conf struct {
	// Enabled toggles validator client
	Enabled bool `json:"enabled"`
	// Provider is the authentication provider name
	Provider string `json:"provider"`
	// ProviderURL is the authentication provider URL
	ProviderURL string `json:"providerURL"`
	// ClientID is the authentication client id
	ClientID string `json:"clientID"`
	// BasicEnabled toggles the Basic Authentication
	BasicEnabled bool `json:"basicEnabled"`
	// Authz is the authorization config
	Authz authz.Conf `json:"authorization"`
}

// Validate validates the configuration object
func (c Conf) Validate() error {

	// Validate Provider
	if c.Provider == "" {
		return errors.New("auth provider name is not specified")
	}

	// Validate ProviderURL
	if c.ProviderURL == "" {
		return errors.New("auth provider URL is not specified")
	}
	_, err := url.Parse(c.ProviderURL)
	if err != nil {
		return errors.New("auth provider URL is invalid: " + err.Error())
	}

	// Validate ClientID
	if c.ClientID == "" {
		return errors.New("auth client ID is not specified")
	}

	// Validate Authorization
	if c.Authz.Enabled {
		if err := c.Authz.Validate(); err != nil {
			return errors.New("authz: " + err.Error())
		}
	}

	return nil
}
