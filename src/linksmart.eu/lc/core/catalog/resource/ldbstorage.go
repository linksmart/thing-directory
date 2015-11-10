package resource

import (
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	avl "github.com/ancientlore/go-avltree"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"

	"linksmart.eu/lc/core/catalog"
)

// LevelDB storage
type LevelDBStorage struct {
	db *leveldb.DB
	wg sync.WaitGroup
	sync.RWMutex

	// sorted resourceID->deviceID map
	resDevice *avl.Tree
	// sorted expiryTime->deviceID map
	expDevice *avl.Tree
}

func NewLevelDBStorage(dsn string, opts *opt.Options) (CatalogStorage, func() error, error) {
	url, err := url.Parse(dsn)
	if err != nil {
		return &LevelDBStorage{}, nil, err
	}

	// Open the database file
	db, err := leveldb.OpenFile(url.Path, opts)
	if err != nil {
		return &LevelDBStorage{}, nil, err
	}

	s := &LevelDBStorage{
		db:        db,
		resDevice: avl.New(stringKeys, 0),
		expDevice: avl.New(timeKeys, avl.AllowDuplicates),
	}

	// Create secondary indices
	s.wg.Add(1)
	iter := s.db.NewIterator(nil, nil)
	for iter.Next() {
		var d Device
		err = json.Unmarshal(iter.Value(), &d)
		if err != nil {
			return &LevelDBStorage{}, nil, err
		}
		s.addIndices(&d)
	}
	iter.Release()
	s.wg.Done()
	err = iter.Error()
	if err != nil {
		return &LevelDBStorage{}, nil, err
	}

	// schedule cleaner
	t := time.Tick(time.Duration(5) * time.Second)
	go func() {
		for now := range t {
			s.cleanExpired(now)
		}
	}()

	return s, s.close, nil
}

// CRUD
func (s *LevelDBStorage) add(d Device) error {
	if !d.validate() {
		return errors.New("Invalid Device registration")
	}

	d.Created = time.Now()
	d.Updated = d.Created
	if d.Ttl >= 0 {
		d.Expires = d.Created.Add(time.Duration(d.Ttl) * time.Second)
	}

	// Add device id to resources
	for i := range d.Resources {
		d.Resources[i].Device = d.Id
	}

	// Add to database
	bytes, err := json.Marshal(&d)
	if err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()
	err = s.db.Put([]byte(d.Id), bytes, nil)
	if err != nil {
		return err
	}

	// Create secondary indices
	s.addIndices(&d)

	return nil
}

func (s *LevelDBStorage) get(id string) (Device, error) {
	// Query from database
	bytes, err := s.db.Get([]byte(id), nil)
	if err == leveldb.ErrNotFound {
		return Device{}, ErrorNotFound
	} else if err != nil {
		return Device{}, err
	}

	var d Device
	err = json.Unmarshal(bytes, &d)
	if err != nil {
		return Device{}, err
	}

	return d, nil
}

func (s *LevelDBStorage) update(id string, d Device) error {
	s.Lock()
	defer s.Unlock()
	// Get the stored device
	sd, err := s.get(id)
	if err == leveldb.ErrNotFound {
		return ErrorNotFound
	} else if err != nil {
		return err
	}

	// Remove old secondary indices
	s.removeIndices(&sd)

	sd.Type = d.Type
	sd.Name = d.Name
	sd.Description = d.Description
	sd.Ttl = d.Ttl
	sd.Updated = time.Now()
	if sd.Ttl >= 0 {
		sd.Expires = sd.Updated.Add(time.Duration(sd.Ttl) * time.Second)
	}

	// Update device IDs in resources
	sd.Resources = nil
	for _, res := range d.Resources {
		res.Device = sd.Id
		sd.Resources = append(sd.Resources, res)
	}

	// Add new secondary indices
	s.addIndices(&sd)

	// Store the modified device
	bytes, err := json.Marshal(&sd)
	if err != nil {
		return err
	}
	err = s.db.Put([]byte(id), bytes, nil)
	if err != nil {
		return err
	}

	return nil
}

func (s *LevelDBStorage) delete(id string) error {
	s.Lock()
	defer s.Unlock()

	oldDevice, err := s.get(id)
	if err == leveldb.ErrNotFound {
		return ErrorNotFound
	} else if err != nil {
		return err
	}

	err = s.db.Delete([]byte(id), nil)
	if err != nil {
		return err
	}

	// Remove secondary indices
	s.removeIndices(&oldDevice)

	return nil
}

// Utilities

func (s *LevelDBStorage) getMany(page int, perPage int) ([]Device, int, error) {
	total := s.getResourcesCount() // already mutex locked
	s.RLock()
	defer s.RUnlock()
	// Slice resources map
	keys := make([]string, total)
	for i, x := range s.resDevice.Data() {
		keys[i] = x.(SortedMap).key.(string)
	}
	slice := catalog.GetPageOfSlice(keys, page, perPage, MaxPerPage)

	if len(slice) == 0 {
		return []Device{}, total, nil
	}

	// Retrieve devices that are in the slice
	sliceMap := make(map[string]bool)
	for _, x := range slice {
		sliceMap[x] = true
	}
	deviceExists := make(map[string]bool)
	var devices []Device
	for _, k := range slice {
		deviceID := s.resDevice.Find(SortedMap{key: k}).(SortedMap).value.(string)
		d, err := s.get(deviceID)
		if err != nil {
			return nil, total, err
		}
		_, exists := deviceExists[deviceID]
		if !exists {
			deviceExists[deviceID] = true

			// Remove unneeded resources
			var existing []Resource
			for _, r := range d.Resources {
				_, exists := sliceMap[r.Id]
				if exists {
					existing = append(existing, r)
				}
			}
			d.Resources = existing

			devices = append(devices, d)
		}
	}

	return devices, total, nil
}

func (s *LevelDBStorage) getDevicesCount() int {
	c := 0
	s.wg.Add(1)
	iter := s.db.NewIterator(nil, nil)
	for iter.Next() {
		c++
	}
	iter.Release()
	s.wg.Done()
	err := iter.Error()
	if err != nil {
		logger.Println("LevelDBStorage.getDevicesCount()", err.Error())
		return 0
	}

	return c
}

func (s *LevelDBStorage) getResourcesCount() int {
	s.RLock()
	l := s.resDevice.Len()
	s.RUnlock()
	return l
}

func (s *LevelDBStorage) getResourceById(id string) (Resource, error) {
	s.RLock()
	defer s.RUnlock()

	deviceID := s.resDevice.Find(SortedMap{key: id}).(SortedMap).value.(string)
	device, err := s.get(deviceID)
	if err != nil {
		return Resource{}, err
	}

	for _, res := range device.Resources {
		if res.Id == id {
			return res, nil
		}
	}

	return Resource{}, ErrorNotFound
}

func (s *LevelDBStorage) devicesFromResources(resources []Resource) []Device {
	// Max len(devices) == len(resources)
	devices := make([]Device, 0, len(resources))
	deviceExists := make(map[string]bool)

	for _, r := range resources {
		d, err := s.get(r.Device)
		if err != nil {
			logger.Println("LevelDBStorage.devicesFromResources()", err.Error())
			continue
		}
		_, exists := deviceExists[r.Device]
		if !exists {
			deviceExists[r.Device] = true

			// only take resources that are provided as input
			d.Resources = nil
			for _, r2 := range resources {
				if r2.Device == d.Id {
					d.Resources = append(d.Resources, r2)
				}
			}

			devices = append(devices, d)
		}
	}

	return devices
}

// Path filtering
func (s *LevelDBStorage) pathFilterDevice(path, op, value string) (Device, error) {
	pathTknz := strings.Split(path, ".")

	// return the first one found
	s.wg.Add(1)
	iter := s.db.NewIterator(nil, nil)
	for iter.Next() {
		var d Device
		err := json.Unmarshal(iter.Value(), &d)
		if err != nil {
			iter.Release()
			s.wg.Done()
			return Device{}, err
		}

		matched, err := catalog.MatchObject(d, pathTknz, op, value)
		if err != nil {
			iter.Release()
			s.wg.Done()
			return Device{}, err
		}
		if matched {
			iter.Release()
			s.wg.Done()
			return d, nil
		}
	}
	iter.Release()
	s.wg.Done()
	err := iter.Error()
	if err != nil {
		return Device{}, err
	}

	return Device{}, nil
}

func (s *LevelDBStorage) pathFilterDevices(path, op, value string, page, perPage int) ([]Device, int, error) {
	var matchedIDs []string
	pathTknz := strings.Split(path, ".")

	s.RLock()
	defer s.RUnlock()

	// Iterate over a latest snapshot of the database
	s.wg.Add(1)
	iter := s.db.NewIterator(nil, nil)
	for iter.Next() {
		var d Device
		err := json.Unmarshal(iter.Value(), &d)
		if err != nil {
			iter.Release()
			s.wg.Done()
			return []Device{}, 0, err
		}

		matched, err := catalog.MatchObject(d, pathTknz, op, value)
		if err != nil {
			iter.Release()
			s.wg.Done()
			return []Device{}, 0, err
		}
		if matched {
			matchedIDs = append(matchedIDs, d.Id)
		}
	}
	iter.Release()
	s.wg.Done()
	err := iter.Error()
	if err != nil {
		return []Device{}, 0, err
	}

	// Apply pagination
	slice := catalog.GetPageOfSlice(matchedIDs, page, perPage, MaxPerPage)

	// page/registry is empty
	if len(slice) == 0 {
		return []Device{}, 0, nil
	}

	devs := make([]Device, 0, len(slice))
	for _, id := range slice {
		d, err := s.get(id)
		if err != nil {
			return nil, len(matchedIDs), err
		}
		devs = append(devs, d)
	}

	return devs, len(matchedIDs), nil
}

func (s *LevelDBStorage) pathFilterResource(path, op, value string) (Resource, error) {
	pathTknz := strings.Split(path, ".")

	// Iterate over a latest snapshot of the database
	s.wg.Add(1)
	iter := s.db.NewIterator(nil, nil)
	for iter.Next() {
		var d Device
		err := json.Unmarshal(iter.Value(), &d)
		if err != nil {
			iter.Release()
			s.wg.Done()
			return Resource{}, err
		}

		for _, res := range d.Resources {
			matched, err := catalog.MatchObject(res, pathTknz, op, value)
			if err != nil {
				iter.Release()
				s.wg.Done()
				return Resource{}, err
			}
			if matched {
				iter.Release()
				s.wg.Done()
				return res, nil
			}
		}
	}
	iter.Release()
	s.wg.Done()
	err := iter.Error()
	if err != nil {
		return Resource{}, err
	}

	return Resource{}, nil
}

func (s *LevelDBStorage) pathFilterResources(path, op, value string, page, perPage int) ([]Resource, int, error) {
	pathTknz := strings.Split(path, ".")
	var resourceIDs []string
	resources := make(map[string]Resource)

	// Iterate over a latest snapshot of the database
	s.wg.Add(1)
	iter := s.db.NewIterator(nil, nil)
	for iter.Next() {
		var d Device
		err := json.Unmarshal(iter.Value(), &d)
		if err != nil {
			iter.Release()
			s.wg.Done()
			return []Resource{}, 0, err
		}

		for _, res := range d.Resources {
			matched, err := catalog.MatchObject(res, pathTknz, op, value)
			if err != nil {
				iter.Release()
				s.wg.Done()
				return []Resource{}, 0, err
			}
			if matched {
				resourceIDs = append(resourceIDs, res.Id)
				resources[res.Id] = res
			}
		}
	}
	iter.Release()
	s.wg.Done()
	err := iter.Error()
	if err != nil {
		return []Resource{}, 0, err
	}

	// Slice sorted resources
	sort.Strings(resourceIDs)
	slice := catalog.GetPageOfSlice(resourceIDs, page, perPage, MaxPerPage)
	ress := make([]Resource, 0, len(slice))
	for _, id := range slice {
		ress = append(ress, resources[id])
	}

	return ress, len(resourceIDs), nil
}

// Clean all remote registrations which expire time is larger than the given timestamp
func (s *LevelDBStorage) cleanExpired(timestamp time.Time) {
	s.Lock()
	for m := range s.expDevice.Iter() {
		if !m.(SortedMap).key.(time.Time).After(timestamp) {
			id := m.(SortedMap).value.(string)
			logger.Printf("LevelDBStorage.cleanExpired() Registration %v has expired\n", id)

			oldDevice, err := s.get(id)
			if err != nil {
				continue
			}

			// Remove the device from db
			err = s.db.Delete([]byte(id), nil)
			if err != nil {
				continue
			}

			// Remove resource indices
			for _, r := range oldDevice.Resources {
				s.resDevice.Remove(SortedMap{key: r.Id})
			}

			// Remove expiry index
			s.expDevice.Remove(m)
		} else {
			// expDevice is sorted by time ascendingly,
			//	so they will expire in order
			break
		}
	}
	s.Unlock()
}

// Creates secondary indices
func (s *LevelDBStorage) addIndices(d *Device) {
	for _, r := range d.Resources {
		s.resDevice.Add(SortedMap{r.Id, d.Id})
	}

	// Add expiry time index
	if d.Ttl >= 0 {
		s.expDevice.Add(SortedMap{d.Expires, d.Id})
	}
}

// Removes secondary indices
func (s *LevelDBStorage) removeIndices(d *Device) {
	// Remove resource indices
	for _, r := range d.Resources {
		s.resDevice.Remove(SortedMap{key: r.Id})
	}

	// Remove the expiry time index
	if d.Ttl >= 0 {
		for m := range s.expDevice.Iter() {
			if m.(SortedMap).value.(string) == d.Id {
				s.expDevice.Remove(m)
				break
			}
		}
	}
}

func (s *LevelDBStorage) close() error {
	s.wg.Wait()
	return s.db.Close()
}
