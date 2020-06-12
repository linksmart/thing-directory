// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	xpath "github.com/antchfx/jsonquery"
	jsonpath "github.com/bhmj/jsonslice"
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
		td[_id] = id
	}
	if err := validateThingDescription(td); err != nil {
		return "", &BadRequestError{err.Error()}
	}

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
	oldTD, err := c.storage.get(id)
	if err != nil {
		return err
	}

	if err := validateThingDescription(td); err != nil {
		return &BadRequestError{err.Error()}
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

func (c *Controller) filterJSONPath(path string, page, perPage int) ([]interface{}, int, error) {
	var results []interface{}

	// query all items
	items, total, err := c.listAll()
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return results, 0, nil
	}

	// serialize to json
	b, err := json.Marshal(items)
	if err != nil {
		return nil, 0, fmt.Errorf("error serializing for jsonpath: %s", err)
	}
	items = nil

	// filter results with jsonpath
	b, err = jsonpath.Get(b, path)
	if err != nil {
		return nil, 0, fmt.Errorf("error evaluating jsonpath: %s", err)
	}

	// de-serialize the filtered results
	err = json.Unmarshal(b, &results)
	if err != nil {
		return nil, 0, fmt.Errorf("error de-serializing jsonpath evaluation results: %s", err)
	}
	b = nil

	// paginate
	offset, limit, err := utils.GetPagingAttr(len(results), page, perPage, MaxPerPage)
	if err != nil {
		return nil, 0, &BadRequestError{fmt.Sprintf("unable to paginate: %s", err)}
	}
	// return the requested page
	return results[offset : offset+limit], len(results), nil
}

func (c *Controller) filterXPath(path string, page, perPage int) ([]interface{}, int, error) {
	var results []interface{}

	// query all items
	items, total, err := c.listAll()
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return results, 0, nil
	}

	// serialize to json
	b, err := json.Marshal(items)
	if err != nil {
		return nil, 0, fmt.Errorf("error serializing entries for xpath filtering: %s", err)
	}
	items = nil

	// parse the json document
	doc, err := xpath.Parse(bytes.NewReader(b))
	if err != nil {
		return nil, 0, fmt.Errorf("error parsing serialized input for xpath filtering: %s", err)
	}
	b = nil

	// filter with xpath
	nodes, err := xpath.QueryAll(doc, path)
	if err != nil {
		return nil, 0, fmt.Errorf("error filtering input with xpath: %s", err)
	}
	for _, n := range nodes {
		results = append(results, getObjectFromXPathNode(n))
	}

	// paginate
	offset, limit, err := utils.GetPagingAttr(len(results), page, perPage, MaxPerPage)
	if err != nil {
		return nil, 0, &BadRequestError{fmt.Sprintf("unable to paginate: %s", err)}
	}
	// return the requested page
	return results[offset : offset+limit], len(results), nil
}

// basicTypeFromXPathStr is a hack to get the actual data type from xpath.TextNode
// Note: This might cause unexpected behaviour e.g. if user explicitly set string value to "true" or "false"
func basicTypeFromXPathStr(strVal string) interface{} {
	floatVal, err := strconv.ParseFloat(strVal, 64)
	if err == nil {
		return floatVal
	}
	// string value is set to "true" or "false" by the library for boolean values.
	boolVal, err := strconv.ParseBool(strVal) // bit value is set to true or false by the library.
	if err == nil {
		return boolVal
	}
	return strVal
}

// getObjectFromXPathNode gets the concrete object from node by parsing the node recursively.
// Ideally this function needs to be part of the library itself
func getObjectFromXPathNode(n *xpath.Node) interface{} {

	if n.Type == xpath.TextNode { // if top most element is of type textnode, then just return the value
		return basicTypeFromXPathStr(n.Data)
	}

	if n.FirstChild.Data == "" { // in case of array, there will be no key
		retArray := make([]interface{}, 0)
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			retArray = append(retArray, getObjectFromXPathNode(child))
		}
		return retArray
	} else { // normal map
		retMap := make(map[string]interface{})

		for child := n.FirstChild; child != nil; child = child.NextSibling {
			if child.Type != xpath.TextNode {
				retMap[child.Data] = getObjectFromXPathNode(child)
			} else {
				return basicTypeFromXPathStr(child.Data)
			}
		}
		return retMap
	}
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
