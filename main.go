// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/context"
	"github.com/justinas/alice"
	_ "github.com/linksmart/go-sec/auth/keycloak/obtainer"
	_ "github.com/linksmart/go-sec/auth/keycloak/validator"
	"github.com/linksmart/go-sec/auth/validator"
	"github.com/linksmart/thing-directory/catalog"
	"github.com/linksmart/thing-directory/wot"
	uuid "github.com/satori/go.uuid"
)

const LINKSMART = `
╦   ╦ ╔╗╔ ╦╔═  ╔═╗ ╔╦╗ ╔═╗ ╦═╗ ╔╦╗
║   ║ ║║║ ╠╩╗  ╚═╗ ║║║ ╠═╣ ╠╦╝  ║
╩═╝ ╩ ╝╚╝ ╩ ╩  ╚═╝ ╩ ╩ ╩ ╩ ╩╚═  ╩
`

var (
	confPath    = flag.String("conf", "conf/thing-directory.json", "Configuration file path")
	schemaPath  = flag.String("schema", "conf/wot_td_schema.json", "WoT Thing Description schema file path")
	version     = flag.Bool("version", false, "Print the API version")
	Version     string // set with build flags
	BuildNumber string // set with build flags
)

func main() {
	flag.Parse()
	if *version {
		fmt.Println(Version)
		return
	}

	fmt.Print(LINKSMART)
	log.Printf("Starting Thing Directory")
	defer log.Println("Stopped.")

	if Version != "" {
		log.Printf("Version: %s", Version)
	}
	if BuildNumber != "" {
		log.Printf("Build Number: %s", BuildNumber)
	}

	config, err := loadConfig(*confPath)
	if err != nil {
		panic("Error reading config file:" + err.Error())
	}
	log.Printf("Loaded config file: " + *confPath)
	if config.ServiceID == "" {
		config.ServiceID = uuid.NewV4().String()
		log.Printf("Service ID not set. Generated new UUID: %s", config.ServiceID)
	}
	log.Print("Loaded schema file: " + *schemaPath)

	err = wot.LoadSchema(*schemaPath)
	if err != nil {
		panic("error loading WoT Thing Description schema: " + err.Error())
	}

	// Setup API storage
	var storage catalog.Storage
	switch config.Storage.Type {
	case catalog.BackendLevelDB:
		storage, err = catalog.NewLevelDBStorage(config.Storage.DSN, nil)
		if err != nil {
			panic("Failed to start LevelDB storage:" + err.Error())
		}
		defer storage.Close()
	default:
		panic("Could not create catalog API storage. Unsupported type:" + config.Storage.Type)
	}

	controller, err := catalog.NewController(storage)
	if err != nil {
		panic("Failed to start the controller:" + err.Error())
	}
	defer controller.Stop()

	// Create catalog API object
	api := catalog.NewHTTPAPI(controller, Version)

	nRouter, err := setupHTTPRouter(config, api)
	if err != nil {
		panic(err)
	}
	// Start listener
	addr := fmt.Sprintf("%s:%d", config.BindAddr, config.BindPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	log.Printf("HTTP server listening on %v", addr)
	go func() { log.Fatalln(http.Serve(listener, nRouter)) }()

	// Publish service using DNS-SD
	if config.DNSSD.Publish {
		shutdown, err := registerDNSSDService(config)
		if err != nil {
			log.Printf("Failed to register DNS-SD service: %s", err)
		}
		defer shutdown()
	}

	// Register in the LinkSmart Service Catalog
	if config.ServiceCatalog.Enabled {
		unregisterService, err := registerInServiceCatalog(config)
		if err != nil {
			panic("Error registering service:" + err.Error())
		}
		// Unregister from the Service Catalog
		defer unregisterService()
	}

	log.Println("Ready!")

	// Ctrl+C / Kill handling
	handler := make(chan os.Signal, 1)
	signal.Notify(handler, os.Interrupt, os.Kill)
	<-handler
	log.Println("Shutting down...")
}

func setupHTTPRouter(config *Config, api *catalog.HTTPAPI) (*negroni.Negroni, error) {

	commonHandlers := alice.New(
		context.ClearHandler,
	)

	// Append auth handler if enabled
	if config.Auth.Enabled {
		// Setup ticket validator
		v, err := validator.Setup(
			config.Auth.Provider,
			config.Auth.ProviderURL,
			config.Auth.ClientID,
			config.Auth.BasicEnabled,
			&config.Auth.Authz)
		if err != nil {
			return nil, err
		}

		commonHandlers = commonHandlers.Append(v.Handler)
	}

	// Configure http api router
	r := newRouter()
	r.get("/", commonHandlers.ThenFunc(indexHandler))

	r.get("/td", commonHandlers.ThenFunc(api.GetMany))
	r.get("/td/filter/{path}/{op}/{value:.*}", commonHandlers.ThenFunc(api.Filter)) // deprecated

	r.post("/td", commonHandlers.ThenFunc(api.Post))
	r.get("/td/{id:.+}", commonHandlers.ThenFunc(api.Get))
	r.put("/td/{id:.+}", commonHandlers.ThenFunc(api.Put))
	r.delete("/td/{id:.+}", commonHandlers.ThenFunc(api.Delete))

	r.get("/validation", commonHandlers.ThenFunc(api.GetValidation))

	logger := negroni.NewLogger()
	logFlags := log.LstdFlags
	if evalEnv(EnvDisableLogTime) {
		logFlags = 0
	}
	logger.ALogger = log.New(os.Stdout, "", logFlags)
	logger.SetFormat("{{.Method}} {{.Request.URL}} {{.Status}} {{.Duration}}")

	// Configure the middleware
	n := negroni.New(
		negroni.NewRecovery(),
		logger,
	)
	// Mount router
	n.UseHandler(r)

	return n, nil
}
