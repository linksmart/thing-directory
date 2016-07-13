// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/oleksandr/bonjour"
	"strconv"
)

const (
	discoveryTimeoutSec = 30
	minKeepaliveSec     = 5
	GetParamPage        = "page"
	GetParamPerPage     = "per_page"
)

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
			logger.Println("[DiscoverCatalogEndpoint] System interrupt signal received. Aborting the discovery")
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

// Returns a 'slice' of the given slice based on the requested 'page'
func GetPageOfSlice(slice []string, page, perPage, maxPerPage int) []string {
	err := ValidatePagingParams(page, perPage, maxPerPage)
	if err != nil {
		logger.Printf("GetPageOfSlice() Bad input: %s\n", err)
		return []string{}
	}

	keys := []string{}
	if page == 1 {
		// first page
		if perPage > len(slice) {
			keys = slice
		} else {
			keys = slice[:perPage]
		}
	} else if page == int(len(slice)/perPage)+1 {
		// last page
		keys = slice[perPage*(page-1):]

	} else if page <= len(slice)/perPage && page*perPage <= len(slice) {
		// slice
		r := page * perPage
		l := r - perPage
		keys = slice[l:r]
	}
	return keys
}

// Returns offset and limit representing a subset of the given slice total size
//	 based on the requested 'page'
func GetPagingAttr(total, page, perPage, maxPerPage int) (int, int) {
	err := ValidatePagingParams(page, perPage, maxPerPage)
	if err != nil {
		logger.Printf("GetPagingAttr() Bad input: %s\n", err)
		return 0, 0
	}

	if page == 1 {
		// first page
		if perPage > total {
			return 0, total
		} else {
			return 0, perPage
		}
	} else if page == int(total/perPage)+1 {
		// last page
		return perPage * (page - 1), total - perPage*(page-1)
	} else if page <= total/perPage && page*perPage <= total {
		// another page
		r := page * perPage
		l := r - perPage
		return l, r - l
	}
	return 0, 0
}

// Validates paging parameters
func ValidatePagingParams(page, perPage, maxPerPage int) error {
	if page < 1 {
		return fmt.Errorf("%s parameter must be positive", GetParamPage)
	}
	if perPage < 1 {
		return fmt.Errorf("%s parameter must be positive", GetParamPerPage)
	}
	if perPage > maxPerPage {
		return fmt.Errorf("%s must less than or equal to %d", GetParamPerPage, maxPerPage)
	}
	return nil
}

// Parses string paging parameters to integers
func ParsePagingParams(page, perPage string, maxPerPage int) (int, int, error) {
	var parsedPage, parsedPerPage int
	var err error

	if page == "" {
		parsedPage = 1
	} else {
		parsedPage, err = strconv.Atoi(page)
		if err != nil {
			return 0, 0, fmt.Errorf("Invalid value for parameter %s: %s", GetParamPage, page)
		}
	}

	if perPage == "" {
		parsedPerPage = maxPerPage
	} else {
		parsedPerPage, err = strconv.Atoi(perPage)
		if err != nil {
			return 0, 0, fmt.Errorf("Invalid value for parameter %s: %s", GetParamPerPage, perPage)
		}
	}

	return parsedPage, parsedPerPage, ValidatePagingParams(parsedPage, parsedPerPage, maxPerPage)
}
