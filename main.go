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
	"syscall"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/context"
	"github.com/justinas/alice"
	_ "github.com/linksmart/go-sec/auth/keycloak/obtainer"
	_ "github.com/linksmart/go-sec/auth/keycloak/validator"
	"github.com/linksmart/go-sec/auth/validator"
	"github.com/linksmart/thing-directory/catalog"
	"github.com/linksmart/thing-directory/wot"
	"github.com/rs/cors"
	uuid "github.com/satori/go.uuid"
)

const LINKSMART = `
╦   ╦ ╔╗╔ ╦╔═  ╔═╗ ╔╦╗ ╔═╗ ╦═╗ ╔╦╗
║   ║ ║║║ ╠╩╗  ╚═╗ ║║║ ╠═╣ ╠╦╝  ║
╩═╝ ╩ ╝╚╝ ╩ ╩  ╚═╝ ╩ ╩ ╩ ╩ ╩╚═  ╩
`

const (
	SwaggerUISchemeLess = "linksmart.github.io/swagger-ui/dist"
	Spec                = "https://raw.githubusercontent.com/linksmart/thing-directory/{version}/apidoc/openapi-spec.yml"
	SourceCodeRepo      = "https://github.com/linksmart/thing-directory"
)

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

	nRouter, err := setupHTTPRouter(&config.HTTP, api)
	if err != nil {
		panic(err)
	}
	// Start listener
	addr := fmt.Sprintf("%s:%d", config.HTTP.BindAddr, config.HTTP.BindPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	log.Printf("HTTP server listening on %v", addr)
	go func() { log.Fatalln(http.Serve(listener, nRouter)) }()

	// Publish service using DNS-SD
	if config.DNSSD.Publish.Enabled {
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
		defer func() {
			err := unregisterService()
			if err != nil {
				log.Printf("Error unregistering service from catalog: %s", err)
			}
		}()
	}

	log.Println("Ready!")

	// Ctrl+C / Kill handling
	handler := make(chan os.Signal, 1)
	signal.Notify(handler, syscall.SIGINT, syscall.SIGTERM)
	<-handler
	log.Println("Shutting down...")
}

func setupHTTPRouter(config *HTTPConfig, api *catalog.HTTPAPI) (*negroni.Negroni, error) {

	corsHandler := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
		ExposedHeaders:   []string{"*"},
	})
	commonHandlers := alice.New(
		context.ClearHandler,
		corsHandler.Handler,
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
	r.options("/{path:.*}", commonHandlers.ThenFunc(optionsHandler))
	// OpenAPI Proxy for Swagger "try it out" feature
	r.get("/openapi-spec-proxy", commonHandlers.ThenFunc(apiSpecProxy))
	r.get("/openapi-spec-proxy/{basepath:.+}", commonHandlers.ThenFunc(apiSpecProxy))
	// TD listing, filtering
	r.get("/td", commonHandlers.ThenFunc(api.GetMany))
	r.get("/search/jsonpath", commonHandlers.ThenFunc(api.SearchJSONPath))
	r.get("/search/xpath", commonHandlers.ThenFunc(api.SearchXPath))
	// TD crud
	r.post("/td", commonHandlers.ThenFunc(api.Post))
	r.get("/td/{id:.+}", commonHandlers.ThenFunc(api.Get))
	r.put("/td/{id:.+}", commonHandlers.ThenFunc(api.Put))
	r.patch("/td/{id:.+}", commonHandlers.ThenFunc(api.Patch))
	r.delete("/td/{id:.+}", commonHandlers.ThenFunc(api.Delete))
	// TD validation
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
