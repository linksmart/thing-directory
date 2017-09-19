// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package discovery

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/oleksandr/bonjour"
)

const discoveryTimeoutSec = 30

// Discovers a catalog endpoint given the serviceType
func DiscoverCatalogEndpoint(serviceType string) (endpoint string, err error) {
	sysSig := make(chan os.Signal, 1)
	signal.Notify(sysSig,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	for {
		// create resolver
		resolver, err := bonjour.NewResolver(nil)
		if err != nil {
			logger.Println("Failed to initialize DNS-SD resolver:", err.Error())
			break
		}
		// init the channel for results
		results := make(chan *bonjour.ServiceEntry)

		// send query and listen for answers
		logger.Println("Browsing...")
		err = resolver.Browse(serviceType, "", results)
		if err != nil {
			logger.Println("Unable to browse DNS-SD services: ", err)
			break
		}

		// if not found - block with timeout
		var foundService *bonjour.ServiceEntry
		select {
		case foundService = <-results:
			logger.Printf("[DiscoverCatalogEndpoint] Discovered service: %v\n", foundService.ServiceInstanceName())
		case <-time.After(time.Duration(discoveryTimeoutSec) * time.Second):
			logger.Println("[DiscoverCatalogEndpoint] Timeout looking for a service")
		case <-sysSig:
			logger.Println("[DiscoverCatalogEndpoint] System interrupt signal received. Aborting the go-discovery")
			return endpoint, fmt.Errorf("Aborted by system interrupt")
		}

		// check if something found
		if foundService == nil {
			logger.Printf("[DiscoverCatalogEndpoint] Could not discover a service %v withing the timeout. Starting from scratch...", serviceType)
			// stop resolver
			resolver.Exit <- true
			// start the new iteration
			continue
		}

		// stop the resolver and close the channel
		resolver.Exit <- true
		close(results)

		uri := ""
		for _, s := range foundService.Text {
			if strings.HasPrefix(s, "uri=") {
				tmp := strings.Split(s, "=")
				if len(tmp) == 2 {
					uri = tmp[1]
					break
				}
			}
		}
		endpoint = fmt.Sprintf("http://%s:%v%s", foundService.HostName, foundService.Port, uri)
		break
	}
	return endpoint, err
}
