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

	r, err := setupRouter(config)
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
			if cat.Auth == nil {
				go sc.RegisterServiceWithKeepalive(cat.Endpoint, cat.Discover, *service, sigCh, &wg, nil)
			} else {
				// Setup ticket obtainer
				o, err := obtainer.Setup(cat.Auth.Provider, cat.Auth.ProviderURL)
				if err != nil {
					fmt.Println(err.Error())
					continue
				}
				// Register with a ticket obtainer client
				go sc.RegisterServiceWithKeepalive(cat.Endpoint, cat.Discover, *service, sigCh, &wg,
					obtainer.NewClient(o, cat.Auth.Username, cat.Auth.Password, cat.Auth.ServiceID),
				)
			}
			regChannels = append(regChannels, sigCh)
			wg.Add(1)
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

func setupRouter(config *Config) (*mux.Router, error) {
	// Create catalog API object
	var api *catalog.WritableCatalogAPI
	if config.Storage.Type == utils.CatalogBackendMemory {
		api = catalog.NewWritableCatalogAPI(
			catalog.NewMemoryStorage(),
			config.ApiLocation,
			utils.StaticLocation,
			config.Description,
		)
	}
	if api == nil {
		return nil, fmt.Errorf("Could not create catalog API structure. Unsupported storage type: %v", config.Storage.Type)
	}

	commonHandlers := alice.New(
		context.ClearHandler,
	)

	// Append auth handler if enabled
	if config.Auth.Enabled {
		// Setup ticket validator
		v, err := validator.Setup(config.Auth.Provider, config.Auth.ProviderURL, config.Auth.ServiceID, config.Auth.Authz)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		commonHandlers = commonHandlers.Append(v.Handler)
	}

	// Configure routers
	r := mux.NewRouter().StrictSlash(true)
	r.Methods("GET").Path(config.ApiLocation).Handler(commonHandlers.ThenFunc(api.List)).Name("list")
	r.Methods("POST").Path(config.ApiLocation + "/").Handler(commonHandlers.ThenFunc(api.Add)).Name("add")
	r.Methods("GET").Path(config.ApiLocation + "/{type}/{path}/{op}/{value:.*}").Handler(commonHandlers.ThenFunc(api.Filter)).Name("filter")

	url := config.ApiLocation + "/{dgwid}/{regid}"
	r.Methods("GET").Path(url).Handler(commonHandlers.ThenFunc(api.Get)).Name("get")
	r.Methods("PUT").Path(url).Handler(commonHandlers.ThenFunc(api.Update)).Name("update")
	r.Methods("DELETE").Path(url).Handler(commonHandlers.ThenFunc(api.Delete)).Name("delete")
	r.Methods("GET").Path(url + "/{resname}").Handler(commonHandlers.ThenFunc(api.GetResource)).Name("details")

	return r, nil
}
