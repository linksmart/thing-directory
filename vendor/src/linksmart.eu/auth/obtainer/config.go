package obtainer

import (
	"errors"
	"net/url"
)

// Obtainer Config
type Conf struct {
	ServerAddr string `json:"serverAddr"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	ServiceID  string `json:"serviceID"`
}

func (c Conf) Validate() error {

	// Validate ServerAddr
	if c.ServerAddr == "" {
		return errors.New("Ticket Obtainer: Server address (serverAddr) is not specified.")
	}
	_, err := url.Parse(c.ServerAddr)
	if err != nil {
		return errors.New("Ticket Obtainer: Server address (serverAddr) is invalid: " + err.Error())
	}

	// Validate Username
	if c.Username == "" {
		return errors.New("Ticket Obtainer: Username (username) is not specified.")
	}

	// Validate ServiceID
	if c.ServiceID == "" {
		return errors.New("Ticket Obtainer: Service ID (serviceID) is not specified.")
	}

	return nil
}
