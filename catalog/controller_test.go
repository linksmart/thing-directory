// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/linksmart/thing-directory/wot"
	"github.com/pborman/uuid"
)

type any = interface{}

func setup(t *testing.T) CatalogController {
	var (
		storage Storage
		tempDir = fmt.Sprintf("%s/thing-directory/test-%s-ldb",
			strings.Replace(os.TempDir(), "\\", "/", -1), uuid.New())
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
		t.Logf("Cleaning up...")
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
		if uuid.Parse(id) == nil {
			t.Fatalf("System-generated URN is not a uuid. Got: %s\n", id)
		}
	})
}

func TestControllerGet(t *testing.T) {
	t.Log("Storage Type: " + TestStorageType)
	controller := setup(t)

	t.Run("add and retrieve", func(t *testing.T) {

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

		storedTD, err := controller.get(id)
		if err != nil {
			t.Fatalf("Error retrieving: %s", err)
		}

		storedTD["created"] = td["created"]
		storedTD["modified"] = td["modified"]

		if !reflect.DeepEqual(td, storedTD) {
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

//
//func TestControllerUpdate(t *testing.T) {
//	t.Log(TestStorageType)
//	controller, shutdown, err := setup()
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//	defer shutdown()
//
//	var d = Device{
//		Name:        "my_device",
//		Meta:        map[string]interface{}{"k": "v"},
//		Description: "description",
//		Ttl:         100,
//	}
//
//	id, err := controller.add(d)
//	if err != nil {
//		t.Fatal("Error adding a device:", err.Error())
//	}
//
//	// Change
//	d.Id = id
//	d.URL = fmt.Sprintf("%s/%s/%s", TestApiLocation, TypeDevices, d.Id)
//	d.Name = "changed"
//	d.Meta = map[string]interface{}{"k": "changed"}
//	d.Description = "changed"
//	d.Ttl = 110
//
//	err = controller.update(d.Id, d)
//	if err != nil {
//		t.Fatal("Error updating device:", err.Error())
//	}
//
//	sd, err := controller.get(id)
//	if err != nil {
//		t.Fatal("Error retrieving device:", err.Error())
//	}
//
//	d.Type = ApiDeviceType
//	d.Created = sd.Created
//	d.Updated = sd.Updated
//	d.Expires = sd.Expires
//	if !reflect.DeepEqual(d.simplify(), sd) {
//		t.Fatalf("Updates were not applied or returned.\n Expected:\n%v\n Returned\n%v\n", *d.simplify(), *sd)
//	}
//}
//
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

//
//func TestControllerList(t *testing.T) {
//	t.Log(TestStorageType)
//	controller, shutdown, err := setup()
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//	defer shutdown()
//
//	var storedDevices []SimpleDevice
//	for i := 0; i < 5; i++ {
//		d := Device{
//			Name:        "my_device",
//			Meta:        map[string]interface{}{"k": "v"},
//			Description: "description",
//		}
//
//		id, err := controller.add(d)
//		if err != nil {
//			t.Fatal("Error adding a device:", err.Error())
//		}
//		sd, err := controller.get(id)
//		if err != nil {
//			t.Fatal("Error retrieving device:", err.Error())
//		}
//
//		storedDevices = append(storedDevices, *sd)
//	}
//
//	var catalogedDevices []SimpleDevice
//	perPage := 3
//	for page := 1; ; page++ {
//		devicesInPage, total, err := controller.list(page, perPage)
//		if err != nil {
//			t.Fatal("Error getting list of devices:", err.Error())
//		}
//
//		if page == 1 && len(devicesInPage) != 3 {
//			t.Fatalf("Page 1 has %d entries instead of 3\n", len(devicesInPage))
//		}
//		if page == 2 && len(devicesInPage) != 2 {
//			t.Fatalf("Page 2 has %d entries instead of 2\n", len(devicesInPage))
//		}
//		if page == 3 && len(devicesInPage) != 0 {
//			t.Fatalf("Page 3 has %d entries instead of being blank\n", len(devicesInPage))
//		}
//
//		catalogedDevices = append(catalogedDevices, devicesInPage...)
//
//		if page*perPage >= total {
//			break
//		}
//	}
//
//	if len(catalogedDevices) != 5 {
//		t.Fatalf("Catalog contains %d entries instead of 5\n", len(catalogedDevices))
//	}
//
//	for i, sd := range catalogedDevices {
//		if !reflect.DeepEqual(storedDevices[i], sd) {
//			t.Fatalf("Device listed in catalog is different with the one stored:\n Stored:\n%v\n Listed\n%v\n",
//				storedDevices[i], sd)
//		}
//	}
//}
//
//func TestControllerFilter(t *testing.T) {
//	t.Log(TestStorageType)
//	controller, shutdown, err := setup()
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//	defer shutdown()
//
//	for i := 0; i < 5; i++ {
//		d := Device{
//			Name:        "my_device",
//			Meta:        map[string]interface{}{"k": "v"},
//			Description: "description",
//		}
//
//		_, err := controller.add(d)
//		if err != nil {
//			t.Fatal("Error adding a device:", err.Error())
//		}
//	}
//
//	controller.add(Device{
//		Name:        "my_device",
//		Meta:        map[string]interface{}{"k": "v"},
//		Description: "interesting",
//	})
//	controller.add(Device{
//		Name:        "my_device",
//		Meta:        map[string]interface{}{"k": "v"},
//		Description: "interesting",
//	})
//
//	devices, total, err := controller.filter("description", "equals", "interesting", 1, 10)
//	if err != nil {
//		t.Fatal("Error filtering devices:", err.Error())
//	}
//	if total != 2 {
//		t.Fatalf("Returned %d instead of 2 devices when filtering description=interesting: \n%v", total, devices)
//	}
//	for _, d := range devices {
//		if d.Description != "interesting" {
//			t.Fatal("Wrong results when filtering description=interesting:\n", d)
//		}
//	}
//}
//
//func TestControllerTotal(t *testing.T) {
//	t.Log(TestStorageType)
//	controller, shutdown, err := setup()
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//	defer shutdown()
//
//	for i := 0; i < 5; i++ {
//		d := Device{
//			Name: "my_device",
//		}
//
//		_, err := controller.add(d)
//		if err != nil {
//			t.Fatal("Error adding a device:", err.Error())
//		}
//	}
//
//	total, err := controller.total()
//	if err != nil {
//		t.Fatal("Error getting total of devices:", err.Error())
//	}
//	if total != 5 {
//		t.Fatal("Expected total 5 but got:", total)
//	}
//}
//
func TestControllerCleanExpired(t *testing.T) {
	t.Log("Storage Type: " + TestStorageType)

	// shorten controller's cleanup interval to test quickly
	controllerExpiryCleanupInterval = 2 * time.Second
	const wait = 3 * time.Second

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
