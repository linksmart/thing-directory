// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package client

import (
	"fmt"
	"sync"
	"time"

	"github.com/linksmart/go-sec/auth/obtainer"
	"github.com/linksmart/service-catalog/v3/catalog"
)

// RegisterService registers service into a catalog
func RegisterService(endpoint string, service catalog.Service, ticket *obtainer.Client) (*catalog.Service, error) {
	// Configure client
	client, err := NewHTTPClient(endpoint, ticket)
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP client: %s", err)
	}

	updatedService, err := client.Put(&service)
	if err != nil {
		return nil, fmt.Errorf("error PUTing registration: %v", err)
	}

	return updatedService, nil
}

// UnregisterService removes service from a catalog
func UnregisterService(endpoint string, service catalog.Service, ticket *obtainer.Client) error {
	// Configure client
	client, err := NewHTTPClient(endpoint, ticket)
	if err != nil {
		return fmt.Errorf("error creating HTTP client: %s", err)
	}

	err = client.Delete(service.ID)
	if err != nil {
		return fmt.Errorf("error PUTing registration: %v", err)
	}

	return nil
}

// RegisterServiceAndKeepalive registers a service into a catalog and continuously updates it in order to avoid expiry
// endpoint: catalog endpoint.
// service: service registration
// ticket: set to nil for no auth
// It returns a function for stopping the keepalive and another function for updating the service in keepalive routine
func RegisterServiceAndKeepalive(endpoint string, service catalog.Service, ticket *obtainer.Client) (func() error, func(catalog.Service), error) {
	mutex := sync.RWMutex{}

	client, err := NewHTTPClient(endpoint, ticket)
	if err != nil {
		return nil, nil, err
	}

	ticker := time.NewTicker(time.Duration(service.TTL) * time.Second)
	go func() {
		for ; true; <-ticker.C {
			mutex.RLock()
			_, err := client.Put(&service)
			if err != nil {
				logger.Printf("Error updating service registration for %s: %s", service.ID, err)
				continue
			}
			mutex.RUnlock()
			logger.Printf("Updated service registration for %s", service.ID)
		}
	}()

	stop := func() error {
		ticker.Stop()
		mutex.RLock()
		client.Delete(service.ID)
		if err != nil {
			logger.Printf("Error removing service registration for %s: %s", service.ID, err)
		}
		mutex.RUnlock()
		return nil
	}

	update := func(updatedService catalog.Service) {
		logger.Printf("Service registration for %s will be updated in the next heartbeat.", service.ID)
		mutex.Lock()
		service = updatedService
		mutex.Unlock()
	}

	return stop, update, nil
}
