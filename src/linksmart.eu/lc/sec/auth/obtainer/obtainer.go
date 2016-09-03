// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package obtainer

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
)

// Interface methods to login, obtain Service Ticket, and logout
type Driver interface {
	// Login must return a Ticket Granting Ticket (TGT), given serverAddr, valid username, password, and serviceID
	Login(serverAddr, username, password, serviceID string) (string, error)
	// RequestTicket must return a Service Ticket, given serverAddr, valid TGT and serviceID
	RequestTicket(serverAddr, TGT, serviceID string) (string, error)
	// Logout must expire the TGT, given serverAddr, and a valid TGT
	Logout(serverAddr, TGT string) error
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
		panic("Auth: Obtainer driver is nil")
	}
	drivers[name] = driver
}

// Setup configures and returns the Obtainer
func Setup(name, serverAddr string) (*Obtainer, error) {
	driversMu.Lock()
	driveri, ok := drivers[name]
	driversMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("Auth: unknown obtainer %s (forgot to import driver?)", name)
	}

	// Initialize the logger
	logger = log.New(os.Stdout, fmt.Sprintf("[%s] ", name), 0)
	v, err := strconv.Atoi(os.Getenv("DEBUG"))
	if err == nil && v == 1 {
		logger.SetFlags(log.Ltime | log.Lshortfile)
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

func (o *Obtainer) Login(username, password, serviceID string) (string, error) {
	return o.driver.Login(o.serverAddr, username, password, serviceID)
}

func (o *Obtainer) RequestTicket(TGT, serviceID string) (string, error) {
	return o.driver.RequestTicket(o.serverAddr, TGT, serviceID)
}

func (o *Obtainer) Logout(TGT string) error {
	return o.driver.Logout(o.serverAddr, TGT)
}
