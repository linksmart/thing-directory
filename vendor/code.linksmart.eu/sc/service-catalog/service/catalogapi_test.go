// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/pborman/uuid"
)

func setupRouter() (*mux.Router, func(), error) {
	var (
		storage CatalogStorage
		err     error
		tempDir string = fmt.Sprintf("%s/lslc/test-%s.ldb",
			strings.Replace(os.TempDir(), "\\", "/", -1), uuid.New())
	)
	switch TestStorageType {
	case CatalogBackendMemory:
		storage = NewMemoryStorage()
	case CatalogBackendLevelDB:
		storage, err = NewLevelDBStorage(tempDir, nil)
		if err != nil {
			return nil, nil, err
		}
	}

	controller, err := NewController(storage, TestApiLocation)
	if err != nil {
		storage.Close()
		return nil, nil, fmt.Errorf("Failed to start the controller: %v", err.Error())
	}

	api := NewCatalogAPI(
		controller,
		TestApiLocation,
		TestStaticLocation,
		"Test catalog",
	)

	r := mux.NewRouter().StrictSlash(true)
	// CRUD
	r.Methods("POST").Path(TestApiLocation + "/").HandlerFunc(api.Post)
	r.Methods("GET").Path(TestApiLocation + "/{id:[^/]+/?[^/]*}").HandlerFunc(api.Get)
	r.Methods("PUT").Path(TestApiLocation + "/{id:[^/]+/?[^/]*}").HandlerFunc(api.Put)
	r.Methods("DELETE").Path(TestApiLocation + "/{id:[^/]+/?[^/]*}").HandlerFunc(api.Delete)
	// List, Filter
	r.Methods("GET").Path(TestApiLocation).HandlerFunc(api.List)
	r.Methods("GET").Path(TestApiLocation + "/{path}/{op}/{value:.*}").HandlerFunc(api.Filter)

	return r, func() {
		controller.Stop()
		os.RemoveAll(tempDir) // Remove temp files
	}, nil
}

func mockedService(id string) *Service {
	return &Service{
		Id:          "TestHost/TestService" + id,
		Type:        ApiRegistrationType,
		Name:        "TestService" + id,
		Meta:        map[string]interface{}{"test-id": id},
		Description: "Test Service",
		Protocols: []Protocol{Protocol{
			Type:         "REST",
			Endpoint:     map[string]interface{}{"url": "http://localhost:9000/api"},
			Methods:      []string{"GET"},
			ContentTypes: []string{"application/json"},
		}},
		Representation: map[string]interface{}{"application/json": "{}"},
		Ttl:            30,
	}
}

func sameServices(s1, s2 *Service, checkID bool) bool {
	// Compare IDs if specified
	if checkID {
		if s1.Id != s2.Type {
			return false
		}
	}

	// Compare metadata
	for k1, v1 := range s1.Meta {
		v2, ok := s2.Meta[k1]
		if !ok || v1 != v2 {
			return false
		}
	}
	for k2, v2 := range s2.Meta {
		v1, ok := s1.Meta[k2]
		if !ok || v1 != v2 {
			return false
		}
	}

	// Compare number of protocols
	if len(s1.Protocols) != len(s2.Protocols) {
		return false
	}

	// Compare all other attributes
	if s1.Type != s2.Type || s1.Name != s2.Name || s1.Description != s2.Description || s1.Ttl != s2.Ttl {
		return false
	}

	return true
}

func TestList(t *testing.T) {
	router, shutdown, err := setupRouter()
	if err != nil {
		t.Fatal(err.Error())
	}
	ts := httptest.NewServer(router)
	defer ts.Close()
	defer shutdown()

	url := ts.URL + TestApiLocation
	t.Log("Calling GET", url)
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err.Error())
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Server should return %v, got instead: %v (%s)", http.StatusOK, res.StatusCode, res.Status)
	}

	if !strings.HasPrefix(res.Header.Get("Content-Type"), "application/ld+json") {
		t.Fatalf("Response should have Content-Type: application/ld+json, got instead %s", res.Header.Get("Content-Type"))
	}

	var collection *Collection
	decoder := json.NewDecoder(res.Body)
	defer res.Body.Close()

	err = decoder.Decode(&collection)
	if err != nil {
		t.Fatal(err.Error())
	}

	if collection.Total > 0 {
		t.Fatal("Server should return empty collection, but got total", collection.Total)
	}
}

func TestCreate(t *testing.T) {
	router, shutdown, err := setupRouter()
	if err != nil {
		t.Fatal(err.Error())
	}
	ts := httptest.NewServer(router)
	defer ts.Close()
	defer shutdown()

	service := mockedService("1")
	service.Id = ""
	b, _ := json.Marshal(service)

	// Create
	url := ts.URL + TestApiLocation + "/"
	t.Log("Calling POST", url)
	res, err := http.Post(url, "application/ld+json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err.Error())
	}

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("Server should return %v, got instead: %v (%s)", http.StatusCreated, res.StatusCode, res.Status)
	}

	if !strings.HasPrefix(res.Header.Get("Content-Type"), "application/ld+json") {
		t.Fatalf("Response should have Content-Type: application/ld+json, got instead %s", res.Header.Get("Content-Type"))
	}

	// Check if system-generated id is in response
	location, err := res.Location()
	if err != nil {
		t.Fatal(err.Error())
	}
	parts := strings.Split(location.String(), "/")
	if !strings.HasPrefix(parts[len(parts)-1], "urn:ls_service:") {
		t.Fatalf("System-generated URN doesn't have `urn:ls_service:` as prefix. Getting location: %v\n", location.String())
	}

	// Retrieve whole collection
	t.Log("Calling GET", ts.URL+TestApiLocation)
	res, err = http.Get(ts.URL + TestApiLocation)
	if err != nil {
		t.Fatal(err.Error())
	}

	var collection *Collection
	decoder := json.NewDecoder(res.Body)
	defer res.Body.Close()

	err = decoder.Decode(&collection)

	if err != nil {
		t.Fatal(err.Error())
	}

	if collection.Total != 1 {
		t.Fatal("Server should return collection with exactly 1 resource, but got total", collection.Total)
	}
}

func TestRetrieve(t *testing.T) {
	router, shutdown, err := setupRouter()
	if err != nil {
		t.Fatal(err.Error())
	}
	ts := httptest.NewServer(router)
	defer ts.Close()
	defer shutdown()

	service := mockedService("1")
	b, _ := json.Marshal(service)

	// Create
	url := ts.URL + TestApiLocation + "/" + service.Id
	t.Log("Calling PUT", url)
	res, err := httpPut(url, bytes.NewReader(b))
	if err != nil {
		t.Fatal(err.Error())
	}

	// Retrieve: service
	t.Log("Calling GET", url)
	res, err = http.Get(url)
	if err != nil {
		t.Fatal(err.Error())
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Server should return %v, got instead: %v (%s)", http.StatusOK, res.StatusCode, res.Status)
	}

	if !strings.HasPrefix(res.Header.Get("Content-Type"), "application/ld+json") {
		t.Fatalf("Response should have Content-Type: application/ld+json, got instead %s", res.Header.Get("Content-Type"))
	}

	var service2 *Service
	decoder := json.NewDecoder(res.Body)
	defer res.Body.Close()

	err = decoder.Decode(&service2)
	if err != nil {
		t.Fatal(err.Error())
	}

	if !strings.HasPrefix(service2.URL, TestApiLocation) {
		t.Fatalf("Service URL should have been prefixed with %v by catalog, retrieved URL: %v", TestApiLocation, service2.URL)
	}
	if !sameServices(service, service2, false) {
		t.Fatalf("The retrieved service is not the same as the added one:\n Added:\n %v \n Retrieved: \n %v", service, service2)
	}
}

func TestUpdate(t *testing.T) {
	router, shutdown, err := setupRouter()
	if err != nil {
		t.Fatal(err.Error())
	}
	ts := httptest.NewServer(router)
	defer ts.Close()
	defer shutdown()

	service := mockedService("1")
	b, _ := json.Marshal(service)

	// Create
	url := ts.URL + TestApiLocation + "/" + service.Id
	t.Log("Calling PUT", url)
	res, err := httpPut(url, bytes.NewReader(b))
	if err != nil {
		t.Fatal(err.Error())
	}

	// Update
	service2 := mockedService("1")
	service2.Description = "Updated Test Service"
	b, _ = json.Marshal(service2)

	t.Log("Calling PUT", url)
	res, err = httpPut(url, bytes.NewReader(b))
	if err != nil {
		t.Fatal(err.Error())
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Server should return %v, got instead: %v (%s)", http.StatusOK, res.StatusCode, res.Status)
	}

	// Retrieve & compare
	t.Log("Calling GET", url)
	res, err = http.Get(url)
	if err != nil {
		t.Fatal(err.Error())
	}

	var service3 *Service
	decoder := json.NewDecoder(res.Body)
	defer res.Body.Close()

	err = decoder.Decode(&service3)
	if err != nil {
		t.Fatal(err.Error())
	}

	if !sameServices(service2, service3, false) {
		t.Fatalf("The retrieved service is not the same as the added one:\n Added:\n %v \n Retrieved: \n %v", service2, service3)
	}

	// Create with user-defined ID (PUT for creation)
	service4 := mockedService("1")
	b, _ = json.Marshal(service4)
	url = ts.URL + TestApiLocation + "/service123"
	t.Log("Calling PUT", url)
	res, err = httpPut(url, bytes.NewReader(b))
	if err != nil {
		t.Fatal(err.Error())
	}

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("Server should return %v, got instead: %v (%s)", http.StatusCreated, res.StatusCode, res.Status)
	}

	// Check if user-defined id is in response
	location, err := res.Location()
	if err != nil {
		t.Fatal(err.Error())
	}
	parts := strings.Split(location.String(), "/")
	if parts[len(parts)-1] != "service123" {
		t.Fatalf("User-defined id is not returned in location. Getting %v\n", location.String())
	}
}

func TestDelete(t *testing.T) {
	router, shutdown, err := setupRouter()
	if err != nil {
		t.Fatal(err.Error())
	}
	ts := httptest.NewServer(router)
	defer ts.Close()
	defer shutdown()

	service := mockedService("1")
	b, _ := json.Marshal(service)

	// Create
	url := ts.URL + TestApiLocation + "/" + service.Id
	t.Log("Calling PUT", url)
	res, err := httpPut(url, bytes.NewReader(b))
	if err != nil {
		t.Fatal(err.Error())
	}

	// Delete
	t.Log("Calling DELETE", url)
	req, err := http.NewRequest("DELETE", url, bytes.NewReader([]byte{}))
	if err != nil {
		t.Fatal(err.Error())
	}
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err.Error())
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Server should return %v, got instead: %v (%s)", http.StatusOK, res.StatusCode, res.Status)
	}

	// Retrieve whole collection
	t.Log("Calling GET", ts.URL+TestApiLocation)
	res, err = http.Get(ts.URL + TestApiLocation)
	if err != nil {
		t.Fatal(err.Error())
	}

	var collection *Collection
	decoder := json.NewDecoder(res.Body)
	defer res.Body.Close()

	err = decoder.Decode(&collection)

	if err != nil {
		t.Fatal(err.Error())
	}

	if collection.Total != 0 {
		t.Fatal("Server should return an empty collection, but got total", collection.Total)
	}

}

func TestFilter(t *testing.T) {
	router, shutdown, err := setupRouter()
	if err != nil {
		t.Fatal(err.Error())
	}
	ts := httptest.NewServer(router)
	defer ts.Close()
	defer shutdown()

	// create 3 services
	service1 := mockedService("1")
	service2 := mockedService("2")
	service3 := mockedService("3")

	// Add
	url := ts.URL + TestApiLocation + "/"
	for _, s := range []*Service{service1, service2, service3} {
		s.Id = ""
		b, _ := json.Marshal(s)

		_, err := http.Post(url, "application/ld+json", bytes.NewReader(b))
		if err != nil {
			t.Fatal(err.Error())
		}
	}

	// Services
	// Filter many
	url = ts.URL + TestApiLocation + "/name/" + FOpPrefix + "/" + "Test"
	t.Log("Calling GET", url)
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err.Error())
	}
	defer res.Body.Close()

	var collection *Collection
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&collection)
	if err != nil {
		t.Fatal(err.Error())
	}

	if collection.Total != 3 {
		t.Fatal("Server should return a collection of *3* resources, but got total", collection.Total)
	}

	// Filter one
	url = ts.URL + TestApiLocation + "/name/" + FOpEquals + "/" + service1.Name
	t.Log("Calling GET", url)
	res, err = http.Get(url)
	if err != nil {
		t.Fatal(err.Error())
	}
	defer res.Body.Close()

	var collection2 *Collection
	decoder2 := json.NewDecoder(res.Body)
	err = decoder2.Decode(&collection2)
	if err != nil {
		t.Fatal(err.Error())
	}

	if !sameServices(service1, &collection2.Services[0], false) {
		t.Fatalf("The retrieved service is not the same as the added one:\n Added:\n %v \n Retrieved: \n %v", service1, collection2.Services[0])
	}
}

func httpPut(url string, r *bytes.Reader) (*http.Response, error) {
	req, err := http.NewRequest("PUT", url, r)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}
