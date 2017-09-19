// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package main

import (
	"flag"
	"fmt"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	_ "code.linksmart.eu/com/go-sec/auth/keycloak/obtainer"
	_ "code.linksmart.eu/com/go-sec/auth/keycloak/validator"
	"code.linksmart.eu/com/go-sec/auth/obtainer"
	"code.linksmart.eu/com/go-sec/auth/validator"
	catalog "code.linksmart.eu/rc/resource-catalog/catalog"
	sc "code.linksmart.eu/sc/service-catalog/service"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/context"
	"github.com/justinas/alice"
	"github.com/oleksandr/bonjour"
)

var (
	confPath = flag.String("conf", "conf/resource-catalog.json", "Resource catalog configuration file path")
)

func main() {
	flag.Parse()

	config, err := loadConfig(*confPath)
	if err != nil {
		logger.Fatalf("Error reading config file %v: %v", *confPath, err)
	}

	router, shutdownAPI, err := setupRouter(config)
	if err != nil {
		logger.Fatal(err.Error())
	}

	// Announce service using DNS-SD
	var bonjourS *bonjour.Server
	if config.DnssdEnabled {
		bonjourS, err = bonjour.Register(config.Description,
			catalog.DNSSDServiceType,
			"",
			config.BindPort,
			[]string{fmt.Sprintf("uri=%s", config.ApiLocation)},
			nil)
		if err != nil {
			logger.Printf("Failed to register DNS-SD service: %s", err.Error())
		} else {
			logger.Println("Registered service via DNS-SD using type", catalog.DNSSDServiceType)
		}
	}

	// Register in the configured Service Catalogs
	regChannels := make([]chan bool, 0, len(config.ServiceCatalog))
	var wg sync.WaitGroup
	if len(config.ServiceCatalog) > 0 {
		logger.Println("Will now register in the configured Service Catalogs")
		service, err := registrationFromConfig(config)
		if err != nil {
			logger.Printf("Unable to parse Service registration: %v\n", err.Error())
			return
		}

		for _, cat := range config.ServiceCatalog {
			// Set TTL
			service.Ttl = cat.Ttl
			sigCh := make(chan bool)
			wg.Add(1)
			if cat.Auth == nil {
				go sc.RegisterServiceWithKeepalive(cat.Endpoint, cat.Discover, *service, sigCh, &wg, nil)
			} else {
				// Setup ticket client
				ticket, err := obtainer.NewClient(cat.Auth.Provider, cat.Auth.ProviderURL, cat.Auth.Username, cat.Auth.Password, cat.Auth.ServiceID)
				if err != nil {
					logger.Println(err.Error())
					continue
				}
				// Register with a ticket obtainer client
				go sc.RegisterServiceWithKeepalive(cat.Endpoint, cat.Discover, *service, sigCh, &wg, ticket)
			}
			regChannels = append(regChannels, sigCh)
		}

	}

	// Setup signal catcher for the server's proper shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		for _ = range c {
			// sig is a ^C, handle it

			//TODO: put here the last will logic

			// Stop bonjour registration
			if bonjourS != nil {
				bonjourS.Shutdown()
				time.Sleep(1e9)
			}

			// Unregister in the service catalog(s)
			for _, sigCh := range regChannels {
				// Notify if the routine hasn't returned already
				select {
				case sigCh <- true:
				default:
				}
			}
			wg.Wait()

			// Shutdown catalog API
			err := shutdownAPI()
			if err != nil {
				logger.Println(err.Error())
			}

			logger.Println("Stopped")
			os.Exit(0)
		}
	}()

	err = mime.AddExtensionType(".jsonld", "application/ld+json")
	if err != nil {
		logger.Println("ERROR: ", err.Error())
	}

	// Configure the middleware
	n := negroni.New(
		negroni.NewRecovery(),
		negroni.NewLogger(),
		&negroni.Static{
			Dir:       http.Dir(config.StaticDir),
			Prefix:    catalog.StaticLocation,
			IndexFile: "index.html",
		},
	)
	// Mount router
	n.UseHandler(router)

	// Start listener
	endpoint := fmt.Sprintf("%s:%s", config.BindAddr, strconv.Itoa(config.BindPort))
	logger.Printf("Starting standalone Resource Catalog at %v%v", endpoint, config.ApiLocation)
	n.Run(endpoint)
}

func setupRouter(config *Config) (*router, func() error, error) {
	// Setup API storage
	var (
		storage catalog.CatalogStorage
		err     error
	)
	switch config.Storage.Type {
	case catalog.CatalogBackendMemory:
		storage = catalog.NewMemoryStorage()
	case catalog.CatalogBackendLevelDB:
		storage, err = catalog.NewLevelDBStorage(config.Storage.DSN, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("Failed to start LevelDB storage: %v", err.Error())
		}
	default:
		return nil, nil, fmt.Errorf("Could not create catalog API storage. Unsupported type: %v", config.Storage.Type)
	}

	controller, err := catalog.NewController(storage, config.ApiLocation)
	if err != nil {
		storage.Close()
		return nil, nil, fmt.Errorf("Failed to start the controller: %v", err.Error())
	}

	// Create catalog API object
	api := catalog.NewWritableCatalogAPI(
		controller,
		config.ApiLocation,
		catalog.StaticLocation,
		config.Description,
	)

	commonHandlers := alice.New(
		context.ClearHandler,
	)

	// Append auth handler if enabled
	if config.Auth.Enabled {
		// Setup ticket validator
		v, err := validator.Setup(
			config.Auth.Provider,
			config.Auth.ProviderURL,
			config.Auth.ServiceID,
			config.Auth.BasicEnabled,
			config.Auth.Authz)
		if err != nil {
			return nil, nil, err
		}

		commonHandlers = commonHandlers.Append(v.Handler)
	}

	// Configure http api router
	r := newRouter()
	// Index
	r.get(config.ApiLocation, commonHandlers.ThenFunc(api.Index))
	// Devices
	r.post(config.ApiLocation+"/devices", commonHandlers.ThenFunc(api.Post))
	r.get(config.ApiLocation+"/devices/{id}", commonHandlers.ThenFunc(api.Get))
	r.put(config.ApiLocation+"/devices/{id}", commonHandlers.ThenFunc(api.Put))
	r.delete(config.ApiLocation+"/devices/{id}", commonHandlers.ThenFunc(api.Delete))
	r.get(config.ApiLocation+"/devices", commonHandlers.ThenFunc(api.List))
	r.get(config.ApiLocation+"/devices/{path}/{op}/{value:.*}", commonHandlers.ThenFunc(api.Filter))
	// Resources
	r.get(config.ApiLocation+"/resources", commonHandlers.ThenFunc(api.ListResources))
	// Accept an id with zero or one slash: [^/]+/?[^/]*
	// -> [^/]+ one or more of anything but slashes /? optional slash [^/]* zero or more of anything but slashes
	r.get(config.ApiLocation+"/resources/{id:[^/]+/?[^/]*}", commonHandlers.ThenFunc(api.GetResource))
	r.get(config.ApiLocation+"/resources/{path}/{op}/{value:.*}", commonHandlers.ThenFunc(api.FilterResources))

	return r, controller.Stop, nil
}
