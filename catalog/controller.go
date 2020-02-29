// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"fmt"
	"log"
	"strings"
	"time"

	uuid "github.com/satori/go.uuid"
)

var controllerExpiryCleanupInterval = 60 * time.Second // to be modified in unit tests

type Controller struct {
	storage Storage
}

func NewController(storage Storage) (CatalogController, error) {
	c := Controller{
		storage: storage,
	}

	go c.cleanExpired()

	return &c, nil
}

func (c *Controller) add(td ThingDescription) (string, error) {
	if err := td.validate(); err != nil {
		return "", &BadRequestError{err.Error()}
	}

	// TODO add rc context

	if td.ID == "" {
		// System generated id
		td.ID = c.newURN()
	}
	td.Created = time.Now().UTC()
	td.Modified = td.Created

	err := c.storage.add(&td)
	if err != nil {
		return "", err
	}

	return td.ID, nil
}

func (c *Controller) get(id string) (*ThingDescription, error) {
	return c.storage.get(id)
}

func (c *Controller) update(id string, td ThingDescription) error {
	if err := td.validate(); err != nil {
		return &BadRequestError{err.Error()}
	}

	td.ID = id
	td.Modified = time.Now().UTC()

	err := c.storage.update(id, &td)
	if err != nil {
		return err
	}

	return nil
}

func (c *Controller) delete(id string) error {
	err := c.storage.delete(id)
	if err != nil {
		return err
	}

	return nil
}

func (c *Controller) list(page, perPage int) ([]ThingDescription, int, error) {
	tds, total, err := c.storage.list(page, perPage)
	if err != nil {
		return nil, 0, err
	}

	return tds, total, nil
}

func (c *Controller) filter(path, op, value string, page, perPage int) ([]ThingDescription, int, error) {

	matches := make([]ThingDescription, 0)
	pp := MaxPerPage
	for p := 1; ; p++ {
		slice, t, err := c.storage.list(p, pp)
		if err != nil {
			return nil, 0, err
		}

		for i := range slice {
			matched, err := MatchObject(slice[i], strings.Split(path, "."), op, value)
			if err != nil {
				return nil, 0, err
			}
			if matched {
				matches = append(matches, slice[i])
			}
		}

		if p*pp >= t {
			break
		}
	}
	// Pagination
	offset, limit, err := GetPagingAttr(len(matches), page, perPage, MaxPerPage)
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
	for t := range time.Tick(controllerExpiryCleanupInterval) {
		var expiredServices []*ThingDescription

		for td := range c.storage.iterator() {
			if td.TTL != 0 {
				// remove if expiry is overdue by half-TTL
				if t.After(td.Modified.Add(time.Duration(td.TTL+td.TTL/2) * time.Second)) {
					expiredServices = append(expiredServices, td)
				}
			}
		}

		for i := range expiredServices {
			log.Printf("cleanExpired() Removing expired registration: %s", expiredServices[i].ID)
			err := c.storage.delete(expiredServices[i].ID)
			if err != nil {
				log.Printf("cleanExpired() Error removing expired registration: %s: %s", expiredServices[i].ID, err)
				continue
			}
		}
	}
}

// Stop the controller
func (c *Controller) Stop() {
	//log.Println("Stopped the controller.")
}

// Generate a unique URN
func (c *Controller) newURN() string {
	return fmt.Sprintf("urn:uuid:%s", uuid.NewV4().String())
}
