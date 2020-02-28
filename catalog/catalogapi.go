// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
)

type ThingsCollection struct {
	Context string             `json:"@context"`
	Things  []ThingDescription `json:"things"`
	ID      string             `json:"id"`
	Page    int                `json:"page"`
	PerPage int                `json:"per_page"`
	Total   int                `json:"total"`
}

// Read-only catalog api
type HTTPAPI struct {
	controller CatalogController
	id         string
}

func NewHTTPAPI(controller CatalogController, serviceID string) *HTTPAPI {
	return &HTTPAPI{
		controller: controller,
		id:         serviceID,
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

	if td.ID != "" {
		ErrorResponse(w, http.StatusBadRequest, "Registering with defined ID is not possible using a POST request.")
		return
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

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.Header().Set("Location", id)
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

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
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
			td.ID = params["id"]
			_, err := a.controller.add(td)
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
			w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
			w.Header().Set("Location", td.ID)
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

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
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

	w.Header().Set("Content-Type", "application/ld+json;version="+ApiVersion)
	w.WriteHeader(http.StatusOK)
}

// Lists devices in a DeviceCollection
func (a *HTTPAPI) List(w http.ResponseWriter, req *http.Request) {
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

	tds, total, err := a.controller.list(page, perPage)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	coll := &ThingsCollection{
		Context: ContextURL,
		ID:      a.id,
		Things:  tds,
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
	page, perPage, err := ParsePagingParams(
		req.Form.Get(GetParamPage), req.Form.Get(GetParamPerPage), MaxPerPage)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Error parsing query parameters:", err.Error())
		return
	}

	tds, total, err := a.controller.filter(path, op, value, page, perPage)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	coll := &ThingsCollection{
		Context: ContextURL,
		ID:      a.id,
		Things:  tds,
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
