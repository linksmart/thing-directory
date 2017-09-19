// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
)

const (
	TypeDevices   = "devices"
	TypeResources = "resources"
	CtxPath       = "/ctx/rc.jsonld"
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

type JSONLDSimpleDevice struct {
	Context string `json:"@context"`
	*SimpleDevice
}

type JSONLDResource struct {
	Context string `json:"@context"`
	*Resource
}

// Read-only catalog api
type ReadableCatalogAPI struct {
	controller  CatalogController
	apiLocation string
	ctxPath     string
	description string
}

// Writable catalog api
type WritableCatalogAPI struct {
	*ReadableCatalogAPI
}

func NewReadableCatalogAPI(controller CatalogController, apiLocation, staticLocation, description string) *ReadableCatalogAPI {
	return &ReadableCatalogAPI{
		controller:  controller,
		apiLocation: apiLocation,
		ctxPath:     staticLocation + CtxPath,
		description: description,
	}
}

func NewWritableCatalogAPI(controller CatalogController, apiLocation, staticLocation, description string) *WritableCatalogAPI {
	return &WritableCatalogAPI{
		NewReadableCatalogAPI(controller, apiLocation, staticLocation, description),
	}
}

// Index of API
func (a *ReadableCatalogAPI) Index(w http.ResponseWriter, req *http.Request) {
	total, err := a.controller.total()
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Error counting devices:", err.Error())
		return
	}
	totalResources, err := a.controller.totalResources()
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Error counting resources:", err.Error())
		return
	}

	index := map[string]interface{}{
		"description":     a.description,
		"api_version":     ApiVersion,
		"total_devices":   total,
		"total_resources": totalResources,
	}

	b, err := json.Marshal(&index)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.Write(b)
}

// DEVICES

// Adds a Device
func (a *WritableCatalogAPI) Post(w http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	var d Device
	if err := json.Unmarshal(body, &d); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error processing the request:", err.Error())
		return
	}

	if d.Id != "" {
		ErrorResponse(w, http.StatusBadRequest, "Creating a device with defined ID is not possible using a POST request.")
		return
	}

	id, err := a.controller.add(d)
	if err != nil {
		switch err.(type) {
		case *ConflictError:
			ErrorResponse(w, http.StatusConflict, "Error creating the registration:", err.Error())
			return
		case *BadRequestError:
			ErrorResponse(w, http.StatusBadRequest, "Invalid device registration:", err.Error())
			return
		default:
			ErrorResponse(w, http.StatusInternalServerError, "Error creating the registration:", err.Error())
			return
		}
	}

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.Header().Set("Location", fmt.Sprintf("%s/%s/%s", a.apiLocation, TypeDevices, id))
	w.WriteHeader(http.StatusCreated)
}

// Gets a single Device
func (a *ReadableCatalogAPI) Get(w http.ResponseWriter, req *http.Request) {
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

	ldd := JSONLDSimpleDevice{
		Context:      a.ctxPath,
		SimpleDevice: d,
	}

	b, err := json.Marshal(ldd)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.Write(b)
}

// Updates an existing device (Response: StatusOK)
// If the device does not exist, a new one will be created with the given id (Response: StatusCreated)
func (a *WritableCatalogAPI) Put(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)

	body, err := ioutil.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	var d Device
	if err := json.Unmarshal(body, &d); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error processing the request:", err.Error())
		return
	}

	err = a.controller.update(params["id"], d)
	if err != nil {
		switch err.(type) {
		case *NotFoundError:
			// Create a new device with the given id
			d.Id = params["id"]
			id, err := a.controller.add(d)
			if err != nil {
				ErrorResponse(w, http.StatusInternalServerError, "Error creating the registration:", err.Error())
				return
			}
			w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
			w.Header().Set("Location", fmt.Sprintf("%s/%s/%s", a.apiLocation, TypeDevices, id))
			w.WriteHeader(http.StatusCreated)
			return
		case *ConflictError:
			ErrorResponse(w, http.StatusConflict, "Error updating the device:", err.Error())
			return
		case *BadRequestError:
			ErrorResponse(w, http.StatusBadRequest, "Invalid device registration:", err.Error())
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
func (a *WritableCatalogAPI) Delete(w http.ResponseWriter, req *http.Request) {
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
func (a *ReadableCatalogAPI) List(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing the query:", err.Error())
		return
	}
	page, perPage, err := ParsePagingParams(
		req.Form.Get(GetParamPage), req.Form.Get(GetParamPerPage), MaxPerPage)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing query parameters:", err.Error())
		return
	}

	simpleDevices, total, err := a.controller.list(page, perPage)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	coll := &DeviceCollection{
		Context: a.ctxPath,
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
func (a *ReadableCatalogAPI) Filter(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	path := params["path"]
	op := params["op"]
	value := params["value"]

	err := req.ParseForm()
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing the query:", err.Error())
		return
	}
	page, perPage, err := ParsePagingParams(
		req.Form.Get(GetParamPage), req.Form.Get(GetParamPerPage), MaxPerPage)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing query parameters:", err.Error())
		return
	}

	simpleDevices, total, err := a.controller.filter(path, op, value, page, perPage)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	coll := &DeviceCollection{
		Context: a.ctxPath,
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
func (a *ReadableCatalogAPI) GetResource(w http.ResponseWriter, req *http.Request) {
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

	ldr := JSONLDResource{
		Context:  a.ctxPath,
		Resource: r,
	}

	b, err := json.Marshal(ldr)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.Write(b)
}

// Lists resources in a ResourceCollection
func (a *ReadableCatalogAPI) ListResources(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing the query:", err.Error())
		return
	}
	page, perPage, err := ParsePagingParams(
		req.Form.Get(GetParamPage), req.Form.Get(GetParamPerPage), MaxPerPage)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing query parameters:", err.Error())
		return
	}

	resources, total, err := a.controller.listResources(page, perPage)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	coll := &ResourceCollection{
		Context:   a.ctxPath,
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
func (a *ReadableCatalogAPI) FilterResources(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	path := params["path"]
	op := params["op"]
	value := params["value"]

	err := req.ParseForm()
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing the query:", err.Error())
		return
	}
	page, perPage, err := ParsePagingParams(
		req.Form.Get(GetParamPage), req.Form.Get(GetParamPerPage), MaxPerPage)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing query parameters:", err.Error())
		return
	}

	resources, total, err := a.controller.filterResources(path, op, value, page, perPage)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Error processing the request:", err.Error())
		return
	}

	coll := &ResourceCollection{
		Context:   a.ctxPath,
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
