// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

// Package obtainer provides an interface for OpenID Connect token obtainment from a provider
package obtainer

import (
	"fmt"
	"sync"
)

// Interface methods to login, obtain Service Ticket, and logout
type Driver interface {
	// ObtainToken requests a token in exchange for user credentials
	ObtainToken(serverAddr string, username, password, clientID string) (token interface{}, err error)
	// TokenString returns the string part of token object (e.g. access_token, id_token strings)
	TokenString(token interface{}) (tokenString string, err error)
	// RenewToken renews the token (when applicable) using information inside the token (e.g. refresh_token)
	RenewToken(serverAddr string, token interface{}, clientID string) (newToken interface{}, err error)
	// RevokeToken revokes a previously obtained token
	RevokeToken(serverAddr string, token interface{}) error
}

var (
	driversMu sync.Mutex
	drivers   = make(map[string]Driver)
)

// Register registers a driver (called by a the driver package)
func Register(name string, driver Driver) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if driver == nil {
		panic("auth obtainer driver is nil")
	}
	drivers[name] = driver
}

// Setup configures and returns the Obtainer
func Setup(name, serverAddr string) (*Obtainer, error) {
	driversMu.Lock()
	driveri, ok := drivers[name]
	driversMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("unknown obtainer: '%s' (forgot to import driver?)", name)
	}

	return &Obtainer{
		driver:     driveri,
		serverAddr: serverAddr,
	}, nil
}

// Obtainer struct
type Obtainer struct {
	driver     Driver
	serverAddr string
}

// Wrapper functions
// These functions are public

func (o *Obtainer) ObtainToken(username, password, clientID string) (token interface{}, err error) {
	return o.driver.ObtainToken(o.serverAddr, username, password, clientID)
}

func (o *Obtainer) TokenString(token interface{}) (tokenString string, err error) {
	return o.driver.TokenString(token)
}

func (o *Obtainer) RenewToken(token interface{}, clientID string) (newToken interface{}, err error) {
	return o.driver.RenewToken(o.serverAddr, token, clientID)
}

func (o *Obtainer) RevokeToken(token interface{}) error {
	return o.driver.RevokeToken(o.serverAddr, token)
}
