package resource

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/pborman/uuid"
)

func setupLevelDB() (CatalogStorage, string, error) {
	tempDir := fmt.Sprintf("%s/lslc/test-%s.ldb",
		strings.Replace(os.TempDir(), "\\", "/", -1), uuid.New())
	storage, err := NewLevelDBStorage(tempDir, nil)
	if err != nil {
		return nil, tempDir, err
	}
	return storage, tempDir, nil
}

func TestLevelDBAddDevice(t *testing.T) {
	storage, tempDir, err := setupLevelDB()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer os.RemoveAll(tempDir)
	defer storage.Close()

	r := &Device{}
	uuid := "E9203BE9-D705-42A8-8B12-F28E7EA2FC99"
	r.Name = "DeviceName"
	r.Id = uuid + "/" + r.Name
	r.Ttl = 30

	//storage := NewMemoryStorage()
	err = storage.add(*r)
	if err != nil {
		t.Errorf("Received unexpected error: %v", err.Error())
	}
}

func TestLevelDBUpdateDevice(t *testing.T) {
	storage, tempDir, err := setupLevelDB()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer os.RemoveAll(tempDir)
	defer storage.Close()

	r := &Device{}
	uuid := "E9203BE9-D705-42A8-8B12-F28E7EA2FC99"
	r.Name = "DeviceName"
	r.Id = uuid + "/" + r.Name
	r.Ttl = 30
	//storage := NewMemoryStorage()

	err = storage.add(*r)
	if err != nil {
		t.Errorf("Unexpected error on add: %v", err.Error())
	}
	ra := r.copy()
	ra.Name = "UpdatedName"
	err = storage.update(ra.Id, ra)
	if err != nil {
		t.Error("Unexpected error on update: %v", err.Error())
	}
}

func TestLevelDBGetDevice(t *testing.T) {
	storage, tempDir, err := setupLevelDB()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer os.RemoveAll(tempDir)
	defer storage.Close()

	r := &Device{
		Name: "TestName",
	}
	uuid := "E9203BE9-D705-42A8-8B12-F28E7EA2FC99"
	r.Name = "DeviceName"
	r.Id = uuid + "/" + r.Name
	r.Ttl = 30
	//storage := NewMemoryStorage()

	err = storage.add(*r)
	if err != nil {
		t.Errorf("Unexpected error on add: %v", err.Error())
	}

	rg, err := storage.get(r.Id)
	if err != nil {
		t.Error("Unexpected error on get: %v", err.Error())
	}

	if rg.Name != r.Name {
		t.Fail()
	}
}

func TestLevelDBDeleteDevice(t *testing.T) {
	storage, tempDir, err := setupLevelDB()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer os.RemoveAll(tempDir)
	defer storage.Close()

	r := &Device{}
	uuid := "E9203BE9-D705-42A8-8B12-F28E7EA2FC99"
	r.Name = "DeviceName"
	r.Id = uuid + "/" + r.Name
	r.Ttl = 30
	//storage := NewMemoryStorage()

	err = storage.add(*r)
	if err != nil {
		t.Errorf("Unexpected error on add: %v", err.Error())
	}
	err = storage.delete(r.Id)
	if err != nil {
		t.Error("Unexpected error on delete: %v", err.Error())
	}

	err = storage.delete(r.Id)
	if err != ErrorNotFound {
		t.Error(err, "The previous call hasn't deleted the Device?")
	}
}

func TestLevelDBGetManyDevices(t *testing.T) {
	storage, tempDir, err := setupLevelDB()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer os.RemoveAll(tempDir)
	defer storage.Close()

	r := Resource{
		Name: "TestResource",
	}
	//storage := NewMemoryStorage()
	// Add 10 entries
	for i := 0; i < 11; i++ {
		d := &Device{
			Name: "TestDevice",
		}
		d.Id = "TestID" + "/" + string(i)
		d.Ttl = 30
		r.Id = d.Id + "/" + r.Name
		d.Resources = append(d.Resources, r)

		err := storage.add(*d)
		if err != nil {
			t.Errorf("Unexpected error on add: %v", err.Error())
		}
	}

	p1pp2, total, _ := storage.getMany(1, 2)
	if total != 11 {
		t.Errorf("Expected total is 11, returned: %v", total)
	}

	if len(p1pp2) != 2 {
		t.Errorf("Wrong number of entries: requested page=1 , perPage=2. Expected: 2, returned: %v", len(p1pp2))
	}

	p2pp2, _, _ := storage.getMany(2, 2)
	if len(p2pp2) != 2 {
		t.Errorf("Wrong number of entries: requested page=2 , perPage=2. Expected: 2, returned: %v", len(p2pp2))
	}

	p2pp5, _, _ := storage.getMany(2, 5)
	if len(p2pp5) != 5 {
		t.Errorf("Wrong number of entries: requested page=2 , perPage=5. Expected: 5, returned: %v", len(p2pp5))
	}

	p4pp3, _, _ := storage.getMany(4, 3)
	if len(p4pp3) != 2 {
		t.Errorf("Wrong number of entries: requested page=4 , perPage=3. Expected: 2, returned: %v", len(p4pp3))
	}
}
