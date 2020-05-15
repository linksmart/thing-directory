// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

//
//import (
//	"bytes"
//	"encoding/json"
//	"fmt"
//	"io/ioutil"
//	"net/http"
//	"net/http/httptest"
//	"os"
//	"reflect"
//	"strings"
//	"testing"
//
//	"github.com/gorilla/mux"
//	"github.com/pborman/uuid"
//)
//
//func setupRouter() (*mux.Router, func(), error) {
//	var (
//		storage CatalogStorage
//		err     error
//		tempDir string = fmt.Sprintf("%s/lslc/test-%s.ldb",
//			strings.Replace(os.TempDir(), "\\", "/", -1), uuid.New())
//	)
//	switch TestStorageType {
//	case CatalogBackendMemory:
//		storage = NewMemoryStorage()
//	case CatalogBackendLevelDB:
//		storage, err = NewLevelDBStorage(tempDir, nil)
//		if err != nil {
//			return nil, nil, err
//		}
//	}
//
//	controller, err := NewController(storage, TestApiLocation)
//	if err != nil {
//		storage.Close()
//		return nil, nil, fmt.Errorf("Failed to start the controller: %v", err.Error())
//	}
//
//	api := NewWritableCatalogAPI(
//		controller,
//		TestApiLocation,
//		TestStaticLocation,
//		"Test catalog",
//	)
//
//	r := mux.NewRouter().StrictSlash(true)
//	// Devices
//	r.Methods("GET").Path(TestApiLocation + "/devices/{id}").HandlerFunc(api.Get)
//	r.Methods("POST").Path(TestApiLocation + "/devices/").HandlerFunc(api.Post)
//	r.Methods("PUT").Path(TestApiLocation + "/devices/{id}").HandlerFunc(api.Put)
//	r.Methods("DELETE").Path(TestApiLocation + "/devices/{id}").HandlerFunc(api.Delete)
//	// Listing, filtering
//	r.Methods("GET").Path(TestApiLocation + "/devices").HandlerFunc(api.List)
//	r.Methods("GET").Path(TestApiLocation + "/devices/{path}/{op}/{value:.*}").HandlerFunc(api.Filter)
//	// Resources
//	r.Methods("GET").Path(TestApiLocation + "/resources").HandlerFunc(api.ListResources)
//	r.Methods("GET").Path(TestApiLocation + "/resources/{id}").HandlerFunc(api.GetResource)
//	r.Methods("GET").Path(TestApiLocation + "/resources/{path}/{op}/{value:.*}").HandlerFunc(api.FilterResources)
//
//	return r, func() {
//		controller.Stop()
//		os.RemoveAll(tempDir) // Remove temp files
//	}, nil
//}
//
//func mockedDevice(id, rid string) *Device {
//	return &Device{
//		Id:          "device_" + id,
//		URL:         fmt.Sprintf("%s/%s/%s", TestApiLocation, TypeDevices, "device_"+id),
//		Type:        ApiDeviceType,
//		Name:        "TestDevice" + id,
//		Meta:        map[string]interface{}{"test-id": id},
//		Description: "Test Device",
//		Ttl:         30,
//		Resources: []Resource{
//			Resource{
//				Id:   "resource_" + rid,
//				URL:  fmt.Sprintf("%s/%s/%s", TestApiLocation, TypeResources, "resource_"+rid),
//				Type: ApiResourceType,
//				Name: "TestResource",
//				Meta: map[string]interface{}{"test-id-resource": id},
//				Protocols: []Protocol{Protocol{
//					Type:         "REST",
//					Endpoint:     map[string]interface{}{"url": "http://localhost:9000/rest/device/resource"},
//					Methods:      []string{"GET"},
//					ContentTypes: []string{"application/senml+json"},
//				}},
//				Representation: map[string]interface{}{"application/senml+json": ""},
//			},
//		},
//	}
//}
//
//// DEVICES
//
//func TestCreate(t *testing.T) {
//	router, shutdown, err := setupRouter()
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//	ts := httptest.NewServer(router)
//	defer ts.Close()
//	defer shutdown()
//
//	device := mockedDevice("1", "10")
//	device.Id = ""
//	b, _ := json.Marshal(device)
//
//	// Create
//	url := ts.URL + TestApiLocation + "/devices/"
//	t.Log("Calling POST", url)
//	res, err := http.Post(url, "application/ld+json", bytes.NewReader(b))
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//
//	if res.StatusCode != http.StatusCreated {
//		t.Fatalf("Server should return %v, got instead: %v (%s)", http.StatusCreated, res.StatusCode, res.Status)
//	}
//
//	if !strings.HasPrefix(res.Header.Get("Content-Type"), "application/ld+json") {
//		t.Fatalf("Response should have Content-Type: application/ld+json, got instead %s", res.Header.Get("Content-Type"))
//	}
//
//	// Check if system-generated id is in response
//	location, err := res.Location()
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//	parts := strings.Split(location.String(), "/")
//	if !strings.HasPrefix(parts[len(parts)-1], "urn:ls_device:") {
//		t.Fatalf("System-generated URN doesn't have `urn:ls_device:` as prefix. Getting location: %v\n", location.String())
//	}
//
//	// Retrieve whole collection
//	t.Log("Calling GET", ts.URL+TestApiLocation)
//	res, err = http.Get(ts.URL + TestApiLocation + "/devices/")
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//
//	var collection *DeviceCollection
//	decoder := json.NewDecoder(res.Body)
//	defer res.Body.Close()
//
//	err = decoder.Decode(&collection)
//
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//
//	if collection.Total != 1 {
//		t.Fatal("Server should return collection with exactly 1 resource, but got total", collection.Total)
//	}
//}
//
//func TestRetrieve(t *testing.T) {
//	router, shutdown, err := setupRouter()
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//	ts := httptest.NewServer(router)
//	defer ts.Close()
//	defer shutdown()
//
//	mockedDevice := mockedDevice("1", "10")
//	b, _ := json.Marshal(mockedDevice)
//
//	// Create
//	url := ts.URL + TestApiLocation + "/devices/" + mockedDevice.Id
//	t.Log("Calling PUT", url)
//	res, err := httpPut(url, bytes.NewReader(b))
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//
//	// Retrieve: device
//	t.Log("Calling GET", url)
//	res, err = http.Get(url)
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//
//	if res.StatusCode != http.StatusOK {
//		t.Fatalf("Server should return %v, got instead: %v (%s)", http.StatusOK, res.StatusCode, res.Status)
//	}
//
//	if !strings.HasPrefix(res.Header.Get("Content-Type"), "application/ld+json") {
//		t.Fatalf("Response should have Content-Type: application/ld+json, got instead %s", res.Header.Get("Content-Type"))
//	}
//
//	var retrievedDevice *SimpleDevice
//	decoder := json.NewDecoder(res.Body)
//	defer res.Body.Close()
//
//	err = decoder.Decode(&retrievedDevice)
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//
//	if !strings.HasPrefix(retrievedDevice.URL, TestApiLocation) {
//		t.Fatalf("Device URL should have been prefixed with %v by catalog, retrieved URL: %v", TestApiLocation, retrievedDevice.URL)
//	}
//
//	simple := mockedDevice.simplify()
//	simple.Created = retrievedDevice.Created
//	simple.Updated = retrievedDevice.Updated
//	simple.Expires = retrievedDevice.Expires
//	simple.Device.Resources = nil
//	if !reflect.DeepEqual(simple, retrievedDevice) {
//		t.Fatalf("The retrieved device is not the same as the added one:\n Added:\n %v \n Retrieved: \n %v", *mockedDevice.simplify(), *retrievedDevice)
//	}
//}
//
//func TestUpdate(t *testing.T) {
//	router, shutdown, err := setupRouter()
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//	ts := httptest.NewServer(router)
//	defer ts.Close()
//	defer shutdown()
//
//	mockedDevice1 := mockedDevice("1", "10")
//	b, _ := json.Marshal(mockedDevice1)
//
//	// Create
//	url := ts.URL + TestApiLocation + "/devices/" + mockedDevice1.Id
//	t.Log("Calling PUT", url)
//	res, err := httpPut(url, bytes.NewReader(b))
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//
//	// Update
//	mockedDevice2 := mockedDevice("1", "10")
//	mockedDevice2.Description = "Updated Test Device"
//	b, _ = json.Marshal(mockedDevice2)
//
//	t.Log("Calling PUT", url)
//	res, err = httpPut(url, bytes.NewReader(b))
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//
//	if res.StatusCode != http.StatusOK {
//		body, err := ioutil.ReadAll(res.Body)
//		if err != nil {
//			t.Fatal(err.Error())
//		}
//		t.Log(string(body))
//		t.Fatalf("Server should return %v, got instead: %v (%s)", http.StatusCreated, res.StatusCode, res.Status)
//	}
//
//	// Retrieve & compare
//	t.Log("Calling GET", url)
//	res, err = http.Get(url)
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//
//	var retrievedDevice *SimpleDevice
//	decoder := json.NewDecoder(res.Body)
//	defer res.Body.Close()
//
//	err = decoder.Decode(&retrievedDevice)
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//
//	simple := mockedDevice2.simplify()
//	simple.Created = retrievedDevice.Created
//	simple.Updated = retrievedDevice.Updated
//	simple.Expires = retrievedDevice.Expires
//	simple.Device.Resources = nil
//	if !reflect.DeepEqual(simple, retrievedDevice) {
//		t.Fatalf("The retrieved device is not the same as the added one:\n Added:\n %v \n Retrieved: \n %v", simple, retrievedDevice)
//	}
//
//	// Create with user-defined ID (PUT for creation)
//	mockedDevice3 := mockedDevice("1", "11")
//	mockedDevice3.Id = ""
//	b, _ = json.Marshal(mockedDevice3)
//	url = ts.URL + TestApiLocation + "/devices/" + "device123"
//	t.Log("Calling PUT", url)
//	res, err = httpPut(url, bytes.NewReader(b))
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//
//	if res.StatusCode != http.StatusCreated {
//		body, err := ioutil.ReadAll(res.Body)
//		if err != nil {
//			t.Fatal(err.Error())
//		}
//		t.Log(string(body))
//		t.Fatalf("Server should return %v, got instead: %v (%s)", http.StatusCreated, res.StatusCode, res.Status)
//	}
//
//	// Check if user-defined id is in response
//	location, err := res.Location()
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//	parts := strings.Split(location.String(), "/")
//	if parts[len(parts)-1] != "device123" {
//		t.Fatalf("User-defined id is not returned in location. Getting %v\n", location.String())
//	}
//}
//
//func TestDelete(t *testing.T) {
//	router, shutdown, err := setupRouter()
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//	ts := httptest.NewServer(router)
//	defer ts.Close()
//	defer shutdown()
//
//	device := mockedDevice("1", "10")
//	b, _ := json.Marshal(device)
//
//	// Create
//	url := ts.URL + TestApiLocation + "/devices/" + device.Id
//	t.Log("Calling POST", url)
//	res, err := httpPut(url, bytes.NewReader(b))
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//
//	// Delete
//	t.Log("Calling DELETE", url)
//	req, err := http.NewRequest("DELETE", url, bytes.NewReader([]byte{}))
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//	res, err = http.DefaultClient.Do(req)
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//
//	if res.StatusCode != http.StatusOK {
//		t.Fatalf("Server should return %v, got instead: %v (%s)", http.StatusOK, res.StatusCode, res.Status)
//	}
//
//	// Retrieve whole collection
//	url = ts.URL + TestApiLocation + "/devices"
//	t.Log("Calling GET", url)
//	res, err = http.Get(url)
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//
//	var collection *DeviceCollection
//	decoder := json.NewDecoder(res.Body)
//	defer res.Body.Close()
//
//	err = decoder.Decode(&collection)
//
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//
//	if collection.Total != 0 {
//		t.Fatal("Server should return an empty collection, but got total", collection.Total)
//	}
//
//}
//
//func TestList(t *testing.T) {
//	router, shutdown, err := setupRouter()
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//	ts := httptest.NewServer(router)
//	defer ts.Close()
//	defer shutdown()
//
//	url := ts.URL + TestApiLocation + "/devices"
//	t.Log("Calling GET", url)
//	res, err := http.Get(url)
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//
//	if res.StatusCode != http.StatusOK {
//		t.Fatalf("Server should return %v, got instead: %v (%s)", http.StatusOK, res.StatusCode, res.Status)
//	}
//
//	if !strings.HasPrefix(res.Header.Get("Content-Type"), "application/ld+json") {
//		t.Fatalf("Response should have Content-Type: application/ld+json, got instead %s", res.Header.Get("Content-Type"))
//	}
//
//	var collection *DeviceCollection
//	decoder := json.NewDecoder(res.Body)
//	defer res.Body.Close()
//
//	err = decoder.Decode(&collection)
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//
//	if collection.Total > 0 {
//		t.Fatal("Server should return empty collection, but got total", collection.Total)
//	}
//}
//
//func TestFilter(t *testing.T) {
//	router, shutdown, err := setupRouter()
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//	ts := httptest.NewServer(router)
//	defer ts.Close()
//	defer shutdown()
//
//	// Create 3 devices
//	url := ts.URL + TestApiLocation + "/devices/"
//	for i := 0; i < 3; i++ {
//		d := mockedDevice(fmt.Sprint(i), fmt.Sprint(i*10))
//		d.Id = ""
//		b, _ := json.Marshal(d)
//
//		_, err := http.Post(url, "application/ld+json", bytes.NewReader(b))
//		if err != nil {
//			t.Fatal(err.Error())
//		}
//	}
//
//	// Devices
//	// Filter many
//	url = ts.URL + TestApiLocation + "/devices/" + "name/" + utils.FOpPrefix + "/" + "Test"
//	t.Log("Calling GET", url)
//	res, err := http.Get(url)
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//
//	var collection *DeviceCollection
//	decoder := json.NewDecoder(res.Body)
//	defer res.Body.Close()
//
//	err = decoder.Decode(&collection)
//
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//
//	if collection.Total != 3 {
//		t.Fatal("Server should return a collection of *3* resources, but got total", collection.Total)
//	}
//}
//
//// RESOURCES
//
//func TestRetrieveResource(t *testing.T) {
//	router, shutdown, err := setupRouter()
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//	ts := httptest.NewServer(router)
//	defer ts.Close()
//	defer shutdown()
//
//	mockedDevice := mockedDevice("1", "10")
//	mockedDevice.Id = ""
//	mockedResource := &mockedDevice.Resources[0]
//	b, _ := json.Marshal(mockedDevice)
//
//	// Create
//	url := ts.URL + TestApiLocation + "/devices/"
//	t.Log("Calling POST", url)
//	res, err := http.Post(url, "application/ld+json", bytes.NewReader(b))
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//
//	// Retrieve: resource
//	url = ts.URL + TestApiLocation + "/resources/" + mockedResource.Id
//	t.Log("Calling GET", url)
//	res, err = http.Get(url)
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//
//	if res.StatusCode != http.StatusOK {
//		t.Fatalf("Server should return %v, got instead: %v (%s)", http.StatusOK, res.StatusCode, res.Status)
//	}
//	if !strings.HasPrefix(res.Header.Get("Content-Type"), "application/ld+json") {
//		t.Fatalf("Response should have Content-Type: application/ld+json, got instead %s", res.Header.Get("Content-Type"))
//	}
//
//	var retrievedResource *Resource
//	decoder := json.NewDecoder(res.Body)
//	defer res.Body.Close()
//
//	err = decoder.Decode(&retrievedResource)
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//
//	if !strings.HasPrefix(retrievedResource.URL, TestApiLocation) {
//		t.Fatalf("Resource URL should have been prefixed with %v by catalog, retrieved URL: %v", TestApiLocation, retrievedResource.URL)
//	}
//
//	mockedResource.Device = retrievedResource.Device
//	if !reflect.DeepEqual(mockedResource, retrievedResource) {
//		t.Fatalf("The retrieved resource is not the same as the added one:\n Added:\n %v \n Retrieved: \n %v", mockedResource, retrievedResource)
//	}
//}
//
//func TestListResources(t *testing.T) {
//	router, shutdown, err := setupRouter()
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//	ts := httptest.NewServer(router)
//	defer ts.Close()
//	defer shutdown()
//
//	// Create 3 devices with 3 resources
//	url := ts.URL + TestApiLocation + "/devices/"
//	for i := 0; i < 3; i++ {
//		d := mockedDevice(fmt.Sprint(i), fmt.Sprint(i*10))
//		d.Id = ""
//		b, _ := json.Marshal(&d)
//
//		_, err := http.Post(url, "application/ld+json", bytes.NewReader(b))
//		if err != nil {
//			t.Fatal(err.Error())
//		}
//	}
//
//	// Filter many
//	url = ts.URL + TestApiLocation + "/resources/" + "name/" + utils.FOpPrefix + "/" + "Test"
//	t.Log("Calling GET", url)
//	res, err := http.Get(url)
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//
//	decoder := json.NewDecoder(res.Body)
//	defer res.Body.Close()
//
//	var collection *ResourceCollection
//	err = decoder.Decode(&collection)
//
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//
//	if collection.Total != 3 {
//		t.Fatal("Server should return a collection of *3* resources, but got total", collection.Total)
//	}
//}
//
//func TestFilterResources(t *testing.T) {
//	router, shutdown, err := setupRouter()
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//	ts := httptest.NewServer(router)
//	defer ts.Close()
//	defer shutdown()
//
//	// Create 3 devices with 3 resources
//	url := ts.URL + TestApiLocation + "/devices/"
//	for i := 0; i < 3; i++ {
//		d := mockedDevice(fmt.Sprint(i), fmt.Sprint(i*10))
//		d.Id = ""
//		b, _ := json.Marshal(d)
//
//		_, err := http.Post(url, "application/ld+json", bytes.NewReader(b))
//		if err != nil {
//			t.Fatal(err.Error())
//		}
//	}
//
//	// Filter many
//	url = ts.URL + TestApiLocation + "/resources/" + "name/" + utils.FOpPrefix + "/" + "Test"
//	t.Log("Calling GET", url)
//	res, err := http.Get(url)
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//
//	decoder := json.NewDecoder(res.Body)
//	defer res.Body.Close()
//
//	var collection *ResourceCollection
//	err = decoder.Decode(&collection)
//
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//
//	if collection.Total != 3 {
//		t.Fatal("Server should return a collection of *3* resources, but got total", collection.Total)
//	}
//}
//
//func httpPut(url string, r *bytes.Reader) (*http.Response, error) {
//	req, err := http.NewRequest("PUT", url, r)
//	if err != nil {
//		return nil, err
//	}
//	res, err := http.DefaultClient.Do(req)
//	if err != nil {
//		return nil, err
//	}
//	return res, nil
//}
