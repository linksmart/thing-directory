// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/linksmart/thing-directory/wot"
	uuid "github.com/satori/go.uuid"
)

func setupTestHTTPServer(t *testing.T) *httptest.Server {
	var (
		storage Storage
		err     error
		tempDir = fmt.Sprintf("%s/lslc/test-%s-ldb",
			strings.Replace(os.TempDir(), "\\", "/", -1), uuid.NewV4())
	)
	switch TestStorageType {
	case BackendLevelDB:
		storage, err = NewLevelDBStorage(tempDir, nil)
		if err != nil {
			t.Fatal(err)
		}
	}

	// TODO: use env var
	err = wot.LoadSchema("../wot/wot_td_schema.json")
	if err != nil {
		t.Fatalf("error loading WoT Thing Description schema: %s", err)
	}

	controller, err := NewController(storage)
	if err != nil {
		storage.Close()
		t.Fatalf("Failed to start the controller: %s", err)
	}

	api := NewHTTPAPI(
		controller,
		"v0.0.0-test",
	)

	r := mux.NewRouter().StrictSlash(true)
	// CRUD
	r.Methods("GET").Path("/td/{id}").HandlerFunc(api.Get)
	r.Methods("POST").Path("/td/").HandlerFunc(api.Post)
	r.Methods("PUT").Path("/td/{id}").HandlerFunc(api.Put)
	r.Methods("DELETE").Path("/td/{id}").HandlerFunc(api.Delete)
	// Listing and filtering
	r.Methods("GET").Path("/td").HandlerFunc(api.List)

	httpServer := httptest.NewServer(r)

	t.Cleanup(func() {
		controller.Stop()
		httpServer.Close()
		err = os.RemoveAll(tempDir) // Remove temp files
		if err != nil {
			t.Fatalf("error removing test files: %s", err)
		}
	})

	return httpServer
}

func mockedTD(id string) []byte {
	var td = map[string]any{
		"@context": "https://www.w3.org/2019/wot/td/v1",
		"title":    "example thing",
		"security": []string{"nosec_sc"},
		"securityDefinitions": map[string]any{
			"nosec_sc": map[string]string{
				"scheme": "nosec",
			},
		},
	}
	if id != "" {
		td["id"] = id
	}
	b, _ := json.Marshal(td)
	return b
}

func TestCreate(t *testing.T) {
	testServer := setupTestHTTPServer(t)

	td := mockedTD("") // without ID

	t.Run("POST a TD", func(t *testing.T) {
		// Create
		url := testServer.URL + "/td/"
		t.Log("Calling POST", url)
		res, err := http.Post(url, wot.MediaTypeThingDescription, bytes.NewReader(td))
		if err != nil {
			t.Fatal(err.Error())
		}

		b, err := ioutil.ReadAll(res.Body)
		if res.StatusCode != http.StatusCreated {
			t.Fatalf("Expected response %v, got: %d. Reponse body: %s", http.StatusCreated, res.StatusCode, b)
		}
		defer res.Body.Close()

		// Check if system-generated id is in response
		location, err := res.Location()
		if err != nil {
			t.Fatal(err.Error())
		}
		if !strings.Contains(location.String(), "urn:uuid:") {
			t.Fatalf("System-generated ID is not a UUID. Get response location: %s\n", location)
		}
	})

	t.Run("Verify the collection", func(t *testing.T) {
		// Retrieve a page
		res, err := http.Get(testServer.URL + "/td/")
		if err != nil {
			t.Fatal(err.Error())
		}
		defer res.Body.Close()

		var collectionPage ThingDescriptionPage
		err = json.NewDecoder(res.Body).Decode(&collectionPage)
		if err != nil {
			t.Fatal(err.Error())
		}

		if collectionPage.Total != 1 {
			t.Fatal("Server should return collection with exactly 1 entry, but got total", collectionPage.Total)
		}
	})
}

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
