// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package obtainer

import (
	"sync"
)

type Client struct {
	obtainer *Obtainer
	username string
	password string
	clientID string
	token    interface{}
	sync.Mutex
}

func NewClient(providerName, providerURL, username, password, clientID string) (*Client, error) {
	// Setup obtainer
	o, err := Setup(providerName, providerURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		obtainer: o,
		username: username,
		password: password,
		clientID: clientID,
	}, nil
}

// Obtain obtains a new token and returns the token string. If token is already available, it just returns the token string
func (c *Client) Obtain() (tokenString string, err error) {
	c.Lock()
	defer c.Unlock()

	if c.token == nil {
		token, err := c.obtainer.ObtainToken(c.username, c.password, c.clientID)
		if err != nil {
			return "", err
		}
		c.token = token
	}
	return c.obtainer.TokenString(c.token)
}

// Renew renews the token and returns the token string
func (c *Client) Renew() (tokenString string, err error) {
	c.Lock()
	defer c.Unlock()

	token, err := c.obtainer.RenewToken(c.token, c.clientID)
	if err != nil {
		// could not renew, try to obtain a new one
		return c.Obtain()
	}
	c.token = token

	return c.obtainer.TokenString(c.token)
}

// Revoke revokes the token
func (c *Client) Revoke() error {
	c.Lock()
	defer c.Unlock()

	err := c.obtainer.RevokeToken(c.token)
	if err != nil {
		return err
	}
	c.token = nil

	return nil
}
