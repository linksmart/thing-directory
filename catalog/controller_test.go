// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/pborman/uuid"
)

// DEVICES

func setup() (CatalogController, func(), error) {
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
		return nil, nil, err
	}

	return controller, func() {
		controller.Stop()
		os.RemoveAll(tempDir) // Remove temp files
	}, nil
}

func TestControllerAdd(t *testing.T) {
	t.Log(TestStorageType)
	controller, shutdown, err := setup()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer shutdown()

	// User-defined id
	var d = Device{
		Id:   "device123",
		Name: "my_device",
		Ttl:  100,
	}

	id, err := controller.add(d)
	if err != nil {
		t.Fatalf("Unexpected error on add: %v", err.Error())
	}
	if id != d.Id {
		t.Fatalf("User defined ID is not returned. Getting %v instead of %v\n", id, d.Id)
	}

	_, err = controller.add(d)
	if err == nil {
		t.Error("Didn't get any error when adding a service with non-unique id.")
	}

	// System-generated id
	var d2 = Device{
		Name: "my_device2",
		Ttl:  100,
	}

	id, err = controller.add(d2)
	if err != nil {
		t.Fatalf("Unexpected error on add: %v", err.Error())
	}
	if !strings.HasPrefix(id, "urn:ls_device:") {
		t.Fatalf("System-generated URN doesn't have `urn:ls_device:` as prefix. Getting location: %v\n", id)
	}
}

func TestControllerGet(t *testing.T) {
	t.Log(TestStorageType)
	controller, shutdown, err := setup()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer shutdown()

	var d = Device{
		Name:        "my_device",
		Meta:        map[string]interface{}{"k": "v"},
		Description: "description",
		Ttl:         100,
	}

	id, err := controller.add(d)
	if err != nil {
		t.Fatal("Error adding a device:", err.Error())
	}

	sd, err := controller.get(id)
	if err != nil {
		t.Fatal("Error retrieving device:", err.Error())
	}

	d.Id = id
	d.URL = fmt.Sprintf("%s/%s/%s", TestApiLocation, TypeDevices, d.Id)
	d.Type = ApiDeviceType
	d.Created = sd.Created
	d.Updated = sd.Updated
	d.Expires = sd.Expires
	if !reflect.DeepEqual(d.simplify(), sd) {
		t.Fatalf("Added and retrieved devices are not equal:\n Added:\n%v\n Retrieved:\n%v\n", *d.simplify(), *sd)
	}

	_, err = controller.get("some_id")
	if err != nil {
		switch err.(type) {
		case *NotFoundError:
		// good
		default:
			t.Fatalf("Device doesn't exist. Expected NotFoundError but got %s", err)
		}
	} else {
		t.Fatal("No error when retrieving a non-existing device.")
	}
}

func TestControllerUpdate(t *testing.T) {
	t.Log(TestStorageType)
	controller, shutdown, err := setup()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer shutdown()

	var d = Device{
		Name:        "my_device",
		Meta:        map[string]interface{}{"k": "v"},
		Description: "description",
		Ttl:         100,
	}

	id, err := controller.add(d)
	if err != nil {
		t.Fatal("Error adding a device:", err.Error())
	}

	// Change
	d.Id = id
	d.URL = fmt.Sprintf("%s/%s/%s", TestApiLocation, TypeDevices, d.Id)
	d.Name = "changed"
	d.Meta = map[string]interface{}{"k": "changed"}
	d.Description = "changed"
	d.Ttl = 110

	err = controller.update(d.Id, d)
	if err != nil {
		t.Fatal("Error updating device:", err.Error())
	}

	sd, err := controller.get(id)
	if err != nil {
		t.Fatal("Error retrieving device:", err.Error())
	}

	d.Type = ApiDeviceType
	d.Created = sd.Created
	d.Updated = sd.Updated
	d.Expires = sd.Expires
	if !reflect.DeepEqual(d.simplify(), sd) {
		t.Fatalf("Updates were not applied or returned.\n Expected:\n%v\n Returned\n%v\n", *d.simplify(), *sd)
	}
}

func TestControllerDelete(t *testing.T) {
	t.Log(TestStorageType)
	controller, shutdown, err := setup()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer shutdown()

	var d = Device{
		Name:        "my_device",
		Meta:        map[string]interface{}{"k": "v"},
		Description: "description",
		Ttl:         100,
	}

	id, err := controller.add(d)
	if err != nil {
		t.Fatal("Error adding a device:", err.Error())
	}

	err = controller.delete(id)
	if err != nil {
		t.Fatal("Error deleting device:", err.Error())
	}

	err = controller.delete(id)
	if err != nil {
		switch err.(type) {
		case *NotFoundError:
		// good
		default:
			t.Fatalf("Device was deleted. Expected NotFoundError but got %s", err)
		}
	} else {
		t.Fatal("No error when deleting a deleted device:", err.Error())
	}

	_, err = controller.get(id)
	if err != nil {
		switch err.(type) {
		case *NotFoundError:
			// good
		default:
			t.Fatalf("Device was deleted. Expected NotFoundError but got %s", err)
		}
	} else {
		t.Fatal("No error when retrieving a deleted device")
	}
}

func TestControllerList(t *testing.T) {
	t.Log(TestStorageType)
	controller, shutdown, err := setup()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer shutdown()

	var storedDevices []SimpleDevice
	for i := 0; i < 5; i++ {
		d := Device{
			Name:        "my_device",
			Meta:        map[string]interface{}{"k": "v"},
			Description: "description",
		}

		id, err := controller.add(d)
		if err != nil {
			t.Fatal("Error adding a device:", err.Error())
		}
		sd, err := controller.get(id)
		if err != nil {
			t.Fatal("Error retrieving device:", err.Error())
		}

		storedDevices = append(storedDevices, *sd)
	}

	var catalogedDevices []SimpleDevice
	perPage := 3
	for page := 1; ; page++ {
		devicesInPage, total, err := controller.list(page, perPage)
		if err != nil {
			t.Fatal("Error getting list of devices:", err.Error())
		}

		if page == 1 && len(devicesInPage) != 3 {
			t.Fatalf("Page 1 has %d entries instead of 3\n", len(devicesInPage))
		}
		if page == 2 && len(devicesInPage) != 2 {
			t.Fatalf("Page 2 has %d entries instead of 2\n", len(devicesInPage))
		}
		if page == 3 && len(devicesInPage) != 0 {
			t.Fatalf("Page 3 has %d entries instead of being blank\n", len(devicesInPage))
		}

		catalogedDevices = append(catalogedDevices, devicesInPage...)

		if page*perPage >= total {
			break
		}
	}

	if len(catalogedDevices) != 5 {
		t.Fatalf("Catalog contains %d entries instead of 5\n", len(catalogedDevices))
	}

	for i, sd := range catalogedDevices {
		if !reflect.DeepEqual(storedDevices[i], sd) {
			t.Fatalf("Device listed in catalog is different with the one stored:\n Stored:\n%v\n Listed\n%v\n",
				storedDevices[i], sd)
		}
	}
}

func TestControllerFilter(t *testing.T) {
	t.Log(TestStorageType)
	controller, shutdown, err := setup()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer shutdown()

	for i := 0; i < 5; i++ {
		d := Device{
			Name:        "my_device",
			Meta:        map[string]interface{}{"k": "v"},
			Description: "description",
		}

		_, err := controller.add(d)
		if err != nil {
			t.Fatal("Error adding a device:", err.Error())
		}
	}

	controller.add(Device{
		Name:        "my_device",
		Meta:        map[string]interface{}{"k": "v"},
		Description: "interesting",
	})
	controller.add(Device{
		Name:        "my_device",
		Meta:        map[string]interface{}{"k": "v"},
		Description: "interesting",
	})

	devices, total, err := controller.filter("description", "equals", "interesting", 1, 10)
	if err != nil {
		t.Fatal("Error filtering devices:", err.Error())
	}
	if total != 2 {
		t.Fatalf("Returned %d instead of 2 devices when filtering description=interesting: \n%v", total, devices)
	}
	for _, d := range devices {
		if d.Description != "interesting" {
			t.Fatal("Wrong results when filtering description=interesting:\n", d)
		}
	}
}

func TestControllerTotal(t *testing.T) {
	t.Log(TestStorageType)
	controller, shutdown, err := setup()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer shutdown()

	for i := 0; i < 5; i++ {
		d := Device{
			Name: "my_device",
		}

		_, err := controller.add(d)
		if err != nil {
			t.Fatal("Error adding a device:", err.Error())
		}
	}

	total, err := controller.total()
	if err != nil {
		t.Fatal("Error getting total of devices:", err.Error())
	}
	if total != 5 {
		t.Fatal("Expected total 5 but got:", total)
	}
}

func TestControllerCleanExpired(t *testing.T) {
	t.Log(TestStorageType)
	controller, shutdown, err := setup()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer shutdown()

	var d = Device{
		Name: "my_device",
		Ttl:  1,
		Resources: []Resource{
			Resource{
				Id:        "my_resource_id",
				Protocols: []Protocol{Protocol{Type: "REST", Endpoint: map[string]interface{}{"url": "http://localhost:9000/api"}}},
			},
		},
	}

	id, err := controller.add(d)
	if err != nil {
		t.Fatal("Error adding a device:", err.Error())
	}

	addingTime := time.Now()
	time.Sleep(6 * time.Second)

	checkingTime := time.Now()
	dd, err := controller.get(id)
	if err != nil {
		switch err.(type) {
		case *NotFoundError:
		// good
		default:
			t.Fatalf("Got an error other than NotFoundError when getting an expired device: %s\n", err)
		}
	} else {
		t.Fatalf("Device was not removed after 1 seconds. \nTTL: %v \nCreated: %v \nExpiry: %v \nNot deleted after: %v at %v\n",
			dd.Ttl,
			dd.Created,
			dd.Expires,
			checkingTime.Sub(addingTime),
			checkingTime.UTC(),
		)
	}

	// Make sure that resource is removed
	_, err = controller.getResource("my_resource_id")
	if err != nil {
		switch err.(type) {
		case *NotFoundError:
		// good
		default:
			t.Fatalf("Got an error other than NotFoundError when getting the resource of an expired device: %s", err)
		}
	} else {
		t.Fatal("Resource of an expired device is not removed.")
	}
}

// RESOURCES

func TestControllerGetResources(t *testing.T) {
	t.Log(TestStorageType)
	controller, shutdown, err := setup()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer shutdown()

	var d = Device{
		Resources: []Resource{
			Resource{
				Id:   "my_resource_id",
				Name: "my_resource",
				Meta: map[string]interface{}{"k": "v"},
				Protocols: []Protocol{Protocol{
					Type:         "REST",
					Endpoint:     map[string]interface{}{"url": "http://localhost:9000/rest/device/resource"},
					Methods:      []string{"GET"},
					ContentTypes: []string{"application/senml+json"},
				}},
				Representation: map[string]interface{}{"application/senml+json": ""},
			},
		},
	}

	id, err := controller.add(d)
	if err != nil {
		t.Fatal("Error adding a device:", err.Error())
	}

	resource, err := controller.getResource("my_resource_id")
	if err != nil {
		t.Fatal("Error retrieving a resource:", err.Error())
	}

	added := d.Resources[0]
	added.URL = fmt.Sprintf("%s/%s/%s", TestApiLocation, TypeResources, added.Id)
	added.Device = fmt.Sprintf("%s/%s/%s", TestApiLocation, TypeDevices, id)
	if !reflect.DeepEqual(*resource, d.Resources[0]) {
		t.Fatalf("Added resource is different with the one retrieved.\n Added:\n%v\n Retrieved\n%v\n",
			added, *resource)
	}

	// Test NotFoundError
	_, err = controller.getResource("some_id")
	if err != nil {
		switch err.(type) {
		case *NotFoundError:
		// good
		default:
			t.Fatalf("Resource doesn't exist. Expected NotFoundError but got %s", err)
		}
	} else {
		t.Fatal("No error when retrieving a non-existing resource")
	}

	// Test deletion of resource
	err = controller.delete(id)
	if err != nil {
		t.Fatal("Error deleting a device:", err.Error())
	}
	_, err = controller.getResource("my_resource_id")
	if err != nil {
		switch err.(type) {
		case *NotFoundError:
		// good
		default:
			t.Fatalf("Device was deleted. Expected NotFoundError when getting its resource but got %s", err)
		}
	} else {
		t.Fatal("No error when retrieving a resource from a deleted device.")
	}
}

func TestControllerListResources(t *testing.T) {
	t.Log(TestStorageType)
	controller, shutdown, err := setup()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer shutdown()

	var storedResources []Resource
	for i := 1; i < 6; i += 2 {
		d := Device{
			Resources: []Resource{
				Resource{
					Id:        fmt.Sprint(i - 1),
					Name:      fmt.Sprintf("my_resource_%d", i),
					Meta:      map[string]interface{}{"k": "v"},
					Protocols: []Protocol{Protocol{Type: "REST", Endpoint: map[string]interface{}{"url": ""}}},
				},
				Resource{
					Id:        fmt.Sprint(i),
					Name:      fmt.Sprintf("my_resource_%d", i+1),
					Meta:      map[string]interface{}{"k": "v"},
					Protocols: []Protocol{Protocol{Type: "REST", Endpoint: map[string]interface{}{"url": ""}}},
				},
			},
		}

		id, err := controller.add(d)
		if err != nil {
			t.Fatal("Error adding a device:", err.Error())
		}

		for _, r := range d.Resources {
			r.URL = fmt.Sprintf("%s/%s/%s", TestApiLocation, TypeResources, r.Id)
			r.Device = fmt.Sprintf("%s/%s/%s", TestApiLocation, TypeDevices, id)
			storedResources = append(storedResources, r)
		}
	}

	var catalogedResources []Resource
	perPage := 4
	for page := 1; ; page++ {
		resourcesInPage, total, err := controller.listResources(page, perPage)
		if err != nil {
			t.Fatal("Error getting list of devices:", err.Error())
		}

		if page == 1 && len(resourcesInPage) != 4 {
			t.Fatalf("Page 1 has %d entries instead of 4\n", len(resourcesInPage))
		}
		if page == 2 && len(resourcesInPage) != 2 {
			t.Fatalf("Page 2 has %d entries instead of 2\n", len(resourcesInPage))
		}
		if page == 3 && len(resourcesInPage) != 0 {
			t.Fatalf("Page 3 has %d entries instead of being blank\n", len(resourcesInPage))
		}

		catalogedResources = append(catalogedResources, resourcesInPage...)

		if page*perPage >= total {
			break
		}
	}

	if len(catalogedResources) != 6 {
		t.Fatalf("Catalog contains %d resources instead of 6\n", len(catalogedResources))
	}

	for i, sr := range catalogedResources {
		if !reflect.DeepEqual(storedResources[i], sr) {
			t.Fatalf("Device listed in catalog is different with the one stored:\n Stored:\n%v\n Listed\n%v\n",
				storedResources[i], sr)
		}
	}
}

func TestControllerFilterResources(t *testing.T) {
	t.Log(TestStorageType)
	controller, shutdown, err := setup()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer shutdown()

	for i := 0; i < 5; i++ {
		_, err := controller.add(Device{
			Resources: []Resource{
				Resource{
					Name:      fmt.Sprintf("boring_%d", i),
					Protocols: []Protocol{Protocol{Type: "REST", Endpoint: map[string]interface{}{"url": ""}}},
				},
			},
		})
		if err != nil {
			t.Fatal("Error adding a device:", err.Error())
		}
	}

	controller.add(Device{
		Resources: []Resource{
			Resource{
				Name:      "interesting_1",
				Protocols: []Protocol{Protocol{Type: "REST", Endpoint: map[string]interface{}{"url": ""}}},
			},
		},
	})
	controller.add(Device{
		Resources: []Resource{
			Resource{
				Name:      "interesting_2",
				Protocols: []Protocol{Protocol{Type: "REST", Endpoint: map[string]interface{}{"url": ""}}},
			},
		},
	})

	resources, total, err := controller.filterResources("name", "prefix", "interesting", 1, 10)
	if err != nil {
		t.Fatal("Error filtering resources:", err.Error())
	}
	if total != 2 {
		t.Fatalf("Returned %d instead of 2 resources when filtering name/prefix/interesting: \n%v", total, resources)
	}
	for _, r := range resources {
		if !strings.Contains(r.Name, "interesting") {
			t.Fatal("Wrong results when filtering name/prefix/interesting:\n", r)
		}
	}
}

func TestControllerTotalResources(t *testing.T) {
	t.Log(TestStorageType)
	controller, shutdown, err := setup()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer shutdown()

	for i := 0; i < 5; i++ {
		_, err := controller.add(Device{
			Resources: []Resource{
				Resource{
					Name:      fmt.Sprintf("resource_%d", i),
					Protocols: []Protocol{Protocol{Type: "REST", Endpoint: map[string]interface{}{"url": ""}}},
				},
			},
		})
		if err != nil {
			t.Fatal("Error adding a device:", err.Error())
		}
	}

	total, err := controller.totalResources()
	if err != nil {
		t.Fatal("Error getting total of resources:", err.Error())
	}
	if total != 5 {
		t.Fatal("Expected total 5 resources but got:", total)
	}
}
