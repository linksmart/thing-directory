// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/linksmart/thing-directory/wot"
	uuid "github.com/satori/go.uuid"
)

type any = interface{}

func setup(t *testing.T) CatalogController {
	var (
		storage Storage
		tempDir = fmt.Sprintf("%s/thing-directory/test-%s-ldb",
			strings.Replace(os.TempDir(), "\\", "/", -1), uuid.NewV4())
	)

	// TODO: use env var
	err := wot.LoadSchema("../wot/wot_td_schema.json")
	if err != nil {
		t.Fatalf("error loading WoT Thing Description schema: %s", err)
	}

	switch TestStorageType {
	case BackendLevelDB:
		storage, err = NewLevelDBStorage(tempDir, nil)
		if err != nil {
			t.Fatalf("error creating leveldb storage: %s", err)
		}
	}

	controller, err := NewController(storage)
	if err != nil {
		storage.Close()
		t.Fatalf("error creating controller: %s", err)
	}

	t.Cleanup(func() {
		//t.Logf("Cleaning up...")
		controller.Stop()
		err = os.RemoveAll(tempDir) // Remove temp files
		if err != nil {
			t.Fatalf("error removing test files: %s", err)
		}
	})

	return controller
}

func TestControllerAdd(t *testing.T) {
	t.Log("Storage Type: " + TestStorageType)
	controller := setup(t)

	t.Run("user-defined ID", func(t *testing.T) {

		var td = map[string]any{
			"@context": "https://www.w3.org/2019/wot/td/v1",
			"id":       "urn:example:test/thing1",
			"title":    "example thing",
			"security": []string{"basic_sc"},
			"securityDefinitions": map[string]any{
				"basic_sc": map[string]string{
					"in":     "header",
					"scheme": "basic",
				},
			},
		}

		id, err := controller.add(td)
		if err != nil {
			t.Fatalf("Unexpected error on add: %s", err)
		}
		if id != td["id"] {
			t.Fatalf("User defined ID is not returned. Getting %s instead of %s\n", id, td["id"])
		}

		// add it again
		_, err = controller.add(td)
		if err == nil {
			t.Error("Didn't get any error when adding a service with non-unique id.")
		}
	})

	t.Run("system-generated ID", func(t *testing.T) {
		// System-generated id
		var td = map[string]any{
			"@context": "https://www.w3.org/2019/wot/td/v1",
			"title":    "example thing",
			"security": []string{"basic_sc"},
			"securityDefinitions": map[string]any{
				"basic_sc": map[string]string{
					"in":     "header",
					"scheme": "basic",
				},
			},
		}

		id, err := controller.add(td)
		if err != nil {
			t.Fatalf("Unexpected error on add: %s", err)
		}
		if !strings.HasPrefix(id, "urn:") {
			t.Fatalf("System-generated ID is not a URN. Got: %s\n", id)
		}
		_, err = uuid.FromString(strings.TrimPrefix(id, "urn:"))
		if err == nil {
			t.Fatalf("System-generated ID is not a uuid. Got: %s\n", id)
		}
	})
}

func TestControllerGet(t *testing.T) {
	t.Log("Storage Type: " + TestStorageType)
	controller := setup(t)

	var td = map[string]any{
		"@context": "https://www.w3.org/2019/wot/td/v1",
		"id":       "urn:example:test/thing1",
		"title":    "example thing",
		"security": []string{"basic_sc"},
		"securityDefinitions": map[string]any{
			"basic_sc": map[string]string{
				"in":     "header",
				"scheme": "basic",
			},
		},
	}

	id, err := controller.add(td)
	if err != nil {
		t.Fatalf("Unexpected error on add: %s", err)
	}

	t.Run("retrieve", func(t *testing.T) {
		storedTD, err := controller.get(id)
		if err != nil {
			t.Fatalf("Error retrieving: %s", err)
		}

		// set system-generated attributes
		storedTD["created"] = td["created"]
		storedTD["modified"] = td["modified"]

		if !SerializedEqual(td, storedTD) {
			t.Fatalf("Added and retrieved TDs are not equal:\n Added:\n%v\n Retrieved:\n%v\n", td, storedTD)
		}
	})

	t.Run("retrieve non-existed", func(t *testing.T) {
		_, err := controller.get("some_id")
		if err != nil {
			switch err.(type) {
			case *NotFoundError:
			// good
			default:
				t.Fatalf("TD doesn't exist. Expected NotFoundError but got %s", err)
			}
		} else {
			t.Fatal("No error when retrieving a non-existed TD.")
		}
	})

}

func TestControllerUpdate(t *testing.T) {
	t.Log("Storage Type: " + TestStorageType)
	controller := setup(t)

	var td = map[string]any{
		"@context": "https://www.w3.org/2019/wot/td/v1",
		"id":       "urn:example:test/thing1",
		"title":    "example thing",
		"security": []string{"basic_sc"},
		"securityDefinitions": map[string]any{
			"basic_sc": map[string]string{
				"in":     "header",
				"scheme": "basic",
			},
		},
	}

	id, err := controller.add(td)
	if err != nil {
		t.Fatalf("Unexpected error on add: %s", err)
	}

	t.Run("update attributes", func(t *testing.T) {
		// Change
		td["title"] = "new title"
		td["description"] = "description of the thing"

		err = controller.update(id, td)
		if err != nil {
			t.Fatal("Error updating TD:", err.Error())
		}

		storedTD, err := controller.get(id)
		if err != nil {
			t.Fatal("Error retrieving TD:", err.Error())
		}

		// set system-generated attributes
		storedTD["created"] = td["created"]
		storedTD["modified"] = td["modified"]

		if !SerializedEqual(td, storedTD) {
			t.Fatalf("Updates were not applied or returned:\n Expected:\n%v\n Retrieved:\n%v\n", td, storedTD)
		}
	})
}

func TestControllerDelete(t *testing.T) {
	t.Log("Storage Type: " + TestStorageType)
	controller := setup(t)

	var td = map[string]any{
		"@context": "https://www.w3.org/2019/wot/td/v1",
		"id":       "urn:example:test/thing1",
		"title":    "example thing",
		"security": []string{"basic_sc"},
		"securityDefinitions": map[string]any{
			"basic_sc": map[string]string{
				"in":     "header",
				"scheme": "basic",
			},
		},
	}

	id, err := controller.add(td)
	if err != nil {
		t.Fatalf("Error adding a TD: %s", err)
	}

	t.Run("delete", func(t *testing.T) {
		err = controller.delete(id)
		if err != nil {
			t.Fatalf("Error deleting TD: %s", err)
		}
	})

	t.Run("delete a deleted TD", func(t *testing.T) {
		err = controller.delete(id)
		if err != nil {
			switch err.(type) {
			case *NotFoundError:
			// good
			default:
				t.Fatalf("TD was deleted. Expected NotFoundError but got %s", err)
			}
		} else {
			t.Fatalf("No error when deleting a deleted TD: %s", err)
		}
	})

	t.Run("retrieve a deleted TD", func(t *testing.T) {
		_, err = controller.get(id)
		if err != nil {
			switch err.(type) {
			case *NotFoundError:
				// good
			default:
				t.Fatalf("TD was deleted. Expected NotFoundError but got %s", err)
			}
		} else {
			t.Fatal("No error when retrieving a deleted TD")
		}
	})
}

func TestControllerList(t *testing.T) {
	t.Log("Storage Type: " + TestStorageType)
	controller := setup(t)

	// add several entries
	var addedTDs []ThingDescription
	for i := 0; i < 5; i++ {
		var td = map[string]any{
			"@context": "https://www.w3.org/2019/wot/td/v1",
			"id":       "urn:example:test/thing_" + strconv.Itoa(i),
			"title":    "example thing",
			"security": []string{"basic_sc"},
			"securityDefinitions": map[string]any{
				"basic_sc": map[string]string{
					"in":     "header",
					"scheme": "basic",
				},
			},
		}

		id, err := controller.add(td)
		if err != nil {
			t.Fatal("Error adding a TD:", err.Error())
		}
		sd, err := controller.get(id)
		if err != nil {
			t.Fatal("Error retrieving TD:", err.Error())
		}

		addedTDs = append(addedTDs, sd)
	}

	// query all pages
	var collection []ThingDescription
	perPage := 3
	for page := 1; ; page++ {
		collectionPage, total, err := controller.list(page, perPage)
		if err != nil {
			t.Fatal("Error getting list of TDs:", err.Error())
		}

		if page == 1 && len(collectionPage) != 3 {
			t.Fatalf("Page 1 has %d entries instead of 3", len(collectionPage))
		}
		if page == 2 && len(collectionPage) != 2 {
			t.Fatalf("Page 2 has %d entries instead of 2", len(collectionPage))
		}
		if page == 3 && len(collectionPage) != 0 {
			t.Fatalf("Page 3 has %d entries instead of being blank", len(collectionPage))
		}

		collection = append(collection, collectionPage...)

		if page*perPage >= total {
			break
		}
	}

	if len(collection) != 5 {
		t.Fatalf("Catalog contains %d entries instead of 5", len(collection))
	}

	// compare added and collection
	for i, sd := range collection {
		if !reflect.DeepEqual(addedTDs[i], sd) {
			t.Fatalf("TDs listed in catalog is different with the one stored:\n Stored:\n%v\n Listed\n%v\n",
				addedTDs[i], sd)
		}
	}
}

func TestControllerFilter(t *testing.T) {
	t.Log("Storage Type: " + TestStorageType)
	controller := setup(t)

	for i := 0; i < 5; i++ {
		var td = map[string]any{
			"@context": "https://www.w3.org/2019/wot/td/v1",
			"id":       "urn:example:test/thing_" + strconv.Itoa(i),
			"title":    "example thing",
			"security": []string{"basic_sc"},
			"securityDefinitions": map[string]any{
				"basic_sc": map[string]string{
					"in":     "header",
					"scheme": "basic",
				},
			},
		}

		_, err := controller.add(td)
		if err != nil {
			t.Fatal("Error adding a TD:", err.Error())
		}
	}

	controller.add(map[string]any{
		"@context": "https://www.w3.org/2019/wot/td/v1",
		"id":       "urn:example:test/thing_x",
		"title":    "interesting thing",
		"security": []string{"basic_sc"},
		"securityDefinitions": map[string]any{
			"basic_sc": map[string]string{
				"in":     "header",
				"scheme": "basic",
			},
		},
	})
	controller.add(map[string]any{
		"@context": "https://www.w3.org/2019/wot/td/v1",
		"id":       "urn:example:test/thing_y",
		"title":    "interesting thing",
		"security": []string{"basic_sc"},
		"securityDefinitions": map[string]any{
			"basic_sc": map[string]string{
				"in":     "header",
				"scheme": "basic",
			},
		},
	})

	t.Run("filter with JSONPath", func(t *testing.T) {
		TDs, total, err := controller.filterJSONPath("$[?(@.title=='interesting thing')]", 1, 10)
		if err != nil {
			t.Fatal("Error filtering:", err.Error())
		}
		if total != 2 {
			t.Fatalf("Returned %d instead of 2 TDs when filtering based on title: \n%v", total, TDs)
		}
		for _, td := range TDs {
			if td.(map[string]any)["title"] != "interesting thing" {
				t.Fatal("Wrong results when filtering based on title:\n", td)
			}
		}
	})

	t.Run("filter with XPath", func(t *testing.T) {
		TDs, total, err := controller.filterXPath("*[title='interesting thing']", 1, 10)
		if err != nil {
			t.Fatal("Error filtering:", err.Error())
		}
		if total != 2 {
			t.Fatalf("Returned %d instead of 2 TDs when filtering based on title: \n%v", total, TDs)
		}
		for _, td := range TDs {
			if td.(map[string]any)["title"] != "interesting thing" {
				t.Fatal("Wrong results when filtering based on title:\n", td)
			}
		}
	})

}

func TestControllerTotal(t *testing.T) {
	t.Log("Storage Type: " + TestStorageType)
	controller := setup(t)

	const createTotal = 5

	for i := 0; i < createTotal; i++ {
		var td = ThingDescription{
			"@context": "https://www.w3.org/2019/wot/td/v1",
			"id":       "urn:example:test/thing_" + strconv.Itoa(i),
			"title":    "example thing",
			"security": []string{"basic_sc"},
			"securityDefinitions": map[string]any{
				"basic_sc": map[string]string{
					"in":     "header",
					"scheme": "basic",
				},
			},
		}

		_, err := controller.add(td)
		if err != nil {
			t.Fatalf("Error adding a TD: %s", err)
		}
	}

	total, err := controller.total()
	if err != nil {
		t.Fatalf("Error getting total of TD: %s", err)
	}
	if total != createTotal {
		t.Fatalf("Expected total %d but got %d", createTotal, total)
	}
}

func TestControllerCleanExpired(t *testing.T) {
	t.Log("Storage Type: " + TestStorageType)

	// shorten controller's cleanup interval to test quickly
	controllerExpiryCleanupInterval = 2 * time.Second
	const wait = 3 * time.Second

	controller := setup(t)

	var td = ThingDescription{
		"@context": "https://www.w3.org/2019/wot/td/v1",
		"id":       "urn:example:test/thing1",
		"title":    "example thing",
		"security": []string{"basic_sc"},
		"securityDefinitions": map[string]any{
			"basic_sc": map[string]string{
				"in":     "header",
				"scheme": "basic",
			},
		},
		"ttl": 1.0, // this should not live long
	}

	id, err := controller.add(td)
	if err != nil {
		t.Fatal("Error adding a TD:", err.Error())
	}

	time.Sleep(wait)

	_, err = controller.get(id)
	if err != nil {
		switch err.(type) {
		case *NotFoundError:
		// good
		default:
			t.Fatalf("Got an error other than NotFoundError when getting an expired TD: %s\n", err)
		}
	} else {
		t.Fatalf("TD was not removed after 1 seconds")
	}

}

func SerializedEqual(td1 ThingDescription, td2 ThingDescription) bool {
	// serialize to ease comparison of interfaces and concrete types
	tdBytes, _ := json.Marshal(td1)
	storedTDBytes, _ := json.Marshal(td2)

	return reflect.DeepEqual(tdBytes, storedTDBytes)
}
