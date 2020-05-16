// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/linksmart/thing-directory/wot"
	uuid "github.com/satori/go.uuid"
)

func setupTestHTTPServer(t *testing.T) (CatalogController, *httptest.Server) {
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
	r.Methods("GET").Path("/td/{id:.+}").HandlerFunc(api.Get)
	r.Methods("POST").Path("/td/").HandlerFunc(api.Post)
	r.Methods("PUT").Path("/td/{id:.+}").HandlerFunc(api.Put)
	r.Methods("DELETE").Path("/td/{id:.+}").HandlerFunc(api.Delete)
	// Listing and filtering
	r.Methods("GET").Path("/td").HandlerFunc(api.GetMany)

	httpServer := httptest.NewServer(r)

	t.Cleanup(func() {
		controller.Stop()
		httpServer.Close()
		err = os.RemoveAll(tempDir) // Remove temp files
		if err != nil {
			t.Fatalf("error removing test files: %s", err)
		}
	})

	return controller, httpServer
}

func mockedTD(id string) ThingDescription {
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
	return td
}

func TestPost(t *testing.T) {
	controller, testServer := setupTestHTTPServer(t)

	t.Run("Without ID", func(t *testing.T) {
		td := mockedTD("") // without ID
		b, _ := json.Marshal(td)

		// create over HTTP
		res, err := http.Post(testServer.URL+"/td/", wot.MediaTypeThingDescription, bytes.NewReader(b))
		if err != nil {
			t.Fatalf("Error posting: %s", err)
		}
		defer res.Body.Close()

		b, err = ioutil.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("Error reading response body: %s", err)
		}

		if res.StatusCode != http.StatusCreated {
			t.Fatalf("Expected response %v, got: %d. Reponse body: %s", http.StatusCreated, res.StatusCode, b)
		}

		// Check if system-generated id is in response
		location, err := res.Location()
		if err != nil {
			t.Fatal(err.Error())
		}
		if !strings.Contains(location.String(), "urn:uuid:") {
			t.Fatalf("System-generated ID is not a UUID. Get response location: %s\n", location)
		}

		//// Check if an item is created
		//total, err := controller.total()
		//if err != nil {
		//	t.Fatalf("Error getting total through controller: %s", err)
		//}
		//if total != 1 {
		//	t.Fatalf("Server should contain exactly 1 entry, but got total: %d", total)
		//}

		storedTD, err := controller.get(location.String())
		if err != nil {
			t.Fatalf("Error getting through controller: %s", err)
		}

		// set system-generated attributes
		td["id"] = storedTD["id"]
		td["created"] = storedTD["created"]
		td["modified"] = storedTD["modified"]

		if !SerializedEqual(td, storedTD) {
			t.Fatalf("Posted:\n%v\n Retrieved:\n%v\n", td, storedTD)
		}
	})

	t.Run("With ID", func(t *testing.T) {
		td := mockedTD("urn:example:test/thing_1")
		b, _ := json.Marshal(td)

		// create over HTTP - this should fail
		res, err := http.Post(testServer.URL+"/td/", wot.MediaTypeThingDescription, bytes.NewReader(b))
		if err != nil {
			t.Fatalf("Error posting: %s", err)
		}
		defer res.Body.Close()

		b, err = ioutil.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("Error reading response body: %s", err)
		}

		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("Expected response %v, got: %d. Reponse body: %s", http.StatusBadRequest, res.StatusCode, b)
		}
	})
}

func TestGet(t *testing.T) {
	controller, testServer := setupTestHTTPServer(t)

	// add through controller
	id := "urn:example:test/thing_1"
	td := mockedTD(id)
	_, err := controller.add(td)
	if err != nil {
		t.Fatalf("Error adding through controller: %s", err)
	}

	// retrieve over HTTP
	res, err := http.Get(testServer.URL + "/td/" + id)
	if err != nil {
		t.Fatalf("Error getting TD: %s", err)
	}
	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %s", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected response %v, got: %d. Reponse body: %s", http.StatusOK, res.StatusCode, b)
	}

	mediaType, _, err := mime.ParseMediaType(res.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("Error parsing media type: %s", err)
	}
	if mediaType != ResponseMediaType {
		t.Fatalf("Expected Content-Type: %s, got %s", ResponseMediaType, res.Header.Get("Content-Type"))
	}

	var retrievedTD ThingDescription
	err = json.Unmarshal(b, &retrievedTD)
	if err != nil {
		t.Fatalf("Error decoding body: %s", err)
	}

	// retrieve through controller to compare
	td, err = controller.get(id)
	if err != nil {
		t.Fatalf("Error getting through controller: %s", err)
	}

	if !SerializedEqual(td, retrievedTD) {
		t.Fatalf("The retrieved TD is not the same as the added one:\n Added:\n %v \n Retrieved: \n %v", td, retrievedTD)
	}
}

func TestPut(t *testing.T) {
	controller, testServer := setupTestHTTPServer(t)

	// add through controller
	id := "urn:example:test/thing_1"
	td := mockedTD(id)
	_, err := controller.add(td)
	if err != nil {
		t.Fatalf("Error adding through controller: %s", err)
	}

	t.Run("Update existing", func(t *testing.T) {
		td["title"] = "updated title"
		b, _ := json.Marshal(td)
		// update over HTTP
		res, err := httpDoRequest(http.MethodPut, testServer.URL+"/td/"+id, bytes.NewReader(b))
		if err != nil {
			t.Fatalf("Error putting TD: %s", err)
		}
		defer res.Body.Close()

		b, err = ioutil.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("Error reading response body: %s", err)
		}

		if res.StatusCode != http.StatusOK {
			t.Fatalf("Expected response %v, got: %d. Reponse body: %s", http.StatusOK, res.StatusCode, b)
		}

		storedTD, err := controller.get(id)
		if err != nil {
			t.Fatalf("Error getting through controller: %s", err)
		}

		// set system-generated attributes
		td["modified"] = storedTD["modified"]

		if !SerializedEqual(td, storedTD) {
			t.Fatalf("Posted:\n%v\n Retrieved:\n%v\n", td, storedTD)
		}
	})

	t.Run("Create with ID", func(t *testing.T) {
		id := "urn:example:test/thing_2"
		td := mockedTD(id)
		b, _ := json.Marshal(td)

		// create over HTTP
		res, err := httpDoRequest(http.MethodPut, testServer.URL+"/td/"+id, bytes.NewReader(b))
		if err != nil {
			t.Fatalf("Error putting TD: %s", err)
		}
		defer res.Body.Close()

		b, err = ioutil.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("Error reading response body: %s", err)
		}

		if res.StatusCode != http.StatusCreated {
			t.Fatalf("Expected response %v, got: %d. Reponse body: %s", http.StatusCreated, res.StatusCode, b)
		}

		// retrieve through controller
		storedTD, err := controller.get(id)
		if err != nil {
			t.Fatalf("Error getting through controller: %s", err)
		}

		// set system-generated attributes
		td["created"] = storedTD["created"]
		td["modified"] = storedTD["modified"]

		if !SerializedEqual(td, storedTD) {
			t.Fatalf("Put:\n%v\n Retrieved:\n%v\n", td, storedTD)
		}
	})
}

func TestDelete(t *testing.T) {
	controller, testServer := setupTestHTTPServer(t)

	// add through controller
	id := "urn:example:test/thing_1"
	td := mockedTD(id)
	_, err := controller.add(td)
	if err != nil {
		t.Fatalf("Error adding through controller: %s", err)
	}

	t.Run("Remove existing", func(t *testing.T) {
		// delete over HTTP
		res, err := httpDoRequest(http.MethodDelete, testServer.URL+"/td/"+id, bytes.NewReader(nil))
		if err != nil {
			t.Fatalf("Error deleting TD: %s", err)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Fatalf("Server should return %v, got instead: %d", http.StatusOK, res.StatusCode)
		}

		// retrieve through controller
		_, err = controller.get(id)
		if err == nil {
			t.Fatalf("No error on deleted item.")
		}
	})

	t.Run("Remove non-existing", func(t *testing.T) {
		// delete over HTTP
		res, err := httpDoRequest(http.MethodDelete, testServer.URL+"/td/something-else", bytes.NewReader(nil))
		if err != nil {
			t.Fatalf("Error deleting TD: %s", err)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("Server should return %v, got instead: %d", http.StatusNotFound, res.StatusCode)
		}
	})

}

func TestGetAll(t *testing.T) {
	controller, testServer := setupTestHTTPServer(t)

	for i := 0; i < 3; i++ {
		// add through controller
		td := mockedTD("urn:example:test/thing_" + strconv.Itoa(i))
		_, err := controller.add(td)
		if err != nil {
			t.Fatalf("Error adding through controller: %s", err)
		}
	}

	t.Run("Response total", func(t *testing.T) {
		res, err := http.Get(testServer.URL + "/td")
		if err != nil {
			t.Fatalf("Error getting TDs: %s", err)
		}
		defer res.Body.Close()

		var collectionPage ThingDescriptionPage
		err = json.NewDecoder(res.Body).Decode(&collectionPage)
		if err != nil {
			t.Fatalf("Error decoding page: %s", err)
		}

		items, ok := collectionPage.Items.([]interface{})
		if !ok {
			t.Fatalf("Items in catalog are not TDs. Got: %v", collectionPage.Items)
		}
		if len(items) != 3 {
			t.Fatalf("Expected 3 items in page, got %d", len(items))
		}
		if collectionPage.Total != 3 {
			t.Fatalf("Expected total value of 3, got %d", collectionPage.Total)
		}
	})

	t.Run("Response headers", func(t *testing.T) {
		res, err := http.Get(testServer.URL + "/td")
		if err != nil {
			t.Fatalf("Error getting TDs: %s", err)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Fatalf("Expected status %v, got: %d", http.StatusOK, res.StatusCode)
		}

		mediaType, _, err := mime.ParseMediaType(res.Header.Get("Content-Type"))
		if err != nil {
			t.Fatalf("Error parsing media type: %s", err)
		}
		if mediaType != ResponseMediaType {
			t.Fatalf("Expected Content-Type: %s, got %s", ResponseMediaType, res.Header.Get("Content-Type"))
		}
	})

	t.Run("Response items", func(t *testing.T) {
		res, err := http.Get(testServer.URL + "/td")
		if err != nil {
			t.Fatalf("Error getting TDs: %s", err)
		}
		defer res.Body.Close()

		var collectionPage ThingDescriptionPage
		err = json.NewDecoder(res.Body).Decode(&collectionPage)
		if err != nil {
			t.Fatalf("Error decoding page: %s", err)
		}

		// get all through controller
		storedTDs, _, err := controller.list(1, 10)
		if err != nil {
			t.Fatal("Error getting list of TDs:", err.Error())
		}

		// compare response and stored
		for i, sd := range collectionPage.Items.([]interface{}) {
			if !reflect.DeepEqual(storedTDs[i], sd) {
				t.Fatalf("TD responded over HTTP is different with the one stored:\n Stored:\n%v\n Listed\n%v\n",
					storedTDs[i], sd)
			}
		}
	})

	t.Run("Filter bad JSONPath", func(t *testing.T) {
		res, err := http.Get(testServer.URL + "/td?jsonpath=*/id")
		if err != nil {
			t.Fatalf("Error getting TDs: %s", err)
		}
		defer res.Body.Close()

		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("Error reading response body: %s", err)
		}
		if res.StatusCode != http.StatusInternalServerError {
			t.Fatalf("Expected status %v, got: %d. Response body:\n%s", http.StatusInternalServerError, res.StatusCode, b)
		}
	})

	t.Run("Filter bad XPath", func(t *testing.T) {
		res, err := http.Get(testServer.URL + "/td?xpath=$[:].id")
		if err != nil {
			t.Fatalf("Error getting TDs: %s", err)
		}
		defer res.Body.Close()

		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("Error reading response body: %s", err)
		}
		if res.StatusCode != http.StatusInternalServerError {
			t.Fatalf("Expected status %v, got: %d. Response body:\n%s", http.StatusInternalServerError, res.StatusCode, b)
		}
	})

	t.Run("Filter multiple paths", func(t *testing.T) {
		res, err := http.Get(testServer.URL + "/td?jsonpath=$[:].id&xpath=*/id")
		if err != nil {
			t.Fatalf("Error getting TDs: %s", err)
		}
		defer res.Body.Close()

		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("Error reading response body: %s", err)
		}
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("Expected status %v, got: %d. Response body:\n%s", http.StatusBadRequest, res.StatusCode, b)
		}
	})
}

func httpDoRequest(method, url string, r *bytes.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, r)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}
