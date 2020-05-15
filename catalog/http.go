// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/linksmart/service-catalog/v3/utils"
)

const (
	ContextURL       = ""
	MaxPerPage       = 100
	ResponseMimeType = "application/ld+json"
	// query parameters
	QueryParamPage    = "page"
	QueryParamPerPage = "perPage"
	// Deprecated
	QueryParamFetchPath = "fetch"
	QueryParamJSONPath  = "jsonpath"
	QueryParamXPath     = "xpath"
)

type ThingDescriptionPage struct {
	Context string      `json:"@context,omitempty"`
	Items   interface{} `json:"items"`
	Page    int         `json:"page"`
	PerPage int         `json:"perPage"`
	Total   int         `json:"total"`
}

// Read-only catalog api
type HTTPAPI struct {
	controller  CatalogController
	contentType string
}

func NewHTTPAPI(controller CatalogController, version string) *HTTPAPI {
	contentType := ResponseMimeType
	if version != "" {
		contentType += ";version=" + version
	}
	return &HTTPAPI{
		controller:  controller,
		contentType: contentType,
	}
}

// add one
func (a *HTTPAPI) Post(w http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	var td ThingDescription
	if err := json.Unmarshal(body, &td); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error processing the request:", err.Error())
		return
	}

	if td[_id] != nil {
		id, ok := td[_id].(string)
		if !ok || id != "" {
			ErrorResponse(w, http.StatusBadRequest, "Registering with defined ID is not possible using a POST request.")
			return
		}
	}

	id, err := a.controller.add(td)
	if err != nil {
		switch err.(type) {
		case *ConflictError:
			ErrorResponse(w, http.StatusConflict, "Error creating the resource:", err.Error())
			return
		case *BadRequestError:
			ErrorResponse(w, http.StatusBadRequest, "Invalid registration:", err.Error())
			return
		default:
			ErrorResponse(w, http.StatusInternalServerError, "Error creating the registration:", err.Error())
			return
		}
	}

	w.Header().Set("Content-Type", a.contentType)
	w.Header().Add("Location", id)
	w.WriteHeader(http.StatusCreated)
}

// get one
func (a *HTTPAPI) Get(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)

	td, err := a.controller.get(params["id"])
	if err != nil {
		switch err.(type) {
		case *NotFoundError:
			ErrorResponse(w, http.StatusNotFound, err.Error())
			return
		default:
			ErrorResponse(w, http.StatusInternalServerError, "Error retrieving the registration:", err.Error())
			return
		}
	}

	b, err := json.Marshal(td)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", a.contentType)
	w.Write(b)
}

// Updates an existing device (Response: StatusOK)
// If the device does not exist, a new one will be created with the given id (Response: StatusCreated)
func (a *HTTPAPI) Put(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)

	body, err := ioutil.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	var td ThingDescription
	if err := json.Unmarshal(body, &td); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error processing the request:", err.Error())
		return
	}

	err = a.controller.update(params["id"], td)
	if err != nil {
		switch err.(type) {
		case *NotFoundError:
			// Create a new device with the given id
			td[_id] = params["id"]
			id, err := a.controller.add(td)
			if err != nil {
				switch err.(type) {
				case *ConflictError:
					ErrorResponse(w, http.StatusConflict, "Error creating the registration:", err.Error())
					return
				case *BadRequestError:
					ErrorResponse(w, http.StatusBadRequest, "Invalid registration:", err.Error())
					return
				default:
					ErrorResponse(w, http.StatusInternalServerError, "Error creating the registration:", err.Error())
					return
				}
			}
			w.Header().Set("Content-Type", a.contentType)
			w.Header().Set("Location", id)
			w.WriteHeader(http.StatusCreated)
			return
		case *BadRequestError:
			ErrorResponse(w, http.StatusBadRequest, "Invalid registration:", err.Error())
			return
		default:
			ErrorResponse(w, http.StatusInternalServerError, "Error updating the registration:", err.Error())
			return
		}
	}

	w.Header().Set("Content-Type", a.contentType)
	w.WriteHeader(http.StatusOK)
}

// delete one
func (a *HTTPAPI) Delete(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)

	err := a.controller.delete(params["id"])
	if err != nil {
		switch err.(type) {
		case *NotFoundError:
			ErrorResponse(w, http.StatusNotFound, err.Error())
			return
		default:
			ErrorResponse(w, http.StatusInternalServerError, "Error deleting the registration:", err.Error())
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

// Lists devices in a DeviceCollection
func (a *HTTPAPI) List(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing the query:", err.Error())
		return
	}
	page, perPage, err := utils.ParsePagingParams(
		req.Form.Get(QueryParamPage), req.Form.Get(QueryParamPerPage), MaxPerPage)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing query parameters:", err.Error())
		return
	}

	var items interface{}
	var total int
	if jsonPath := req.Form.Get(QueryParamJSONPath); jsonPath != "" {
		w.Header().Add("X-Request-Jsonpath", jsonPath)
		items, total, err = a.controller.filterJSONPath(jsonPath, page, perPage)
		if err != nil {
			ErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
	} else if xPath := req.Form.Get(QueryParamXPath); xPath != "" {
		w.Header().Add("X-Request-Xpath", xPath)
		items, total, err = a.controller.filterXPath(xPath, page, perPage)
		if err != nil {
			ErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
	} else if req.Form.Get(QueryParamFetchPath) != "" {
		ErrorResponse(w, http.StatusBadRequest, "fetch query parameter is deprecated. Use jsonpath or xpath")
		return
	} else {
		items, total, err = a.controller.list(page, perPage)
		if err != nil {
			ErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	coll := &ThingDescriptionPage{
		Context: ContextURL,
		Items:   items,
		Page:    page,
		PerPage: perPage,
		Total:   total,
	}

	b, err := json.Marshal(coll)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", a.contentType)
	w.Header().Add("X-Request-URL", req.RequestURI)
	w.Write(b)
}

// Deprecated:
// Lists filtered devices in a DeviceCollection
func (a *HTTPAPI) Filter(w http.ResponseWriter, req *http.Request) {
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
		req.Form.Get(QueryParamPage), req.Form.Get(QueryParamPerPage), MaxPerPage)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing query parameters:", err.Error())
		return
	}

	items, total, err := a.controller.filter(path, op, value, page, perPage)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	coll := &ThingDescriptionPage{
		Context: ContextURL,
		Items:   items,
		Page:    page,
		PerPage: perPage,
		Total:   total,
	}

	b, err := json.Marshal(coll)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", a.contentType)
	w.Write(b)
}