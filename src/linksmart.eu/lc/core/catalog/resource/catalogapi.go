package resource

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"linksmart.eu/lc/core/catalog"
)

const (
	FTypeDevice     = "device"
	FTypeDevices    = "devices"
	FTypeResource   = "resource"
	FTypeResources  = "resources"
	GetParamPage    = "page"
	GetParamPerPage = "per_page"
	CtxRootDir      = "/ctx"
	CtxPathCatalog  = "/catalog.jsonld"
)

type Collection struct {
	Context     string                 `json:"@context,omitempty"`
	Id          string                 `json:"id"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Devices     map[string]EmptyDevice `json:"devices"`
	Resources   []Resource             `json:"resources"`
	Page        int                    `json:"page"`
	PerPage     int                    `json:"per_page"`
	Total       int                    `json:"total"`
}

// Device object with empty resources
type EmptyDevice struct {
	*Device
	Resources []Resource `json:"resources,omitempty"`
}

// Device object with paginated resources
type PaginatedDevice struct {
	*Device
	Resources []Resource `json:"resources"`
	Page      int        `json:"page"`
	PerPage   int        `json:"per_page"`
	Total     int        `json:"total"`
}

// Read-only catalog api
type ReadableCatalogAPI struct {
	catalogStorage CatalogStorage
	apiLocation    string
	ctxPathRoot    string
	description    string
}

// Writable catalog api
type WritableCatalogAPI struct {
	*ReadableCatalogAPI
}

func NewReadableCatalogAPI(storage CatalogStorage, apiLocation, staticLocation, description string) *ReadableCatalogAPI {
	return &ReadableCatalogAPI{
		catalogStorage: storage,
		apiLocation:    apiLocation,
		ctxPathRoot:    staticLocation + CtxRootDir,
		description:    description,
	}
}

func NewWritableCatalogAPI(storage CatalogStorage, apiLocation, staticLocation, description string) *WritableCatalogAPI {
	return &WritableCatalogAPI{
		&ReadableCatalogAPI{
			catalogStorage: storage,
			apiLocation:    apiLocation,
			ctxPathRoot:    staticLocation + CtxRootDir,
			description:    description,
		}}
}

func (self *Device) ldify(apiLocation string) Device {
	rc := self.copy()
	for i, res := range rc.Resources {
		rc.Resources[i] = res.ldify(apiLocation)
	}
	rc.Id = fmt.Sprintf("%v/%v", apiLocation, self.Id)
	return rc
}

func (self *Resource) ldify(apiLocation string) Resource {
	resc := self.copy()
	resc.Id = fmt.Sprintf("%v/%v", apiLocation, self.Id)
	resc.Device = fmt.Sprintf("%v/%v", apiLocation, self.Device)
	return resc
}

func (self *Device) unLdify(apiLocation string) Device {
	rc := self.copy()
	for i, res := range rc.Resources {
		rc.Resources[i] = res.unLdify(apiLocation)
	}
	rc.Id = strings.TrimPrefix(self.Id, apiLocation+"/")
	return rc
}

func (self *Resource) unLdify(apiLocation string) Resource {
	resc := self.copy()
	resc.Id = strings.TrimPrefix(self.Id, apiLocation+"/")
	resc.Device = strings.TrimPrefix(self.Device, apiLocation+"/")
	return resc
}

func (self ReadableCatalogAPI) collectionFromDevices(devices []Device, page, perPage, total int) *Collection {
	respDevices := make(map[string]EmptyDevice)
	var respResources []Resource

	for _, d := range devices {
		dld := d.ldify(self.apiLocation)
		for _, res := range dld.Resources {
			respResources = append(respResources, res)
		}

		respDevices[d.Id] = EmptyDevice{
			&dld,
			nil,
		}
	}

	return &Collection{
		Context:     self.ctxPathRoot + CtxPathCatalog,
		Id:          self.apiLocation,
		Type:        ApiCollectionType,
		Description: self.description,
		Devices:     respDevices,
		Resources:   respResources,
		Page:        page,
		PerPage:     perPage,
		Total:       total,
	}
}

func (self ReadableCatalogAPI) paginatedDeviceFromDevice(d Device, page, perPage int) *PaginatedDevice {
	dev := d.ldify(self.apiLocation)
	pd := &PaginatedDevice{
		&dev,
		make([]Resource, 0, len(d.Resources)),
		page,
		perPage,
		len(d.Resources),
	}

	resourceIds := make([]string, 0, len(d.Resources))
	for _, r := range d.Resources {
		resourceIds = append(resourceIds, r.Id)
	}

	pageResourceIds := catalog.GetPageOfSlice(resourceIds, page, perPage, MaxPerPage)
	for _, id := range pageResourceIds {
		for _, r := range d.Resources {
			if r.Id == id {
				pd.Resources = append(pd.Resources, r.ldify(self.apiLocation))
			}
		}
	}

	return pd
}

// Error describes an API error (serializable in JSON)
type Error struct {
	// Code is the (http) code of the error
	Code int `json:"code"`
	// Message is the (human-readable) error message
	Message string `json:"message"`
}

// ErrorResponse writes error to HTTP ResponseWriter
func ErrorResponse(w http.ResponseWriter, code int, msgs ...string) {
	msg := strings.Join(msgs, " ")
	e := &Error{
		code,
		msg,
	}
	logger.Println("ERROR:", msg)
	b, _ := json.Marshal(e)
	w.Header().Set("Content-Type", "application/json;version="+ApiVersion)
	w.WriteHeader(code)
	w.Write(b)
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

	devices, total, err := self.catalogStorage.getMany(page, perPage)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	coll := self.collectionFromDevices(devices, page, perPage, total)

	b, err := json.Marshal(coll)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.Write(b)

	return
}

func (self ReadableCatalogAPI) Filter(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	ftype := params["type"]
	fpath := params["path"]
	fop := params["op"]
	fvalue := params["value"]

	err := req.ParseForm()
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing the query:", err.Error())
		return
	}
	page, _ := strconv.Atoi(req.Form.Get(GetParamPage))
	perPage, _ := strconv.Atoi(req.Form.Get(GetParamPerPage))
	page, perPage = catalog.ValidatePagingParams(page, perPage, MaxPerPage)

	var (
		data  interface{}
		total int
	)

	switch ftype {
	case FTypeDevice:
		data, err = self.catalogStorage.pathFilterDevice(fpath, fop, fvalue)
		if data.(Device).Id != "" {
			data = self.paginatedDeviceFromDevice(data.(Device), page, perPage)
		} else {
			data = nil
		}

	case FTypeDevices:
		data, total, err = self.catalogStorage.pathFilterDevices(fpath, fop, fvalue, page, perPage)
		data = self.collectionFromDevices(data.([]Device), page, perPage, total)
		if data.(*Collection).Total == 0 {
			data = nil
		}

	case FTypeResource:
		data, err = self.catalogStorage.pathFilterResource(fpath, fop, fvalue)
		if data.(Resource).Id != "" {
			res := data.(Resource)
			data = res.ldify(self.apiLocation)
		} else {
			data = nil
		}

	case FTypeResources:
		data, total, err = self.catalogStorage.pathFilterResources(fpath, fop, fvalue, page, perPage)
		data = self.collectionFromDevices(data.([]Device), page, perPage, total)
		if data.(*Collection).Total == 0 {
			data = nil
		}
	}

	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Error processing the request:", err.Error())
		return
	}

	if data == nil {
		ErrorResponse(w, http.StatusNotFound, "No matched entries found.")
		return
	}

	b, err := json.Marshal(data)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.Write(b)
}

func (self ReadableCatalogAPI) Get(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing the query:", err.Error())
		return
	}
	page, _ := strconv.Atoi(req.Form.Get(GetParamPage))
	perPage, _ := strconv.Atoi(req.Form.Get(GetParamPerPage))
	page, perPage = catalog.ValidatePagingParams(page, perPage, MaxPerPage)

	params := mux.Vars(req)
	id := fmt.Sprintf("%v/%v", params["dgwid"], params["regid"])

	d, err := self.catalogStorage.get(id)
	if err == ErrorNotFound {
		ErrorResponse(w, http.StatusNotFound, "Device not found.")
		return
	} else if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Error requesting the device:", err.Error())
		return
	}

	pd := self.paginatedDeviceFromDevice(d, page, perPage)
	b, err := json.Marshal(pd)
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
	devid := fmt.Sprintf("%v/%v", params["dgwid"], params["regid"])
	resid := fmt.Sprintf("%v/%v", devid, params["resname"])

	// check if device devid exists
	_, err := self.catalogStorage.get(devid)
	if err == ErrorNotFound {
		ErrorResponse(w, http.StatusNotFound, "Registration not found.")
		return
	} else if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Error requesting the device:", err.Error())
		return
	}

	// check if it has a resource resid
	res, err := self.catalogStorage.getResourceById(resid)
	if err == ErrorNotFound {
		ErrorResponse(w, http.StatusNotFound, "Registration not found.")
		return
	} else if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Error requesting the resource:", err.Error())
		return
	}

	b, err := json.Marshal(res.ldify(self.apiLocation))
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
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	req.Body.Close()

	var d Device
	err = json.Unmarshal(body, &d)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error processing the request:", err.Error())
		return
	}

	err = self.catalogStorage.add(d)
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
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	req.Body.Close()

	var d Device
	err = json.Unmarshal(body, &d)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error processing the request:", err.Error())
		return
	}

	err = self.catalogStorage.update(id, d)
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

	err := self.catalogStorage.delete(id)
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
