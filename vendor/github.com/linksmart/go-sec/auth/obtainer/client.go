// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package obtainer

import (
	"sync"
)

type Client struct {
	obtainer  *Obtainer
	username  string
	password  string
	clientID string
	tgt       string
	ticket    string
	sync.Mutex
}

func NewClient(providerName, providerURL, username, password, clientID string) (*Client, error) {
	// Setup obtainer
	o, err := Setup(providerName, providerURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		obtainer:  o,
		username:  username,
		password:  password,
		clientID: clientID,
	}, nil
}

// Obtain the ticket, create one if it's not available
func (c *Client) Obtain() (string, error) {
	c.Lock()
	defer c.Unlock()

	if c.ticket == "" {
		// Get Ticket Granting Ticket
		TGT, err := c.obtainer.Login(c.username, c.password, c.clientID)
		if err != nil {
			return "", err
		}
		c.tgt = TGT

		// Get Service Ticket
		ticket, err := c.obtainer.RequestTicket(TGT, c.clientID)
		if err != nil {
			return "", err
		}
		c.ticket = ticket

	}

	return c.ticket, nil
}

// Renew the ticket
func (c *Client) Renew() (string, error) {
	c.Lock()
	defer c.Unlock()

	// Renew Service Ticket using previous TGT
	ticket, err := c.obtainer.RequestTicket(c.tgt, c.clientID)
	if err != nil {
		// Get a new Ticket Granting Ticket
		TGT, err := c.obtainer.Login(c.username, c.password, c.clientID)
		if err != nil {
			return "", err
		}
		c.tgt = TGT

		// Get Service Ticket
		ticket, err = c.obtainer.RequestTicket(TGT, c.clientID)
		if err != nil {
			return "", err
		}
	}
	c.ticket = ticket

	return c.ticket, nil
}

// Delete the ticket granting ticket
func (c *Client) Delete() error {
	c.Lock()
	defer c.Unlock()

	err := c.obtainer.Logout(c.tgt)
	if err != nil {
		return err
	}
	c.tgt = ""
	c.ticket = ""

	return nil
}
