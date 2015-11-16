package service

import (
	"encoding/json"
	"fmt"
	"net/url"
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
		expDevice: avl.New(timeKeys, avl.AllowDuplicates),
	}

	// Create secondary indices
	s.wg.Add(1)
	iter := s.db.NewIterator(nil, nil)
	for iter.Next() {
		var srv Service
		err = json.Unmarshal(iter.Value(), &srv)
		if err != nil {
			return &LevelDBStorage{}, nil, err
		}
		s.addIndices(&srv)
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
func (s *LevelDBStorage) add(srv Service) error {
	if !srv.validate() {
		return fmt.Errorf("Invalid Service registration")
	}

	s.Lock()
	defer s.Unlock()

	srv.Created = time.Now()
	srv.Updated = srv.Created
	if srv.Ttl >= 0 {
		srv.Expires = srv.Created.Add(time.Duration(srv.Ttl) * time.Second)
	}

	// Add to database
	bytes, err := json.Marshal(&srv)
	if err != nil {
		return err
	}
	err = s.db.Put([]byte(srv.Id), bytes, nil)
	if err != nil {
		return err
	}

	// Create secondary indices
	s.addIndices(&srv)

	return nil
}

func (s *LevelDBStorage) get(id string) (Service, error) {
	// Query from database
	bytes, err := s.db.Get([]byte(id), nil)
	if err == leveldb.ErrNotFound {
		return Service{}, ErrorNotFound
	} else if err != nil {
		return Service{}, err
	}

	var srv Service
	err = json.Unmarshal(bytes, &srv)
	if err != nil {
		return Service{}, err
	}

	return srv, nil
}

func (s *LevelDBStorage) update(id string, srv Service) error {
	s.Lock()
	defer s.Unlock()

	// Get the stored service
	storedSrv, err := s.get(id)
	if err == leveldb.ErrNotFound {
		return ErrorNotFound
	} else if err != nil {
		return err
	}

	// Remove old secondary indices
	s.removeIndices(&storedSrv)

	storedSrv.Type = srv.Type
	storedSrv.Name = srv.Name
	storedSrv.Description = srv.Description
	storedSrv.Ttl = srv.Ttl
	storedSrv.Updated = time.Now()
	if srv.Ttl >= 0 {
		storedSrv.Expires = storedSrv.Updated.Add(time.Duration(srv.Ttl) * time.Second)
	}

	// Store the modified service
	bytes, err := json.Marshal(&storedSrv)
	if err != nil {
		return err
	}
	err = s.db.Put([]byte(id), bytes, nil)
	if err != nil {
		return err
	}

	// Add new secondary indices
	s.addIndices(&storedSrv)

	return nil
}

func (s *LevelDBStorage) delete(id string) error {
	s.Lock()
	defer s.Unlock()

	oldSrv, err := s.get(id)
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
	s.removeIndices(&oldSrv)

	return nil
}

// Utilities

func (s *LevelDBStorage) getMany(page int, perPage int) ([]Service, int, error) {
	s.RLock()
	defer s.RUnlock()

	// Retrieve all keys and slice them
	var keys []string
	s.wg.Add(1)
	iter := s.db.NewIterator(nil, nil)
	for iter.Next() {
		keys = append(keys, string(iter.Key()))
	}
	iter.Release()
	s.wg.Done()
	err := iter.Error()
	if err != nil {
		return nil, 0, err
	}
	slice := catalog.GetPageOfSlice(keys, page, perPage, MaxPerPage)

	if len(slice) == 0 {
		return []Service{}, len(keys), nil
	}

	// Retrieve services that are in the slice
	services := make([]Service, 0, len(slice))
	for _, id := range slice {
		srv, err := s.get(id)
		if err != nil {
			return nil, len(keys), err
		}
		services = append(services, srv)
	}

	return services, len(keys), nil
}

func (s *LevelDBStorage) getCount() int {
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

// Path filtering
// Filter one registration
func (s *LevelDBStorage) pathFilterOne(path string, op string, value string) (Service, error) {
	pathTknz := strings.Split(path, ".")

	// return the first one found
	s.wg.Add(1)
	iter := s.db.NewIterator(nil, nil)
	for iter.Next() {
		var srv Service
		err := json.Unmarshal(iter.Value(), &srv)
		if err != nil {
			return Service{}, err
		}

		matched, err := catalog.MatchObject(srv, pathTknz, op, value)
		if err != nil {
			iter.Release()
			s.wg.Done()
			return Service{}, err
		}
		if matched {
			iter.Release()
			s.wg.Done()
			return srv, nil
		}
	}
	iter.Release()
	s.wg.Done()
	err := iter.Error()
	if err != nil {
		return Service{}, err
	}

	return Service{}, nil
}

// Filter multiple registrations
func (s *LevelDBStorage) pathFilter(path, op, value string, page, perPage int) ([]Service, int, error) {
	var matchedIDs []string
	pathTknz := strings.Split(path, ".")

	s.Lock()
	defer s.Unlock()
	s.wg.Add(1)
	iter := s.db.NewIterator(nil, nil)
	for iter.Next() {
		var srv Service
		err := json.Unmarshal(iter.Value(), &srv)
		if err != nil {
			return []Service{}, 0, err
		}

		matched, err := catalog.MatchObject(srv, pathTknz, op, value)
		if err != nil {
			iter.Release()
			s.wg.Done()
			return []Service{}, 0, err
		}
		if matched {
			matchedIDs = append(matchedIDs, srv.Id)
		}
	}
	iter.Release()
	s.wg.Done()
	err := iter.Error()
	if err != nil {
		return nil, 0, err
	}

	slice := catalog.GetPageOfSlice(matchedIDs, page, perPage, MaxPerPage)
	if len(slice) == 0 {
		return []Service{}, len(matchedIDs), nil
	}

	services := make([]Service, 0, len(slice))
	for _, id := range slice {
		srv, err := s.get(id)
		if err != nil {
			return nil, len(matchedIDs), err
		}
		services = append(services, srv)
	}

	return services, len(matchedIDs), nil
}

// Clean all remote registrations which expire time is larger than the given timestamp
func (s *LevelDBStorage) cleanExpired(timestamp time.Time) {
	s.Lock()

	var expiredList []SortedMap
	for m := range s.expDevice.Iter() {
		if !m.(SortedMap).key.(time.Time).After(timestamp) {
			expiredList = append(expiredList, m.(SortedMap))
		} else {
			// expDevice is sorted by time ascendingly: its elements expire in order
			break
		}
	}

	for _, m := range expiredList {
		// Remove expiry index
		id := s.expDevice.Remove(m).(SortedMap).value.(string)
		logger.Printf("LevelDBStorage.cleanExpired() Registration %v has expired\n", id)

		// Remove the device from db
		err := s.db.Delete([]byte(id), nil)
		if err != nil {
			logger.Println("LevelDBStorage.cleanExpired()", err.Error())
			continue
		}
	}

	s.Unlock()
}

// Creates secondary indices
// WARNING: the caller must obtain the lock before calling
func (s *LevelDBStorage) addIndices(srv *Service) {
	// Add expiry time index
	if srv.Ttl >= 0 {
		s.expDevice.Add(SortedMap{srv.Expires, srv.Id})
	}
}

// Removes secondary indices
// WARNING: the caller must obtain the lock before calling
func (s *LevelDBStorage) removeIndices(srv *Service) {
	// Remove the expiry time index
	if srv.Ttl >= 0 {
		var temp []SortedMap // Keep duplicates in temp and store them back
		for m := range s.expDevice.Iter() {
			id := m.(SortedMap).value.(string)
			if id == srv.Id {
				for { // go through all duplicates
					r := s.expDevice.Remove(m)
					if r == nil {
						break
					}
					if id != srv.Id {
						temp = append(temp, r.(SortedMap))
					}
				}
				break
			}
		}
		for _, r := range temp {
			s.expDevice.Add(r)
		}
	}
}

func (s *LevelDBStorage) close() error {
	s.wg.Wait()
	return s.db.Close()
}
