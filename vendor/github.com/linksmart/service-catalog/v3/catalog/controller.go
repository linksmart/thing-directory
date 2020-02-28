// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/linksmart/service-catalog/v3/utils"
	uuid "github.com/satori/go.uuid"
)

var ControllerExpiryCleanupInterval = 60 * time.Second // to be modified in unit tests

type Controller struct {
	wg sync.WaitGroup
	sync.RWMutex
	storage   Storage
	listeners []Listener
}

func NewController(storage Storage, listeners ...Listener) (*Controller, error) {
	c := Controller{
		storage:   storage,
		listeners: listeners,
	}

	go c.cleanExpired()

	return &c, nil
}

func (c *Controller) add(s Service) (*Service, error) {
	if err := s.validate(); err != nil {
		return nil, &BadRequestError{err.Error()}
	}

	c.Lock()
	defer c.Unlock()

	if s.ID == "" {
		// System generated id
		s.ID = uuid.NewV4().String()
	}
	s.CreatedAt = time.Now().UTC()
	s.UpdatedAt = s.CreatedAt

	s.ExpiresAt = s.CreatedAt.Add(time.Duration(s.TTL) * time.Second)

	err := c.storage.add(&s)
	if err != nil {
		return nil, err
	}

	// notify listeners
	for _, l := range c.listeners {
		go l.added(s)
	}

	return &s, nil
}

func (c *Controller) get(id string) (*Service, error) {
	return c.storage.get(id)
}

func (c *Controller) update(id string, s Service) (*Service, error) {
	if err := s.validate(); err != nil {
		return nil, &BadRequestError{err.Error()}
	}

	c.Lock()
	defer c.Unlock()

	// Get the stored service
	ss, err := c.storage.get(id)
	if err != nil {
		return nil, err
	}

	ss.Title = s.Title
	ss.Description = s.Description
	ss.Type = s.Type
	ss.APIs = s.APIs
	ss.Doc = s.Doc
	ss.Meta = s.Meta
	ss.TTL = s.TTL
	ss.UpdatedAt = time.Now().UTC()
	ss.ExpiresAt = ss.UpdatedAt.Add(time.Duration(ss.TTL) * time.Second)

	err = c.storage.update(id, ss)
	if err != nil {
		return nil, err
	}

	// notify listeners
	for _, l := range c.listeners {
		go l.updated(s)
	}

	return ss, nil
}

func (c *Controller) delete(id string) error {
	c.Lock()
	defer c.Unlock()

	old, err := c.storage.get(id)
	if err != nil {
		return err
	}

	err = c.storage.delete(id)
	if err != nil {
		return err
	}

	// notify listeners
	for _, l := range c.listeners {
		go l.deleted(*old)
	}

	return nil
}

func (c *Controller) list(page, perPage int) ([]Service, int, error) {
	return c.storage.list(page, perPage)
}

func (c *Controller) filter(path, op, value string, page, perPage int) ([]Service, int, error) {
	c.RLock()
	defer c.RUnlock()

	matches := make([]Service, 0)
	pp := MaxPerPage
	for p := 1; ; p++ {
		services, t, err := c.storage.list(p, pp)
		if err != nil {
			return nil, 0, err
		}

		for i := range services {
			matched, err := utils.MatchObject(services[i], strings.Split(path, "."), op, value)
			if err != nil {
				return nil, 0, err
			}
			if matched {
				matches = append(matches, services[i])
			}
		}

		if p*pp >= t {
			break
		}
	}
	// Pagination
	offset, limit, err := utils.GetPagingAttr(len(matches), page, perPage, MaxPerPage)
	if err != nil {
		return nil, 0, &BadRequestError{fmt.Sprintf("Unable to paginate: %s", err)}
	}
	// Return the page
	return matches[offset : offset+limit], len(matches), nil
}

func (c *Controller) total() (int, error) {
	return c.storage.total()
}

func (c *Controller) cleanExpired() {
	clean := func(t time.Time) {
		c.Lock()
		var expiredServices []*Service

		for s := range c.storage.iterator() {
			// remove if expiry is overdue by half-TTL
			if t.After(s.ExpiresAt.Add(time.Duration(s.TTL/2) * time.Second)) {
				expiredServices = append(expiredServices, s)
			}
		}

		for i := range expiredServices {
			logger.Printf("cleanExpired() Removing expired registration: %s", expiredServices[i].ID)
			err := c.storage.delete(expiredServices[i].ID)
			if err != nil {
				logger.Printf("cleanExpired() Error removing expired registration: %s: %s", expiredServices[i].ID, err)
				continue
			}
			// notify listeners
			for li := range c.listeners {
				go c.listeners[li].deleted(*expiredServices[i])
			}
		}
		c.Unlock()
	}

	clean(time.Now())
	for t := range time.Tick(ControllerExpiryCleanupInterval) {
		clean(t)
	}
}

func (c *Controller) AddListener(listener Listener) {
	c.Lock()
	c.listeners = append(c.listeners, listener)
	c.Unlock()
}

func (c *Controller) RemoveListener(listener Listener) {
	c.Lock()
	for i, l := range c.listeners {
		if l == listener {
			//delete the entry and break
			c.listeners = append(c.listeners[:i], c.listeners[i+1:]...)
			break
		}
	}
	c.Unlock()
}

// Stop the controller
func (c *Controller) Stop() error {
	return c.storage.Close()
}
