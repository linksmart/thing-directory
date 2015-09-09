package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"sync"

	cas "linksmart.eu/auth/cas/obtainer"
	auth "linksmart.eu/auth/obtainer"
	catalog "linksmart.eu/lc/core/catalog/service"
)

var (
	confPath = flag.String("conf", "", "Path to the service configuration file")
	endpoint = flag.String("endpoint", "", "Service Catalog endpoint")
	discover = flag.Bool("discover", false, "Use DNS-SD service discovery to find Service Catalog endpoint")
	// Authentication configuration
	authServer = flag.String("authServer", "", "Authentication server address")
	authUser   = flag.String("authUser", "", "Auth. server username")
	authPass   = flag.String("authPass", "", "Auth. server password")
	serviceID  = flag.String("serviceID", "", "Service ID at the auth. server")
)

func main() {
	flag.Parse()

	if *confPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	// requiresAuth if authServer is specified
	var requiresAuth bool = (*authServer != "")

	if *endpoint == "" && !*discover {
		logger.Println("ERROR: -endpoint was not provided and discover flag not set.")
		flag.Usage()
		os.Exit(1)
	}

	service, err := LoadConfigFromFile(*confPath)
	if err != nil {
		logger.Fatal("Unable to read service configuration from file: ", err)
	}

	// Launch the registration routine
	var wg sync.WaitGroup
	regCh := make(chan bool)

	if !requiresAuth {
		go catalog.RegisterServiceWithKeepalive(*endpoint, *discover, *service, regCh, &wg, nil)
	} else {
		// Setup auth client with a CAS obtainer
		catalog.RegisterServiceWithKeepalive(*endpoint, *discover, *service, regCh, &wg,
			auth.NewClient(cas.New(*authServer), *authUser, *authPass, *serviceID),
		)
	}
	wg.Add(1)

	// Ctrl+C handling
	handler := make(chan os.Signal, 1)
	signal.Notify(handler, os.Interrupt)
	for sig := range handler {
		if sig == os.Interrupt {
			logger.Println("Caught interrupt signal...")
			break
		}
	}
	// Signal shutdown to the registration routine
	select {
	// Notify if the routine hasn't returned already
	case regCh <- true:
	default:
	}
	wg.Wait()

	logger.Println("Stopped")
	os.Exit(0)
}

// Loads service registration from a config file
func LoadConfigFromFile(confPath string) (*catalog.Service, error) {
	if !strings.HasSuffix(confPath, ".json") {
		return nil, fmt.Errorf("Config should be a .json file")
	}
	f, err := ioutil.ReadFile(confPath)
	if err != nil {
		return nil, err
	}

	config := &catalog.ServiceConfig{}
	err = json.Unmarshal(f, config)
	if err != nil {
		return nil, fmt.Errorf("Error parsing config")
	}

	service, err := config.GetService()
	return service, err
}
