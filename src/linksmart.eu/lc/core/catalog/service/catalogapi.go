package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"linksmart.eu/lc/core/catalog"
)

const (
	CtxRootDir      = "/ctx"
	CtxPathCatalog  = "/catalog.jsonld"
)

type Collection struct {
	Context     string    `json:"@context,omitempty"`
	Id          string    `json:"id"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Services    []Service `json:"services"`
	Page        int       `json:"page"`
	PerPage     int       `json:"per_page"`
	Total       int       `json:"total"`
}

// Read-only catalog api
type ReadableCatalogAPI struct {
	controller  CatalogController
	apiLocation string
	ctxPathRoot string
	description string
	listeners   []Listener
}

// Writable catalog api
type WritableCatalogAPI struct {
	*ReadableCatalogAPI
}

func NewReadableCatalogAPI(controller CatalogController, apiLocation, staticLocation, description string, listeners ...Listener) *ReadableCatalogAPI {
	return &ReadableCatalogAPI{
		controller:  controller,
		apiLocation: apiLocation,
		ctxPathRoot: staticLocation + CtxRootDir,
		description: description,
		listeners:   listeners,
	}
}

func NewWritableCatalogAPI(controller CatalogController, apiLocation, staticLocation, description string, listeners ...Listener) *WritableCatalogAPI {
	return &WritableCatalogAPI{
		NewReadableCatalogAPI(controller, apiLocation, staticLocation, description, listeners...),
	}
}

// API Index: Lists services
func (self ReadableCatalogAPI) List(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing the query:", err.Error())
		return
	}
	page, perPage, err := catalog.ParsePagingParams(
		req.Form.Get(catalog.GetParamPage), req.Form.Get(catalog.GetParamPerPage), MaxPerPage)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing query parameters:", err.Error())
		return
	}

	services, total, err := self.controller.list(page, perPage)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	coll := &Collection{
		Context:     self.ctxPathRoot + CtxPathCatalog,
		Id:          self.apiLocation,
		Type:        ApiCollectionType,
		Description: self.description,
		Services:    services,
		Page:        page,
		PerPage:     perPage,
		Total:       total,
	}

	b, err := json.Marshal(coll)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.Write(b)
}

// Filters services
func (self ReadableCatalogAPI) Filter(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	path := params["path"]
	op := params["op"]
	value := params["value"]

	err := req.ParseForm()
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing the query:", err.Error())
		return
	}
	page, perPage, err := catalog.ParsePagingParams(
		req.Form.Get(catalog.GetParamPage), req.Form.Get(catalog.GetParamPerPage), MaxPerPage)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing query parameters:", err.Error())
		return
	}

	services, total, err := self.controller.filter(path, op, value, page, perPage)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	coll := &Collection{
		Context:     self.ctxPathRoot + CtxPathCatalog,
		Id:          self.apiLocation,
		Type:        ApiCollectionType,
		Description: self.description,
		Services:    services,
		Page:        page,
		PerPage:     perPage,
		Total:       total,
	}

	b, err := json.Marshal(coll)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.Write(b)
}

// Retrieves a service
func (a ReadableCatalogAPI) Get(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)

	s, err := a.controller.get(params["id"])
	if err != nil {
		switch err.(type) {
		case *NotFoundError:
			ErrorResponse(w, http.StatusNotFound, err.Error())
			return
		default:
			ErrorResponse(w, http.StatusInternalServerError, "Error retrieving the service:", err.Error())
			return
		}
	}

	b, err := json.Marshal(s)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.Write(b)
}

// Adds a service
func (a WritableCatalogAPI) Add(w http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	req.Body.Close()

	var s Service
	if err := json.Unmarshal(body, &s); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error processing the request:", err.Error())
		return
	}

	if s.Id != "" {
		ErrorResponse(w, http.StatusBadRequest, "Creating a service with defined ID is not possible using a POST request.")
		return
	}

	id, err := a.controller.add(s)
	if err != nil {
		switch err.(type) {
		case *ConflictError:
			ErrorResponse(w, http.StatusConflict, "Error creating the registration:", err.Error())
			return
		case *BadRequestError:
			ErrorResponse(w, http.StatusBadRequest, "Invalid service registration:", err.Error())
			return
		default:
			ErrorResponse(w, http.StatusInternalServerError, "Error creating the registration:", err.Error())
			return
		}
	}

	// notify listeners
	for _, l := range a.listeners {
		go l.added(s)
	}

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.Header().Set("Location", fmt.Sprintf("%s/%s", a.apiLocation, id))
	w.WriteHeader(http.StatusCreated)
}

// Updates an existing service (Response: StatusOK)
// or creates a new one with the given id (Response: StatusCreated)
func (a WritableCatalogAPI) Update(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)

	body, err := ioutil.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	var s Service
	if err := json.Unmarshal(body, &s); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error processing the request:", err.Error())
		return
	}

	err = a.controller.update(params["id"], s)
	if err != nil {
		switch err.(type) {
		case *NotFoundError:
			// Create a new service with the given id
			s.Id = params["id"]
			id, err := a.controller.add(s)
			if err != nil {
				ErrorResponse(w, http.StatusInternalServerError, "Error creating the registration:", err.Error())
				return
			}
			w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
			w.Header().Set("Location", fmt.Sprintf("%s/%s", a.apiLocation, id))
			w.WriteHeader(http.StatusCreated)
			return
		case *ConflictError:
			ErrorResponse(w, http.StatusConflict, "Error updating the service:", err.Error())
			return
		case *BadRequestError:
			ErrorResponse(w, http.StatusBadRequest, "Invalid service registration:", err.Error())
			return
		default:
			ErrorResponse(w, http.StatusInternalServerError, "Error updating the service:", err.Error())
			return
		}
	}

	// notify listeners
	for _, l := range a.listeners {
		go l.updated(s)
	}

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.WriteHeader(http.StatusOK)
}

// Deletes a service
func (a WritableCatalogAPI) Delete(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)

	err := a.controller.delete(params["id"])
	if err != nil {
		switch err.(type) {
		case *NotFoundError:
			ErrorResponse(w, http.StatusNotFound, err.Error())
			return
		default:
			ErrorResponse(w, http.StatusInternalServerError, "Error deleting the service:", err.Error())
			return
		}
	}

	// notify listeners
	for _, l := range a.listeners {
		go l.deleted(params["id"])
	}

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.WriteHeader(http.StatusOK)
}
