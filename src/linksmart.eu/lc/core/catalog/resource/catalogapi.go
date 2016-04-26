package resource

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"linksmart.eu/lc/core/catalog"
)

const (
	FTypeDevices    = "devices"
	FTypeResources  = "resources"
	GetParamPage    = "page"
	GetParamPerPage = "per_page"
	CtxRootDir      = "/ctx"
	CtxPathCatalog  = "/catalog.jsonld"
)

type DeviceCollection struct {
	Context string         `json:"@context,omitempty"`
	Id      string         `json:"id"`
	Type    string         `json:"type"`
	Devices []SimpleDevice `json:"devices"`
	Page    int            `json:"page"`
	PerPage int            `json:"per_page"`
	Total   int            `json:"total"`
}

type ResourceCollection struct {
	Context   string     `json:"@context,omitempty"`
	Id        string     `json:"id"`
	Type      string     `json:"type"`
	Resources []Resource `json:"resources"`
	Page      int        `json:"page"`
	PerPage   int        `json:"per_page"`
	Total     int        `json:"total"`
}

// Read-only catalog api
type ReadableCatalogAPI struct {
	controller  CatalogController
	apiLocation string
	ctxPathRoot string
	description string
}

// Writable catalog api
type WritableCatalogAPI struct {
	*ReadableCatalogAPI
}

func NewReadableCatalogAPI(storage CatalogStorage, apiLocation, staticLocation, description string) *ReadableCatalogAPI {
	controller, err := NewController(storage, apiLocation)
	if err != nil {
		log.Panicln("TODO:", err.Error())
	}

	return &ReadableCatalogAPI{
		controller:  controller,
		apiLocation: apiLocation,
		ctxPathRoot: staticLocation + CtxRootDir,
		description: description,
	}
}

func NewWritableCatalogAPI(storage CatalogStorage, apiLocation, staticLocation, description string) *WritableCatalogAPI {
	return &WritableCatalogAPI{
		NewReadableCatalogAPI(storage, apiLocation, staticLocation, description),
	}
}

// DEVICES

// Adds a Device
func (a WritableCatalogAPI) Add(w http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	var d Device
	err = json.Unmarshal(body, &d)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error processing the request:", err.Error())
		return
	}

	err = a.controller.add(&d)
	if err != nil {
		switch err.(type) {
		case *ConflictError:
			ErrorResponse(w, http.StatusConflict, "Error creating the registration:", err.Error())
			return
		default:
			ErrorResponse(w, http.StatusInternalServerError, "Error creating the registration:", err.Error())
			return
		}
	}

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.Header().Set("Location", fmt.Sprintf("%s/%s", a.apiLocation, d.Id))
	w.WriteHeader(http.StatusCreated)
}

// Gets a single Device
func (a ReadableCatalogAPI) Get(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)

	d, err := a.controller.get(params["id"])
	if err != nil {
		switch err.(type) {
		case *NotFoundError:
			ErrorResponse(w, http.StatusNotFound, err.Error())
			return
		default:
			ErrorResponse(w, http.StatusInternalServerError, "Error retrieving the device:", err.Error())
			return
		}
	}

	b, err := json.Marshal(d)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.Write(b)
}

// Updates an existing device (Response: StatusOK)
// If the device does not exist, a new one will be created with the given id (Response: StatusCreated)
func (a WritableCatalogAPI) Update(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)

	body, err := ioutil.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	var d Device
	err = json.Unmarshal(body, &d)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error processing the request:", err.Error())
		return
	}

	err = a.controller.update(params["id"], &d)
	if err != nil {
		switch err.(type) {
		case *NotFoundError:
			// Create a new device with the given id
			d.Id = params["id"]
			err = a.controller.add(&d)
			if err != nil {
				ErrorResponse(w, http.StatusInternalServerError, "Error creating the registration:", err.Error())
				return
			}
			w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
			w.Header().Set("Location", fmt.Sprintf("%s/%s", a.apiLocation, d.Id))
			w.WriteHeader(http.StatusCreated)
			return
		case *ConflictError:
			ErrorResponse(w, http.StatusConflict, "Error updating the device:", err.Error())
			return
		default:
			ErrorResponse(w, http.StatusInternalServerError, "Error updating the device:", err.Error())
			return
		}
	}

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.WriteHeader(http.StatusOK)
}

// Deletes a device
func (a WritableCatalogAPI) Delete(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)

	err := a.controller.delete(params["id"])
	if err != nil {
		switch err.(type) {
		case *NotFoundError:
			ErrorResponse(w, http.StatusNotFound, err.Error())
			return
		default:
			ErrorResponse(w, http.StatusInternalServerError, "Error deleting the device:", err.Error())
			return
		}
	}

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.WriteHeader(http.StatusOK)
}

// Lists devices in a DeviceCollection
func (a ReadableCatalogAPI) List(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing the query:", err.Error())
		return
	}
	page, _ := strconv.Atoi(req.Form.Get(GetParamPage))
	perPage, _ := strconv.Atoi(req.Form.Get(GetParamPerPage))
	page, perPage = catalog.ValidatePagingParams(page, perPage, MaxPerPage)

	simpleDevices, total, err := a.controller.list(page, perPage)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	coll := &DeviceCollection{
		Context: a.ctxPathRoot + CtxPathCatalog,
		Id:      a.apiLocation,
		Type:    ApiDeviceCollectionType,
		Devices: simpleDevices,
		Page:    page,
		PerPage: perPage,
		Total:   total,
	}

	b, err := json.Marshal(coll)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.Write(b)
}

// Lists filtered devices in a DeviceCollection
func (a ReadableCatalogAPI) Filter(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	path := params["path"]
	op := params["op"]
	value := params["value"]

	err := req.ParseForm()
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing the query:", err.Error())
		return
	}
	page, _ := strconv.Atoi(req.Form.Get(GetParamPage))
	perPage, _ := strconv.Atoi(req.Form.Get(GetParamPerPage))
	page, perPage = catalog.ValidatePagingParams(page, perPage, MaxPerPage)

	simpleDevices, total, err := a.controller.filter(path, op, value, page, perPage)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	coll := &DeviceCollection{
		Context: a.ctxPathRoot + CtxPathCatalog,
		Id:      a.apiLocation,
		Type:    ApiDeviceCollectionType,
		Devices: simpleDevices,
		Page:    page,
		PerPage: perPage,
		Total:   total,
	}

	b, err := json.Marshal(coll)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.Write(b)
}

// RESOURCES

// Gets a single Resource
func (a ReadableCatalogAPI) GetResource(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)

	r, err := a.controller.getResource(params["id"])
	if err != nil {
		switch err.(type) {
		case *NotFoundError:
			ErrorResponse(w, http.StatusNotFound, err.Error())
			return
		default:
			ErrorResponse(w, http.StatusInternalServerError, "Error retrieving the device:", err.Error())
			return
		}
	}

	b, err := json.Marshal(r)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.Write(b)
}

// Lists resources in a ResourceCollection
func (a ReadableCatalogAPI) ListResources(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing the query:", err.Error())
		return
	}
	page, _ := strconv.Atoi(req.Form.Get(GetParamPage))
	perPage, _ := strconv.Atoi(req.Form.Get(GetParamPerPage))
	page, perPage = catalog.ValidatePagingParams(page, perPage, MaxPerPage)

	resources, total, err := a.controller.listResources(page, perPage)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	coll := &ResourceCollection{
		Context:   a.ctxPathRoot + CtxPathCatalog,
		Id:        a.apiLocation,
		Type:      ApiResourceCollectionType,
		Resources: resources,
		Page:      page,
		PerPage:   perPage,
		Total:     total,
	}

	b, err := json.Marshal(coll)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.Write(b)
}

// Lists filtered resources in a ResourceCollection
func (a ReadableCatalogAPI) FilterResources(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	path := params["path"]
	op := params["op"]
	value := params["value"]

	err := req.ParseForm()
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing the query:", err.Error())
		return
	}
	page, _ := strconv.Atoi(req.Form.Get(GetParamPage))
	perPage, _ := strconv.Atoi(req.Form.Get(GetParamPerPage))
	page, perPage = catalog.ValidatePagingParams(page, perPage, MaxPerPage)

	resources, total, err := a.controller.filterResources(path, op, value, page, perPage)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Error processing the request:", err.Error())
		return
	}

	coll := &ResourceCollection{
		Context:   a.ctxPathRoot + CtxPathCatalog,
		Id:        a.apiLocation,
		Type:      ApiResourceCollectionType,
		Resources: resources,
		Page:      page,
		PerPage:   perPage,
		Total:     total,
	}

	b, err := json.Marshal(coll)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.Write(b)
}
