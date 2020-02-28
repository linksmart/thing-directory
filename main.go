// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package main

import (
	"flag"
	"fmt"
	"log"
	"mime"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/context"
	"github.com/justinas/alice"
	_ "github.com/linksmart/go-sec/auth/keycloak/obtainer"
	_ "github.com/linksmart/go-sec/auth/keycloak/validator"
	"github.com/linksmart/go-sec/auth/validator"
	"github.com/linksmart/resource-catalog/catalog"
	"github.com/oleksandr/bonjour"
	uuid "github.com/satori/go.uuid"
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
	if config.ServiceID == "" {
		config.ServiceID = uuid.NewV4().String()
		log.Printf("Service ID not set. Generated new UUID: %s", config.ServiceID)
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
			[]string{"uri=/"},
			nil)
		if err != nil {
			logger.Printf("Failed to register DNS-SD service: %s", err.Error())
		} else {
			logger.Println("Registered service via DNS-SD using type", catalog.DNSSDServiceType)
		}
	}

	// Register in the LinkSmart Service Catalog
	if config.ServiceCatalog != nil {
		unregisterService, err := registerInServiceCatalog(config)
		if err != nil {
			log.Fatalf("Error registering service: %s", err)
		}
		// Unregister from the Service Catalog
		defer unregisterService()
	}

	err = mime.AddExtensionType(".jsonld", "application/ld+json")
	if err != nil {
		logger.Println("ERROR: ", err.Error())
	}

	// Configure the middleware
	n := negroni.New(
		negroni.NewRecovery(),
		negroni.NewLogger(),
	)
	// Mount router
	n.UseHandler(router)

	// Start listener
	endpoint := fmt.Sprintf("%s:%s", config.BindAddr, strconv.Itoa(config.BindPort))
	logger.Printf("Starting standalone Resource Catalog at %v", endpoint)
	go n.Run(endpoint)

	// Ctrl+C / Kill handling
	handler := make(chan os.Signal, 1)
	signal.Notify(handler, os.Interrupt, os.Kill)
	<-handler
	logger.Println("Shutting down...")

	// Stop bonjour registration
	if bonjourS != nil {
		bonjourS.Shutdown()
		time.Sleep(1e9)
	}

	// Shutdown catalog API
	err = shutdownAPI()
	if err != nil {
		logger.Println(err)
	}

	logger.Println("Stopped")
}

func setupRouter(config *Config) (*router, func() error, error) {
	// Setup API storage
	var (
		storage catalog.Storage
		err     error
	)
	switch config.Storage.Type {
	case catalog.CatalogBackendLevelDB:
		storage, err = catalog.NewLevelDBStorage(config.Storage.DSN, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("Failed to start LevelDB storage: %v", err.Error())
		}
	default:
		return nil, nil, fmt.Errorf("Could not create catalog API storage. Unsupported type: %v", config.Storage.Type)
	}

	controller, err := catalog.NewController(storage)
	if err != nil {
		storage.Close()
		return nil, nil, fmt.Errorf("Failed to start the controller: %v", err.Error())
	}

	// Create catalog API object
	api := catalog.NewHTTPAPI(
		controller,
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
	r.post("/", commonHandlers.ThenFunc(api.Post))
	r.get("/{id:.+}", commonHandlers.ThenFunc(api.Get))
	r.put("/{id:.+}", commonHandlers.ThenFunc(api.Put))
	r.delete("/{id:.+}", commonHandlers.ThenFunc(api.Delete))
	r.get("/", commonHandlers.ThenFunc(api.List))
	r.get("/filter/{path}/{op}/{value:.*}", commonHandlers.ThenFunc(api.Filter))

	return r, controller.Stop, nil
}
