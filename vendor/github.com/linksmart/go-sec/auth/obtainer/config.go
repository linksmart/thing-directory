package obtainer

import (
	"errors"
	"net/url"
)

// Conf is a reference configuration struct for Obtainer
type Conf struct {
	// Enabled is to toggle obtainer client
	Enabled bool `json:"enabled"`
	// Provider is the authentication provider name
	Provider string `json:"provider"`
	// ProviderURL is the authentication provider URL
	ProviderURL string `json:"providerURL"`
	// ClientID is the authentication client id.
	ClientID string `json:"clientID"`
	// Username is the client's username
	Username string `json:"username"`
	// Password is the client's password
	Password string `json:"password"`
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

	// Validate Username
	if c.Username == "" {
		return errors.New("auth username is not specified")
	}

	// Validate ClientID
	if c.ClientID == "" {
		return errors.New("auth client ID is not specified")
	}

	return nil
}
