// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"runtime/debug"
	"strings"
	"time"

	"github.com/antchfx/jsonquery"
	"github.com/bhmj/jsonslice"
	"github.com/linksmart/service-catalog/v3/utils"
	uuid "github.com/satori/go.uuid"
)

var controllerExpiryCleanupInterval = 10 * time.Second // to be modified in unit tests

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
	id, ok := td[_id].(string)
	if !ok || id == "" {
		// System generated id
		id = c.newURN()
	}
	if err := validateThingDescription(td); err != nil {
		return "", &BadRequestError{err.Error()}
	}

	td[_id] = id
	td[_created] = time.Now().UTC()
	td[_modified] = td[_created]

	err := c.storage.add(id, td)
	if err != nil {
		return "", err
	}

	return id, nil
}

func (c *Controller) get(id string) (ThingDescription, error) {
	return c.storage.get(id)
}

func (c *Controller) update(id string, td ThingDescription) error {
	td[_id] = id
	if err := validateThingDescription(td); err != nil {
		return &BadRequestError{err.Error()}
	}

	oldTD, err := c.storage.get(id)
	if err != nil {
		return err
	}

	td[_created] = oldTD[_created]
	td[_modified] = time.Now().UTC()

	err = c.storage.update(id, td)
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

// Deprecated
func (c *Controller) filter(path, op, value string, page, perPage int) ([]ThingDescription, int, error) {

	matches := make([]ThingDescription, 0)
	pp := MaxPerPage
	for p := 1; ; p++ {
		slice, t, err := c.storage.list(p, pp)
		if err != nil {
			return nil, 0, err
		}

		for i := range slice {
			matched, err := utils.MatchObject(slice[i], strings.Split(path, "."), op, value)
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
	offset, limit, err := utils.GetPagingAttr(len(matches), page, perPage, MaxPerPage)
	if err != nil {
		return nil, 0, &BadRequestError{fmt.Sprintf("Unable to paginate: %s", err)}
	}
	// Return the page
	return matches[offset : offset+limit], len(matches), nil
}

func (c *Controller) listAll() ([]ThingDescription, int, error) {
	var items []ThingDescription
	pp := MaxPerPage
	for p := 1; ; p++ {
		slice, total, err := c.storage.list(p, pp)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, slice...)

		if p*pp >= total {
			return items, total, nil
		}
	}
}

func (c *Controller) filterJSONPath(jsonpath string, page, perPage int) ([]interface{}, int, error) {
	var results []interface{}

	items, total, err := c.listAll()
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return results, 0, nil
	}

	b, err := json.Marshal(items)
	if err != nil {
		return nil, 0, fmt.Errorf("error serializing for jsonpath: %s", err)
	}
	items = nil

	b, err = jsonslice.Get(b, jsonpath)
	if err != nil {
		return nil, 0, fmt.Errorf("error evaluating jsonpath: %s", err)
	}

	err = json.Unmarshal(b, &results)
	if err != nil {
		return nil, 0, fmt.Errorf("error de-serializing for jsonpath: %s", err)
	}
	b = nil

	// Pagination
	offset, limit, err := utils.GetPagingAttr(len(results), page, perPage, MaxPerPage)
	if err != nil {
		return nil, 0, &BadRequestError{fmt.Sprintf("Unable to paginate: %s", err)}
	}
	// Return the page
	return results[offset : offset+limit], len(results), nil
}

func (c *Controller) filterXPath(xpath string, page, perPage int) ([]interface{}, int, error) {
	var results []interface{}

	items, total, err := c.listAll()
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return results, 0, nil
	}

	b, err := json.Marshal(items)
	if err != nil {
		return nil, 0, fmt.Errorf("error serializing entries for xpath parset: %s", err)
	}
	items = nil

	doc, err := jsonquery.Parse(bytes.NewReader(b))
	if err != nil {
		return nil, 0, fmt.Errorf("error parsing JSON for xpath filtering: %s", err)
	}
	b = nil

	nodes, err := jsonquery.QueryAll(doc, xpath)
	if err != nil {
		return nil, 0, fmt.Errorf("error parsing JSON for xpath filtering: %s", err)
	}

	for _, n := range nodes {
		// TODO
		fmt.Println(n)
	}

	// Pagination
	offset, limit, err := utils.GetPagingAttr(len(results), page, perPage, MaxPerPage)
	if err != nil {
		return nil, 0, &BadRequestError{fmt.Sprintf("Unable to paginate: %s", err)}
	}
	// Return the page
	return results[offset : offset+limit], len(results), nil
}

func (c *Controller) total() (int, error) {
	return c.storage.total()
}

func (c *Controller) cleanExpired() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic: %v\n%s\n", r, debug.Stack())
			go c.cleanExpired()
		}
	}()

	for t := range time.Tick(controllerExpiryCleanupInterval) {
		var expiredServices []ThingDescription

		for td := range c.storage.iterator() {
			if td[_ttl] != nil {
				ttl := td[_ttl].(float64)
				if ttl != 0 {
					// remove if expiry is overdue by half-TTL
					modified, err := time.Parse(time.RFC3339, td[_modified].(string))
					if err != nil {
						log.Printf("cleanExpired() error: %s", err)
						continue
					}
					if t.After(modified.Add(time.Duration(1.5*ttl) * time.Second)) {
						expiredServices = append(expiredServices, td)
					}
				}
			}
		}

		for i := range expiredServices {
			id := expiredServices[i][_id].(string)
			log.Printf("cleanExpired() Removing expired registration: %s", id)
			err := c.storage.delete(id)
			if err != nil {
				log.Printf("cleanExpired() Error removing expired registration: %s: %s", id, err)
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
