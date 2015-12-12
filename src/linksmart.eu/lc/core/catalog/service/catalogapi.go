package service

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
	GetParamPage    = "page"
	GetParamPerPage = "per_page"
	FTypeService    = "service"
	FTypeServices   = "services"
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
	catalogStorage CatalogStorage
	apiLocation    string
	ctxPathRoot    string
	description    string
	listeners      []Listener
}

// Writable catalog api
type WritableCatalogAPI struct {
	*ReadableCatalogAPI
}

func NewReadableCatalogAPI(storage CatalogStorage, apiLocation, staticLocation, description string, listeners ...Listener) *ReadableCatalogAPI {
	return &ReadableCatalogAPI{
		catalogStorage: storage,
		apiLocation:    apiLocation,
		ctxPathRoot:    staticLocation + CtxRootDir,
		description:    description,
		listeners:      listeners,
	}
}

func NewWritableCatalogAPI(storage CatalogStorage, apiLocation, staticLocation, description string, listeners ...Listener) *WritableCatalogAPI {
	return &WritableCatalogAPI{
		&ReadableCatalogAPI{
			catalogStorage: storage,
			apiLocation:    apiLocation,
			ctxPathRoot:    staticLocation + CtxRootDir,
			description:    description,
			listeners:      listeners,
		}}
}

func (self *Service) ldify(apiLocation string) Service {
	sc := self.copy()
	sc.Id = fmt.Sprintf("%v/%v", apiLocation, self.Id)
	return sc
}

func (self *Service) unLdify(apiLocation string) Service {
	sc := self.copy()
	sc.Id = strings.TrimPrefix(self.Id, apiLocation+"/")
	return sc
}

func (self ReadableCatalogAPI) collectionFromServices(services []Service, page, perPage, total int) *Collection {
	respServices := make([]Service, 0, len(services))
	for _, svc := range services {
		svcld := svc.ldify(self.apiLocation)
		respServices = append(respServices, svcld)
	}

	return &Collection{
		Context:     self.ctxPathRoot + CtxPathCatalog,
		Id:          self.apiLocation,
		Type:        ApiCollectionType,
		Description: self.description,
		Services:    respServices,
		Page:        page,
		PerPage:     perPage,
		Total:       total,
	}
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

func (self ReadableCatalogAPI) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

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

	services, total, err := self.catalogStorage.getMany(page, perPage)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	coll := self.collectionFromServices(services, page, perPage, total)

	b, err := json.Marshal(coll)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.Write(b)
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

	var data interface{}

	switch ftype {
	case FTypeService:
		data, err = self.catalogStorage.pathFilterOne(fpath, fop, fvalue)
		if data.(Service).Id != "" {
			svc := data.(Service)
			data = svc.ldify(self.apiLocation)
		} else {
			data = nil
		}

	case FTypeServices:
		var total int
		data, total, err = self.catalogStorage.pathFilter(fpath, fop, fvalue, page, perPage)
		data = self.collectionFromServices(data.([]Service), page, perPage, total)
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
	params := mux.Vars(req)
	id := fmt.Sprintf("%v/%v", params["hostid"], params["regid"])

	r, err := self.catalogStorage.get(id)
	if err == ErrorNotFound {
		ErrorResponse(w, http.StatusNotFound, "Service not found.")
		return
	} else if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Error requesting the service:", err.Error())
		return
	}

	b, err := json.Marshal(r.ldify(self.apiLocation))
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

	var s Service
	err = json.Unmarshal(body, &s)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error processing the request:", err.Error())
		return
	}

	err = self.catalogStorage.add(s)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Error creating the service:", err.Error())
		return
	}

	// notify listeners
	for _, l := range self.listeners {
		go l.added(s)
	}

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.Header().Set("Location", fmt.Sprintf("%s/%s", self.apiLocation, s.Id))
	w.WriteHeader(http.StatusCreated)
	return
}

func (self WritableCatalogAPI) Update(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	id := fmt.Sprintf("%v/%v", params["hostid"], params["regid"])

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	req.Body.Close()

	var s Service
	err = json.Unmarshal(body, &s)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error processing the request:", err.Error())
		return
	}

	err = self.catalogStorage.update(id, s)
	if err == ErrorNotFound {
		ErrorResponse(w, http.StatusNotFound, "Service not found.")
		return
	} else if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Error requesting the service:", err.Error())
		return
	}

	// notify listeners
	for _, l := range self.listeners {
		go l.updated(s)
	}

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.WriteHeader(http.StatusOK)
	return
}

func (self WritableCatalogAPI) Delete(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	id := fmt.Sprintf("%v/%v", params["hostid"], params["regid"])

	err := self.catalogStorage.delete(id)
	if err == ErrorNotFound {
		ErrorResponse(w, http.StatusNotFound, "Not found.")
		return
	} else if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Error deleting the device:", err.Error())
		return
	}

	// notify listeners
	for _, l := range self.listeners {
		go l.deleted(id)
	}

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.WriteHeader(http.StatusOK)
	return
}
