// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	catalog "linksmart.eu/lc/core/catalog/resource"

	_ "linksmart.eu/lc/sec/auth/cas/validator"
	"linksmart.eu/lc/sec/auth/validator"
	"strings"
)

// errorResponse used to serialize errors into JSON for RESTful responses
type errorResponse struct {
	Error string `json:"error"`
}

// RESTfulAPI contains all required configuration for running a RESTful API
// for device gateway
type RESTfulAPI struct {
	config         *Config
	restConfig     *RestProtocol
	router         *mux.Router
	dataCh         chan<- DataRequest
	commonHandlers alice.Chain
}

// Constructs a RESTfulAPI data structure
func newRESTfulAPI(conf *Config, dataCh chan<- DataRequest) (*RESTfulAPI, error) {
	restConfig, _ := conf.Protocols[ProtocolTypeREST].(RestProtocol)

	// Common handlers
	commonHandlers := alice.New(
		context.ClearHandler,
	)

	// Append auth handler if enabled
	if conf.Auth.Enabled {
		// Setup ticket validator
		v, err := validator.Setup(
			conf.Auth.Provider,
			conf.Auth.ProviderURL,
			conf.Auth.ServiceID,
			conf.Auth.BasicEnabled,
			conf.Auth.Authz)
		if err != nil {
			return nil, err
		}

		commonHandlers = commonHandlers.Append(v.Handler)
	}

	api := &RESTfulAPI{
		config:         conf,
		restConfig:     &restConfig,
		router:         mux.NewRouter().StrictSlash(true),
		dataCh:         dataCh,
		commonHandlers: commonHandlers,
	}
	return api, nil
}

// Setup all routers, handlers and start a HTTP server (blocking call)
func (api *RESTfulAPI) start(catalogController catalog.CatalogController) {
	api.mountCatalog(catalogController)
	api.mountResources()

	api.router.Methods("GET", "POST").Path("/dashboard").Handler(
		api.commonHandlers.ThenFunc(api.dashboardHandler(*confPath)))
	api.router.Methods("GET").Path(api.restConfig.Location).Handler(
		api.commonHandlers.ThenFunc(api.indexHandler()))

	err := mime.AddExtensionType(".jsonld", "application/ld+json")
	if err != nil {
		logger.Println("RESTfulAPI.start() ERROR:", err.Error())
	}

	// Configure the middleware
	n := negroni.New(
		negroni.NewRecovery(),
		negroni.NewLogger(),
		&negroni.Static{
			Dir:       http.Dir(api.config.StaticDir),
			Prefix:    StaticLocation,
			IndexFile: "index.html",
		},
	)
	// Mount router
	n.UseHandler(api.router)

	// Start the listener
	addr := fmt.Sprintf("%v:%v", api.config.Http.BindAddr, api.config.Http.BindPort)
	logger.Printf("RESTfulAPI.start() Starting server at http://%v%v", addr, api.restConfig.Location)
	n.Run(addr)
}

// Create a HTTP handler to serve and update dashboard configuration
func (api *RESTfulAPI) dashboardHandler(confPath string) http.HandlerFunc {
	dashboardConfPath := filepath.Join(filepath.Dir(confPath), "dashboard.json")

	return func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "application/json")

		if req.Method == "POST" {
			body, err := ioutil.ReadAll(req.Body)
			req.Body.Close()
			if err != nil {
				api.respondWithBadRequest(rw, err.Error())
				return
			}

			err = ioutil.WriteFile(dashboardConfPath, body, 0755)
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				errData := map[string]string{"error": err.Error()}
				b, _ := json.Marshal(errData)
				rw.Write(b)
				return
			}

			rw.WriteHeader(http.StatusCreated)
			rw.Write([]byte("{}"))

		} else if req.Method == "GET" {
			data, err := ioutil.ReadFile(dashboardConfPath)
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				errData := map[string]string{"error": err.Error()}
				b, _ := json.Marshal(errData)
				rw.Write(b)
				return
			}
			rw.WriteHeader(http.StatusOK)
			rw.Write(data)
		} else {
			rw.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func (api *RESTfulAPI) indexHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		b, _ := json.Marshal("Welcome to Device Gateway RESTful API")
		rw.Header().Set("Content-Type", "application/json")
		rw.Write(b)
	}
}

func (api *RESTfulAPI) mountResources() {
	for _, device := range api.config.Devices {
		for _, resource := range device.Resources {
			for _, protocol := range resource.Protocols {
				if protocol.Type != ProtocolTypeREST {
					continue
				}
				uri := api.restConfig.Location + "/" + device.Name + "/" + resource.Name
				logger.Println("RESTfulAPI.mountResources() Mounting resource:", uri)
				rid := device.ResourceId(resource.Name)
				for _, method := range protocol.Methods {
					switch method {
					case "GET":
						api.router.Methods("GET").Path(uri).Handler(
							api.commonHandlers.ThenFunc(api.createResourceGetHandler(rid)))
					case "PUT":
						api.router.Methods("PUT").Path(uri).Handler(
							api.commonHandlers.ThenFunc(api.createResourcePutHandler(rid)))
					}
				}
			}
		}
	}
}

func (api *RESTfulAPI) mountCatalog(catalogController catalog.CatalogController) {
	catalogAPI := catalog.NewReadableCatalogAPI(
		catalogController,
		CatalogLocation,
		StaticLocation,
		fmt.Sprintf("Local catalog at %s", api.config.Description),
	)

	// Configure routers
	api.router.Methods("GET").Path(CatalogLocation).Handler(api.commonHandlers.ThenFunc(catalogAPI.Index))

	// Devices
	// CRUD
	api.router.Methods("GET").Path(CatalogLocation + "/devices/{id}").Handler(
		api.commonHandlers.ThenFunc(catalogAPI.Get))
	// Listing, filtering
	api.router.Methods("GET").Path(CatalogLocation + "/devices").Handler(
		api.commonHandlers.ThenFunc(catalogAPI.List))
	api.router.Methods("GET").Path(CatalogLocation + "/devices/{path}/{op}/{value:.*}").Handler(
		api.commonHandlers.ThenFunc(catalogAPI.Filter))

	// Resources
	api.router.Methods("GET").Path(CatalogLocation + "/resources").Handler(
		api.commonHandlers.ThenFunc(catalogAPI.ListResources))
	api.router.Methods("GET").Path(CatalogLocation + "/resources/{id:[^/]+/?[^/]*}").Handler(
		api.commonHandlers.ThenFunc(catalogAPI.GetResource))
	api.router.Methods("GET").Path(CatalogLocation + "/resources/{path}/{op}/{value:.*}").Handler(
		api.commonHandlers.ThenFunc(catalogAPI.FilterResources))

	logger.Printf("RESTfulAPI.mountCatalog() Mounted local catalog at %v", CatalogLocation)
}

func (api *RESTfulAPI) createResourceGetHandler(resourceId string) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		logger.Printf("RESTfulAPI.createResourceGetHandler() %s %s", req.Method, req.RequestURI)

		resource, found := api.config.FindResource(resourceId)
		if !found {
			api.respondWithNotFound(rw, "Resource does not exist")
			return
		}

		// Get the first content-type
		for _, p := range resource.Protocols {
			if p.Type == ProtocolTypeREST && len(p.ContentTypes) > 0 {
				rw.Header().Set("Content-Type", p.ContentTypes[0])
			}
		}

		// Retrieve data
		dr := DataRequest{
			ResourceId: resourceId,
			Type:       DataRequestTypeRead,
			Arguments:  nil,
			Reply:      make(chan AgentResponse),
		}
		api.dataCh <- dr

		// Wait for the response
		repl := <-dr.Reply
		if repl.IsError {
			api.respondWithInternalServerError(rw, string(repl.Payload))
			return
		}
		rw.Write(repl.Payload)
	}
}

func (api *RESTfulAPI) createResourcePutHandler(resourceId string) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		logger.Printf("RESTfulAPI.createResourcePutHandler() %s %s", req.Method, req.RequestURI)

		// Resolve mediaType
		v := req.Header.Get("Content-Type")
		mediaType, _, err := mime.ParseMediaType(v)
		if err != nil {
			api.respondWithBadRequest(rw, err.Error())
			return
		}

		// Check if mediaType is supported by resource
		isSupported := false
		resource, found := api.config.FindResource(resourceId)
		if !found {
			api.respondWithNotFound(rw, "Resource does not exist")
			return
		}
		for _, p := range resource.Protocols {
			if p.Type == ProtocolTypeREST {
				for _, ct := range p.ContentTypes {
					ct := strings.ToLower(ct)
					if ct == mediaType {
						isSupported = true
					}
				}
			}
		}
		if !isSupported {
			api.respondWithUnsupportedMediaType(rw,
				fmt.Sprintf("`%s` media type is not supported by this resource", mediaType))
			return
		}

		// Extract PUT's body
		body, err := ioutil.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			api.respondWithBadRequest(rw, err.Error())
			return
		}

		// Submit data request
		dr := DataRequest{
			ResourceId: resourceId,
			Type:       DataRequestTypeWrite,
			Arguments:  body,
			Reply:      make(chan AgentResponse),
		}
		logger.Printf("RESTfulAPI.createResourcePutHandler() Submitting data request %#v", dr)
		api.dataCh <- dr

		// Wait for the response
		repl := <-dr.Reply
		if repl.IsError {
			api.respondWithInternalServerError(rw, string(repl.Payload))
			return
		}
		rw.WriteHeader(http.StatusAccepted)
	}
}

func (api *RESTfulAPI) respondWithNotFound(rw http.ResponseWriter, msg string) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusNotFound)
	err := &errorResponse{Error: msg}
	b, _ := json.Marshal(err)
	rw.Write(b)
}

func (api *RESTfulAPI) respondWithBadRequest(rw http.ResponseWriter, msg string) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusBadRequest)
	err := &errorResponse{Error: msg}
	b, _ := json.Marshal(err)
	rw.Write(b)
}

func (api *RESTfulAPI) respondWithUnsupportedMediaType(rw http.ResponseWriter, msg string) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusUnsupportedMediaType)
	err := &errorResponse{Error: msg}
	b, _ := json.Marshal(err)
	rw.Write(b)
}

func (api *RESTfulAPI) respondWithInternalServerError(rw http.ResponseWriter, msg string) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusInternalServerError)
	err := &errorResponse{Error: msg}
	b, _ := json.Marshal(err)
	rw.Write(b)
}
