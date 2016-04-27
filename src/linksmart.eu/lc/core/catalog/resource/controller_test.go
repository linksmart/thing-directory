package resource

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/pborman/uuid"

	"reflect"

	utils "linksmart.eu/lc/core/catalog"
	"time"
)

//  DEVICES

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

	controller, err := NewController(storage, "/rc")
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
		Resources:   []Resource{},
	}

	added, err := controller.add(d)
	if err != nil {
		t.Fatal("Error adding a device:", err.Error())
	}

	d.Id = added.Id
	d.URL = added.URL
	d.Type = added.Type
	d.Created = added.Created
	d.Updated = added.Updated
	d.Expires = added.Expires
	db, _ := json.Marshal(d)
	addedb, _ := json.Marshal(added)
	if string(db) != string(addedb) {
		t.Fatalf("Added and returned devices are not equal:\n Added:\n%v\n Returned:\n%v\n", string(db), string(addedb))
	}
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

	added, err := controller.add(d)
	if err != nil {
		t.Fatal("Error adding a device:", err.Error())
	}

	sd, err := controller.get(added.Id)
	if err != nil {
		t.Fatal("Error retrieving device:", err.Error())
	}

	if !reflect.DeepEqual(added, sd) {
		t.Fatalf("Added and retrieved devices are not equal:\n Added:\n%v\n Retrieved:\n%v\n", *added, *sd)
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

	added, err := controller.add(d)
	if err != nil {
		t.Fatal("Error adding a device:", err.Error())
	}

	// Change
	d.Id = added.Id
	d.URL = added.URL
	d.Name = "changed"
	d.Meta = map[string]interface{}{"k": "changed"}
	d.Description = "changed"
	d.Ttl = 110

	updated, err := controller.update(d.Id, d)
	if err != nil {
		t.Fatal("Error updating device:", err.Error())
	}

	if !reflect.DeepEqual(d.simplify(), updated) {
		t.Fatalf("Updates were not applied or returned.\n Expected:\n%v\n Returned\n%v\n", d.simplify(), updated)
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

	added, err := controller.add(d)
	if err != nil {
		t.Fatal("Error adding a device:", err.Error())
	}

	err = controller.delete(added.Id)
	if err != nil {
		t.Fatal("Error deleting device:", err.Error())
	}

	err = controller.delete(added.Id)
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

	_, err = controller.get(added.Id)
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

		sd, err := controller.add(d)
		if err != nil {
			t.Fatal("Error adding a device:", err.Error())
		}

		storedDevices = append(storedDevices, *sd)
	}

	var catalogedDevices []SimpleDevice
	perPage := 3
	for page := 1; ; page++ {
		fmt.Println("1", TestStorageType)
		devicesInPage, total, err := controller.list(page, perPage)
		if err != nil {
			t.Fatal("Error getting list of devices:", err.Error())
		}
		fmt.Println("2")

		if page == 1 && len(devicesInPage) != 3 {
			t.Fatalf("Page 1 has %d entries instead of 3\n", len(devicesInPage))
		}
		if page == 2 && len(devicesInPage) != 2 {
			t.Fatalf("Page 2 has %d entries instead of 2\n", len(devicesInPage))
		}
		if page == 3 && len(devicesInPage) != 0 {
			t.Fatalf("Page 3 has %d entries instead of being blank\n", len(devicesInPage))
		}
		fmt.Println("3")
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
	t.Skip("Todo")
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

	added, err := controller.add(d)
	if err != nil {
		t.Fatal("Error adding a device:", err.Error())
	}

	time.Sleep(1100 * time.Millisecond)

	_, err = controller.get(added.Id)
	if err != nil {
		switch err.(type) {
		case *NotFoundError:
		// good
		default:
			t.Fatalf("Device was not removed after 1 seconds. Got error %s", err)
		}
	} else {
		t.Fatalf("Device was not removed after 1 seconds")
	}
}

//  RESOURCES

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
