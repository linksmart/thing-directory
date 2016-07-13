// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package obtainer

import (
	"fmt"
	"sync"
)

// Interface methods to login, obtain Service Ticket, and logout
type Driver interface {
	// Given serverAddr, valid username and password,
	// 	Login must return a Ticket Granting Ticket (TGT).
	Login(serverAddr, username, password string) (string, error)
	// Given serverAddr, valid TGT and serviceID,
	//	RequestTicket must return a Service Ticket.
	RequestTicket(serverAddr, TGT, serviceID string) (string, error)
	// Given serverAddr, and a valid TGT,
	// 	Logout must expire the TGT.
	Logout(serverAddr, TGT string) error
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
		panic("Auth: Obtainer driver is nil")
	}
	drivers[name] = driver
}

// Setup the driver
func Setup(name, serverAddr string) (*Obtainer, error) {
	driversMu.Lock()
	driveri, ok := drivers[name]
	driversMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("Auth: unknown obtainer %s (forgot to import driver?)", name)
	}
	obtainer := &Obtainer{
		driver:     driveri,
		serverAddr: serverAddr,
	}
	return obtainer, nil
}

// Obtainer struct
type Obtainer struct {
	driver     Driver
	serverAddr string
}

// Wrapper functions
// These functions are public

func (o *Obtainer) Login(username, password string) (string, error) {
	return o.driver.Login(o.serverAddr, username, password)
}

func (o *Obtainer) RequestTicket(TGT, serviceID string) (string, error) {
	return o.driver.RequestTicket(o.serverAddr, TGT, serviceID)
}

func (o *Obtainer) Logout(TGT string) error {
	return o.driver.Logout(o.serverAddr, TGT)
}
