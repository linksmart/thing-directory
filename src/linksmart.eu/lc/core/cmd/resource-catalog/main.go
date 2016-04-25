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

	"github.com/codegangsta/negroni"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"github.com/oleksandr/bonjour"
	utils "linksmart.eu/lc/core/catalog"
	catalog "linksmart.eu/lc/core/catalog/resource"
	sc "linksmart.eu/lc/core/catalog/service"

	_ "linksmart.eu/lc/sec/auth/cas/obtainer"
	"linksmart.eu/lc/sec/auth/obtainer"

	_ "linksmart.eu/lc/sec/auth/cas/validator"
	"linksmart.eu/lc/sec/auth/validator"
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
			Prefix:    utils.StaticLocation,
			IndexFile: "index.html",
		},
	)
	// Mount router
	n.UseHandler(r)

	// Start listener
	endpoint := fmt.Sprintf("%s:%s", config.BindAddr, strconv.Itoa(config.BindPort))
	logger.Printf("Starting standalone Resource Catalog at %v%v", endpoint, config.ApiLocation)
	n.Run(endpoint)
}

func setupRouter(config *Config) (*mux.Router, func() error, error) {
	// Setup API storage
	var (
		storage catalog.CatalogStorage
		//err     error
	)
	switch config.Storage.Type {
	case utils.CatalogBackendMemory:
		storage = catalog.NewMemoryStorage()
	case utils.CatalogBackendLevelDB:
		//storage, err = catalog.NewLevelDBStorage(config.Storage.DSN, nil)
		//if err != nil {
		//	return nil, nil, fmt.Errorf("Failed to start LevelDB storage: %v", err.Error())
		//}
	default:
		return nil, nil, fmt.Errorf("Could not create catalog API storage. Unsupported type: %v", config.Storage.Type)
	}

	// Create catalog API object
	api := catalog.NewWritableCatalogAPI(
		storage,
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

	// Devices
	// CRUD
	r.Methods("POST").Path(config.ApiLocation + "/devices/").Handler(commonHandlers.ThenFunc(api.Add))
	r.Methods("GET").Path(config.ApiLocation + "/devices/{id}").Handler(commonHandlers.ThenFunc(api.Get))
	r.Methods("PUT").Path(config.ApiLocation + "/devices/{id}").Handler(commonHandlers.ThenFunc(api.Update))
	r.Methods("DELETE").Path(config.ApiLocation + "/devices/{id}").Handler(commonHandlers.ThenFunc(api.Delete))
	// Listing, filtering
	r.Methods("GET").Path(config.ApiLocation + "/devices").Handler(commonHandlers.ThenFunc(api.List))
	r.Methods("GET").Path(config.ApiLocation + "/devices/{path}/{op}/{value:.*}").Handler(commonHandlers.ThenFunc(api.Filter))

	// Resources
	r.Methods("GET").Path(config.ApiLocation + "/resources").Handler(commonHandlers.ThenFunc(api.ListResources))
	r.Methods("GET").Path(config.ApiLocation + "/resources/{id}").Handler(commonHandlers.ThenFunc(api.GetResource))
	r.Methods("GET").Path(config.ApiLocation + "/resources/{path}/{op}/{value:.*}").Handler(commonHandlers.ThenFunc(api.FilterResources))

	return r, storage.Close, nil
}
