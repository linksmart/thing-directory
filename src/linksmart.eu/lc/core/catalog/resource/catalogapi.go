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
	//FTypeDevice     = "device"
	//FTypeDevices    = "devices"
	//FTypeResource   = "resource"
	//FTypeResources  = "resources"
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
	ID        string     `json:"id"`
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

func (self ReadableCatalogAPI) List(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing the query:", err.Error())
		return
	}
	page, _ := strconv.Atoi(req.Form.Get(GetParamPage))
	perPage, _ := strconv.Atoi(req.Form.Get(GetParamPerPage))
	page, perPage = catalog.ValidatePagingParams(page, perPage, MaxPerPage)

	simpleDevices, total, err := self.controller.listDevices(page, perPage)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	coll := &DeviceCollection{
		Context: self.ctxPathRoot + CtxPathCatalog,
		Id:      self.apiLocation,
		Type:    ApiCollectionType,
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
	return
}

//func (self ReadableCatalogAPI) Filter(w http.ResponseWriter, req *http.Request) {
//	params := mux.Vars(req)
//	ftype := params["type"]
//	fpath := params["path"]
//	fop := params["op"]
//	fvalue := params["value"]
//
//	err := req.ParseForm()
//	if err != nil {
//		ErrorResponse(w, http.StatusBadRequest, "Error parsing the query:", err.Error())
//		return
//	}
//	page, _ := strconv.Atoi(req.Form.Get(GetParamPage))
//	perPage, _ := strconv.Atoi(req.Form.Get(GetParamPerPage))
//	page, perPage = catalog.ValidatePagingParams(page, perPage, MaxPerPage)
//
//	var (
//		data  interface{}
//		total int
//	)
//
//	switch ftype {
//	case FTypeDevice:
//		data, err = self.catalogStorage.pathFilterDevice(fpath, fop, fvalue)
//		if data.(Device).Id != "" {
//			data = self.paginatedDeviceFromDevice(data.(Device), page, perPage)
//		} else {
//			data = nil
//		}
//
//	case FTypeDevices:
//		data, total, err = self.catalogStorage.pathFilterDevices(fpath, fop, fvalue, page, perPage)
//		data = self.collectionFromDevices(data.([]Device), page, perPage, total)
//		if data.(*Collection).Total == 0 {
//			data = nil
//		}
//
//	case FTypeResource:
//		data, err = self.catalogStorage.pathFilterResource(fpath, fop, fvalue)
//		if data.(Resource).Id != "" {
//			res := data.(Resource)
//			data = res.ldify(self.apiLocation)
//		} else {
//			data = nil
//		}
//
//	case FTypeResources:
//		data, total, err = self.catalogStorage.pathFilterResources(fpath, fop, fvalue, page, perPage)
//		data = self.collectionFromDevices(data.([]Device), page, perPage, total)
//		if data.(*Collection).Total == 0 {
//			data = nil
//		}
//	}
//
//	if err != nil {
//		ErrorResponse(w, http.StatusInternalServerError, "Error processing the request:", err.Error())
//		return
//	}
//
//	if data == nil {
//		ErrorResponse(w, http.StatusNotFound, "No matched entries found.")
//		return
//	}
//
//	b, err := json.Marshal(data)
//	if err != nil {
//		ErrorResponse(w, http.StatusInternalServerError, err.Error())
//		return
//	}
//	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
//	w.Write(b)
//}

func (self ReadableCatalogAPI) Get(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	//id := fmt.Sprintf("%v/%v", params["dgwid"], params["id"])

	d, err := self.controller.getDevice(params["id"])
	switch err.(type) {
	case *NotFoundError:
		ErrorResponse(w, http.StatusNotFound, err.Error())
		return
	default:
		ErrorResponse(w, http.StatusInternalServerError, "Error retrieving the device:", err.Error())
		return
	}

	b, err := json.Marshal(d)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.Write(b)
	return
}

func (self ReadableCatalogAPI) GetResource(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)

	r, err := self.controller.getResource(params["id"])
	switch err.(type) {
	case *NotFoundError:
		ErrorResponse(w, http.StatusNotFound, err.Error())
		return
	default:
		ErrorResponse(w, http.StatusInternalServerError, "Error retrieving the device:", err.Error())
		return
	}

	b, err := json.Marshal(r)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.Write(b)
	return
}

func (self WritableCatalogAPI) Add(w http.ResponseWriter, req *http.Request) {
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

	err = self.controller.addDevice(&d)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Error creating the registration:", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.Header().Set("Location", fmt.Sprintf("%s/%s", self.apiLocation, d.Id))
	w.WriteHeader(http.StatusCreated)
	return
}

func (self WritableCatalogAPI) Update(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	id := fmt.Sprintf("%v/%v", params["dgwid"], params["regid"])

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

	err = self.controller.updateDevice(id, &d)
	if err == ErrorNotFound {
		ErrorResponse(w, http.StatusNotFound, "Not found.")
		return
	} else if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Error updating the device:", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.WriteHeader(http.StatusOK)
	return
}

func (self WritableCatalogAPI) Delete(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	id := fmt.Sprintf("%v/%v", params["dgwid"], params["regid"])

	err := self.controller.deleteDevice(id)
	if err == ErrorNotFound {
		ErrorResponse(w, http.StatusNotFound, "Not found.")
		return
	} else if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Error deleting the device:", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.WriteHeader(http.StatusOK)
	return
}

func (self ReadableCatalogAPI) ListResources(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing the query:", err.Error())
		return
	}
	page, _ := strconv.Atoi(req.Form.Get(GetParamPage))
	perPage, _ := strconv.Atoi(req.Form.Get(GetParamPerPage))
	page, perPage = catalog.ValidatePagingParams(page, perPage, MaxPerPage)

	resources, total, err := self.controller.listResources(page, perPage)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	coll := &ResourceCollection{
		Context:   self.ctxPathRoot + CtxPathCatalog,
		Type:      ApiCollectionType,
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
	return
}
