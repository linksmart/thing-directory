package validator

import (
	"fmt"
	"net/http"
	"sync"

	"linksmart.eu/lc/sec/authz"
)

// Interface methods to validate Service Ticket
type Driver interface {
	// Given a valid ticket for the specified serviceID,
	//	ValidateTicket must return true with a set of user attributes.
	Validate(serverAddr, serviceID, ticket string) (bool, map[string]string, error)
	// Handler must perform ticket validation and authorization
	//	in form of a HTTP middleware function
	// The usage of *authz.Conf is optional and must be discarded if set to nil
	Handler(serverAddr, serviceID string, authz *authz.Conf, next http.Handler) http.Handler
}

var (
	driversMu sync.Mutex
	drivers   = make(map[string]Driver)
)

// Register a driver
func Register(name string, driver Driver) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if driver == nil {
		panic("Auth: Validator driver is nil")
	}
	drivers[name] = driver
}

// Setup the driver
// 	authz is optional and can be set to nil
func Setup(name, serverAddr, serviceID string, authz *authz.Conf) (*Validator, error) {
	driversMu.Lock()
	driveri, ok := drivers[name]
	driversMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("Auth: unknown validator %s (forgot to import driver?)", name)
	}
	validator := &Validator{
		driver:     driveri,
		serverAddr: serverAddr,
		serviceID:  serviceID,
		authz:      authz,
	}

	return validator, nil
}

// Obtainer struct
type Validator struct {
	driver     Driver
	serverAddr string
	serviceID  string
	// Authorization is optional
	authz *authz.Conf
}

// Wrapper functions
// These functions are public

func (v *Validator) Validate(ticket string) (bool, map[string]string, error) {
	return v.driver.Validate(v.serverAddr, v.serviceID, ticket)
}

func (v *Validator) Handler(next http.Handler) http.Handler {
	return v.driver.Handler(v.serverAddr, v.serviceID, v.authz, next)
}
