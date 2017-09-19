// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package service

import (
	"fmt"
	"sync"

	"code.linksmart.eu/sc/service-catalog/utils"

	avl "github.com/ancientlore/go-avltree"
)

// In-memory storage
type MemoryStorage struct {
	sync.RWMutex
	services *avl.Tree
}

func NewMemoryStorage() *MemoryStorage {
	storage := &MemoryStorage{
		services: avl.New(operator, 0),
	}

	return storage
}

func (ms *MemoryStorage) add(s *Service) error {
	ms.Lock()
	defer ms.Unlock()

	_, duplicate := ms.services.Add(*s)
	if duplicate {
		return &ConflictError{fmt.Sprintf("Service id %s is not unique", s.Id)}
	}

	return nil
}

func (ms *MemoryStorage) get(id string) (*Service, error) {
	ms.RLock()
	defer ms.RUnlock()

	s := ms.services.Find(Service{Id: id})
	if s == nil {
		return nil, &NotFoundError{fmt.Sprintf("Service with id %s is not found", id)}
	}
	service := s.(Service)

	return &service, nil
}

func (ms *MemoryStorage) update(id string, s *Service) error {
	ms.Lock()
	defer ms.Unlock()

	r := ms.services.Remove(Service{Id: id})
	if r == nil {
		return &NotFoundError{fmt.Sprintf("Service with id %s is not found", id)}
	}

	ms.services.Add(*s)

	return nil
}

func (ms *MemoryStorage) delete(id string) error {
	ms.Lock()
	defer ms.Unlock()

	r := ms.services.Remove(Service{Id: id})
	if r == nil {
		return &NotFoundError{fmt.Sprintf("Service with id %s is not found", id)}
	}

	return nil
}

func (ms *MemoryStorage) list(page int, perPage int) ([]Service, int, error) {
	ms.RLock()
	defer ms.RUnlock()

	total := ms.services.Len()
	offset, limit, err := utils.GetPagingAttr(total, page, perPage, MaxPerPage)
	if err != nil {
		return nil, 0, &BadRequestError{fmt.Sprintf("Unable to paginate: %s", err)}
	}

	// page/registry is empty
	if limit == 0 {
		return []Service{}, 0, nil
	}

	services := make([]Service, limit)
	data := ms.services.Data()
	for i := 0; i < limit; i++ {
		services[i] = data[i+offset].(Service)
	}

	return services, total, nil
}

func (ms *MemoryStorage) total() (int, error) {
	ms.RLock()
	defer ms.RUnlock()

	return ms.services.Len(), nil
}

func (ms *MemoryStorage) Close() error {
	return nil
}

// Comparison operator for AVL Tree
func operator(a interface{}, b interface{}) int {
	if a.(Service).Id < b.(Service).Id {
		return -1
	} else if a.(Service).Id > b.(Service).Id {
		return 1
	}
	return 0
}
