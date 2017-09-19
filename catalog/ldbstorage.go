// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// LevelDB storage
type LevelDBStorage struct {
	db *leveldb.DB
	wg sync.WaitGroup
}

func NewLevelDBStorage(dsn string, opts *opt.Options) (CatalogStorage, error) {
	url, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}

	// Open the database file
	db, err := leveldb.OpenFile(url.Path, opts)
	if err != nil {
		return nil, err
	}

	return &LevelDBStorage{db: db}, nil
}

// CRUD
func (s *LevelDBStorage) add(d *Device) error {

	bytes, err := json.Marshal(d)
	if err != nil {
		return err
	}

	_, err = s.db.Get([]byte(d.Id), nil)
	if err == nil {
		return &ConflictError{"Device id is not unique."}
	} else if err != leveldb.ErrNotFound {
		return err
	}

	err = s.db.Put([]byte(d.Id), bytes, nil)
	if err != nil {
		return err
	}

	return nil
}

func (s *LevelDBStorage) get(id string) (*Device, error) {

	bytes, err := s.db.Get([]byte(id), nil)
	if err == leveldb.ErrNotFound {
		return nil, &NotFoundError{fmt.Sprintf("Device with id %s is not found", id)}
	} else if err != nil {
		return nil, err
	}

	var d Device
	err = json.Unmarshal(bytes, &d)
	if err != nil {
		return nil, err
	}

	return &d, nil
}

func (s *LevelDBStorage) update(id string, d *Device) error {

	bytes, err := json.Marshal(d)
	if err != nil {
		return err
	}
	err = s.db.Put([]byte(id), bytes, nil)
	if err == leveldb.ErrNotFound {
		return &NotFoundError{fmt.Sprintf("Device with id %s is not found", id)}
	} else if err != nil {
		return err
	}

	return nil
}

func (s *LevelDBStorage) delete(id string) error {

	err := s.db.Delete([]byte(id), nil)
	if err == leveldb.ErrNotFound {
		return &NotFoundError{fmt.Sprintf("Device with id %s is not found", id)}
	} else if err != nil {
		return err
	}

	return nil
}

func (s *LevelDBStorage) list(page int, perPage int) (Devices, int, error) {

	total, err := s.total()
	if err != nil {
		return nil, 0, err
	}
	offset, limit, err := GetPagingAttr(total, page, perPage, MaxPerPage)
	if err != nil {
		return nil, 0, &BadRequestError{fmt.Sprintf("Unable to paginate: %s", err)}
	}

	// TODO: is there a better way to do this?
	// github.com/syndtr/goleveldb/leveldb/iterator
	devices := make([]Device, limit)
	s.wg.Add(1)
	iter := s.db.NewIterator(nil, nil)
	i := 0
	for iter.Next() {
		var d Device
		err = json.Unmarshal(iter.Value(), &d)
		if err != nil {
			return nil, 0, err
		}

		if i >= offset && i < offset+limit {
			devices[i-offset] = d
		} else if i >= offset+limit {
			break
		}
		i++
	}
	iter.Release()
	s.wg.Done()
	err = iter.Error()
	if err != nil {
		return nil, 0, err
	}

	return devices, total, nil
}

func (s *LevelDBStorage) total() (int, error) {
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
		return 0, err
	}
	return c, nil
}

func (s *LevelDBStorage) Close() error {
	s.wg.Wait()
	return s.db.Close()
}
