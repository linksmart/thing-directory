package resource

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/pborman/uuid"
	utils "linksmart.eu/lc/core/catalog"
	"time"
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
	case utils.CatalogBackendMemory:
		storage = NewMemoryStorage()
	case utils.CatalogBackendLevelDB:
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
	t.Skip("Tested in TestControllerGet")
}

func TestControllerGet(t *testing.T) {
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
	d.URL = fmt.Sprintf("%s/%s/%s", TestApiLocation, FTypeDevices, d.Id)
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
		t.Fatal("No error when retrieving a non-existed device:", err.Error())
	}
}

func TestControllerUpdate(t *testing.T) {
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
	d.URL = fmt.Sprintf("%s/%s/%s", TestApiLocation, FTypeDevices, d.Id)
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
		t.Fatal("No error when retrieving a deleted device:", err.Error())
	}
}

func TestControllerList(t *testing.T) {
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
		t.Fatalf("Catalog contains %d entries instead of 5\n", len(storedDevices))
	}

	for i, sd := range catalogedDevices {
		if !reflect.DeepEqual(storedDevices[i], sd) {
			t.Fatalf("Device listed in catalog is different with the one stored:\n Stored:\n%v\n Listed\n%v\n", storedDevices[i], sd)
		}
	}
}

func TestControllerFilter(t *testing.T) {
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
		t.Fatalf("Returned %d instead of 2 devices when filtering description=interesting: %v", total, devices)
	}
	for _, d := range devices {
		if d.Description != "interesting" {
			t.Fatal("Wrong results when filtering description=interesting:", d)
		}
	}
}

func TestControllerTotal(t *testing.T) {
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
	controller, shutdown, err := setup()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer shutdown()

	var d = Device{
		Name: "my_device",
		Ttl:  1,
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
}

// RESOURCES

func TestControllerGetResources(t *testing.T) {
	t.Skip("Todo")
}

func TestControllerListResources(t *testing.T) {
	t.Skip("Todo")
}

func TestControllerFilterResources(t *testing.T) {
	t.Skip("Todo")
}

func TestControllerTotalResources(t *testing.T) {
	t.Skip("Todo")
}
