// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"

	"code.linksmart.eu/sc/service-catalog/utils"
	"github.com/gorilla/mux"
)

const (
	CtxPath = "/ctx/sc.jsonld"
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

type JSONLDService struct {
	Context string `json:"@context"`
	*Service
}

// Read-only catalog api
type CatalogAPI struct {
	controller  CatalogController
	apiLocation string
	ctxPath     string
	description string
}

func NewCatalogAPI(controller CatalogController, apiLocation, staticLocation, description string) *CatalogAPI {
	return &CatalogAPI{
		controller:  controller,
		apiLocation: apiLocation,
		ctxPath:     staticLocation + CtxPath,
		description: description,
	}
}

// API Index: Lists services
func (a *CatalogAPI) List(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing the query:", err.Error())
		return
	}
	page, perPage, err := utils.ParsePagingParams(
		req.Form.Get(utils.GetParamPage), req.Form.Get(utils.GetParamPerPage), MaxPerPage)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing query parameters:", err.Error())
		return
	}

	services, total, err := a.controller.list(page, perPage)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	coll := &Collection{
		Context:     a.ctxPath,
		Id:          a.apiLocation,
		Type:        ApiCollectionType,
		Description: a.description,
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
func (a *CatalogAPI) Filter(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	path := params["path"]
	op := params["op"]
	value := params["value"]

	err := req.ParseForm()
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing the query:", err.Error())
		return
	}
	page, perPage, err := utils.ParsePagingParams(
		req.Form.Get(utils.GetParamPage), req.Form.Get(utils.GetParamPerPage), MaxPerPage)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing query parameters:", err.Error())
		return
	}

	services, total, err := a.controller.filter(path, op, value, page, perPage)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	coll := &Collection{
		Context:     a.ctxPath,
		Id:          a.apiLocation,
		Type:        ApiCollectionType,
		Description: a.description,
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
func (a *CatalogAPI) Get(w http.ResponseWriter, req *http.Request) {
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

	lds := JSONLDService{
		Context: a.ctxPath,
		Service: s,
	}

	b, err := json.Marshal(lds)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.Write(b)
}

// Adds a service
func (a *CatalogAPI) Post(w http.ResponseWriter, req *http.Request) {
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

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.Header().Set("Location", fmt.Sprintf("%s/%s", a.apiLocation, id))
	w.WriteHeader(http.StatusCreated)
}

// Updates an existing service (Response: StatusOK)
// or creates a new one with the given id (Response: StatusCreated)
func (a *CatalogAPI) Put(w http.ResponseWriter, req *http.Request) {
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

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.WriteHeader(http.StatusOK)
}

// Deletes a service
func (a *CatalogAPI) Delete(w http.ResponseWriter, req *http.Request) {
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

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.WriteHeader(http.StatusOK)
}

// Serves static and all /static/ctx files as ld+json
func NewStaticHandler(staticDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.RequestURI, StaticLocation+"/ctx/") {
			w.Header().Set("Content-Type", "application/ld+json")
		}
		urlParts := strings.Split(req.URL.Path, "/")
		http.ServeFile(w, req, filepath.Join(staticDir, strings.Join(urlParts[2:], "/")))
	}
}
