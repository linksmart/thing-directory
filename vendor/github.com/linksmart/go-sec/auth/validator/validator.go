// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

// Package validator provides an interface for OpenID Connect token validation
package validator

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/linksmart/go-sec/authz"
)

// Interface methods to validate Service Ticket
type Driver interface {
	// Validate must validate a ticket, given the server address and service ID
	//	When ticket is valid, it must return true together with the UserProfile
	//	When ticket is invalid, it must return false and provide the reason in the UserProfile.Status
	Validate(serverAddr, serviceID, ticket string) (bool, *UserProfile, error)
}

var (
	driversMu sync.Mutex
	drivers   = make(map[string]Driver)
	logger    *log.Logger
)

// Register registers a driver (called by a the driver package)
func Register(name string, driver Driver) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if driver == nil {
		panic("auth validator driver is nil")
	}
	drivers[name] = driver
}

// Setup configures and returns the Validator
// 	parameter authz is optional and can be set to nil
func Setup(name, serverAddr, clientID string, basicEnabled bool, authz *authz.Conf) (*Validator, error) {
	driversMu.Lock()
	driveri, ok := drivers[name]
	driversMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("unknown validator %s (forgot to import driver?)", name)
	}

	// Initialize the logger
	logger = log.New(os.Stdout, fmt.Sprintf("[%s] ", name), 0)
	v, err := strconv.Atoi(os.Getenv("DEBUG"))
	if err == nil && v == 1 {
		logger.SetFlags(log.Ltime | log.Lshortfile)
	}

	return &Validator{
		driver:       driveri,
		driverName:   name,
		serverAddr:   serverAddr,
		clientID:     clientID,
		basicEnabled: basicEnabled,
		authz:        authz,
	}, nil
}

// Validator struct
type Validator struct {
	driver       Driver
	driverName   string
	serverAddr   string
	clientID     string
	basicEnabled bool
	// Authorization is optional
	authz *authz.Conf
}

// Validate validates a ticket
//	When ticket is valid, it returns true together with the UserProfile
//	When ticket is invalid, it returns false and provide the reason in the UserProfile.Status
func (v *Validator) Validate(ticket string) (bool, *UserProfile, error) {
	return v.driver.Validate(v.serverAddr, v.clientID, ticket)
}

// UserProfile is the profile of user that is returned by the Validator
type UserProfile struct {
	Username string
	Groups   []string
	// Status is the message given when token is not validated
	Status string
}
