// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package main

import (
	"flag"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"github.com/oleksandr/bonjour"
	utils "linksmart.eu/lc/core/catalog"
	catalog "linksmart.eu/lc/core/catalog/service"

	_ "linksmart.eu/lc/sec/auth/cas/validator"
	_ "linksmart.eu/lc/sec/auth/keycloak/validator"
	"linksmart.eu/lc/sec/auth/validator"
)

var (
	confPath = flag.String("conf", "conf/service-catalog.json", "Service catalog configuration file path")
)

func main() {
	flag.Parse()

	config, err := loadConfig(*confPath)
	if err != nil {
		logger.Fatalf("Error reading config file %v: %v", *confPath, err)
	}

	r, shutdownAPI, err := setupRouter(config)
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
			Prefix:    utils.StaticLocation,
			IndexFile: "index.html",
		},
	)
	// Mount router
	n.UseHandler(r)

	// Start listener
	endpoint := fmt.Sprintf("%s:%s", config.BindAddr, strconv.Itoa(config.BindPort))
	logger.Printf("Starting standalone Service Catalog at %v%v", endpoint, config.ApiLocation)
	n.Run(endpoint)
}

func setupRouter(config *Config) (*mux.Router, func() error, error) {
	var listeners []catalog.Listener
	// GC publisher if configured
	if config.GC.TunnelingService != "" {
		endpoint, _ := url.Parse(config.GC.TunnelingService)
		listeners = append(listeners, catalog.NewGCPublisher(*endpoint))
	}

	// Setup API storage
	var (
		storage catalog.CatalogStorage
		err     error
	)
	switch config.Storage.Type {
	case utils.CatalogBackendMemory:
		storage = catalog.NewMemoryStorage()
	case utils.CatalogBackendLevelDB:
		storage, err = catalog.NewLevelDBStorage(config.Storage.DSN, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("Failed to start LevelDB storage: %v", err.Error())
		}
	default:
		return nil, nil, fmt.Errorf("Could not create catalog API storage. Unsupported type: %v", config.Storage.Type)
	}

	controller, err := catalog.NewController(storage, config.ApiLocation, listeners...)
	if err != nil {
		storage.Close()
		return nil, nil, fmt.Errorf("Failed to start the controller: %v", err.Error())
	}

	// Create catalog API object
	api := catalog.NewCatalogAPI(
		controller,
		config.ApiLocation,
		utils.StaticLocation,
		config.Description,
	)

	commonHandlers := alice.New(
		context.ClearHandler,
	)

	// Append auth handler if enabled
	if config.Auth.Enabled {
		// Setup ticket validator
		v, err := validator.Setup(config.Auth.Provider, config.Auth.ProviderURL, config.Auth.ServiceID, config.Auth.Authz)
		if err != nil {
			return nil, nil, err
		}

		commonHandlers = commonHandlers.Append(v.Handler)
	}

	// Configure routers
	r := mux.NewRouter().StrictSlash(true)
	// Handlers
	r.Methods("POST").Path(config.ApiLocation + "/").Handler(commonHandlers.ThenFunc(api.Post))
	// Accept an id with zero or one slash: [^/]+/?[^/]*
	// -> [^/]+ one or more of anything but slashes /? optional slash [^/]* zero or more of anything but slashes
	r.Methods("GET").Path(config.ApiLocation + "/{id:[^/]+/?[^/]*}").Handler(commonHandlers.ThenFunc(api.Get))
	r.Methods("PUT").Path(config.ApiLocation + "/{id:[^/]+/?[^/]*}").Handler(commonHandlers.ThenFunc(api.Put))
	r.Methods("DELETE").Path(config.ApiLocation + "/{id:[^/]+/?[^/]*}").Handler(commonHandlers.ThenFunc(api.Delete))
	// List, Filter
	r.Methods("GET").Path(config.ApiLocation).Handler(commonHandlers.ThenFunc(api.List))
	r.Methods("GET").Path(config.ApiLocation + "/{path}/{op}/{value:.*}").Handler(commonHandlers.ThenFunc(api.Filter))

	return r, controller.Stop, nil
}
