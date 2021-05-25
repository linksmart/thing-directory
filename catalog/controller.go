// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"runtime/debug"
	"strconv"
	"time"

	xpath "github.com/antchfx/jsonquery"
	jsonpath "github.com/bhmj/jsonslice"
	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/linksmart/service-catalog/v3/utils"
	"github.com/linksmart/thing-directory/wot"
	uuid "github.com/satori/go.uuid"
)

var controllerExpiryCleanupInterval = 60 * time.Second // to be modified in unit tests

type Controller struct {
	storage   Storage
	listeners eventHandler
}

func NewController(storage Storage) (CatalogController, error) {
	c := Controller{
		storage: storage,
	}

	go c.cleanExpired()

	return &c, nil
}

func (c *Controller) AddSubscriber(listener EventListener) {
	c.listeners = append(c.listeners, listener)
}

func (c *Controller) add(td ThingDescription) (string, error) {
	id, ok := td[wot.KeyThingID].(string)
	if !ok || id == "" {
		// System generated id
		id = c.newURN()
		td[wot.KeyThingID] = id
	}

	results, err := validateThingDescription(td)
	if err != nil {
		return "", err
	}
	if len(results) != 0 {
		return "", &ValidationError{results}
	}

	now := time.Now().UTC()
	tr := ThingRegistration(td)
	td[wot.KeyThingRegistration] = wot.ThingRegistration{
		Created:  &now,
		Modified: &now,
		Expires:  computeExpiry(tr, now),
		TTL:      ThingTTL(tr),
	}

	err = c.storage.add(id, td)
	if err != nil {
		return "", err
	}

	go c.listeners.created(td)

	return id, nil
}

func (c *Controller) get(id string) (ThingDescription, error) {
	td, err := c.storage.get(id)
	if err != nil {
		return nil, err
	}

	//tr := ThingRegistration(td)
	//now := time.Now()
	//tr.Retrieved = &now
	//td[wot.KeyThingRegistration] = tr

	return td, nil
}

func (c *Controller) update(id string, td ThingDescription) error {
	oldTD, err := c.storage.get(id)
	if err != nil {
		return err
	}

	results, err := validateThingDescription(td)
	if err != nil {
		return err
	}
	if len(results) != 0 {
		return &ValidationError{ValidationErrors: results}
	}

	now := time.Now().UTC()
	oldTR := ThingRegistration(oldTD)
	tr := ThingRegistration(td)
	td[wot.KeyThingRegistration] = wot.ThingRegistration{
		Created:  oldTR.Created,
		Modified: &now,
		Expires:  computeExpiry(tr, now),
		TTL:      ThingTTL(tr),
	}

	err = c.storage.update(id, td)
	if err != nil {
		return err
	}

	go c.listeners.updated(oldTD, td)

	return nil
}

// TODO: Improve patch by reducing the number of (de-)serializations
func (c *Controller) patch(id string, td ThingDescription) error {
	oldTD, err := c.storage.get(id)
	if err != nil {
		return err
	}

	// serialize to json for mergepatch input
	oldBytes, err := json.Marshal(oldTD)
	if err != nil {
		return err
	}
	patchBytes, err := json.Marshal(td)
	if err != nil {
		return err
	}
	//fmt.Printf("%s", patchBytes)

	newBytes, err := jsonpatch.MergePatch(oldBytes, patchBytes)
	if err != nil {
		return err
	}
	oldBytes, patchBytes = nil, nil

	td = ThingDescription{}
	err = json.Unmarshal(newBytes, &td)
	if err != nil {
		return err
	}

	results, err := validateThingDescription(td)
	if err != nil {
		return err
	}
	if len(results) != 0 {
		return &ValidationError{results}
	}

	//td[wot.KeyThingRegistrationModified] = time.Now().UTC()
	now := time.Now().UTC()
	oldTR := ThingRegistration(oldTD)
	tr := ThingRegistration(td)
	td[wot.KeyThingRegistration] = wot.ThingRegistration{
		Created:  oldTR.Created,
		Modified: &now,
		Expires:  computeExpiry(tr, now),
		TTL:      ThingTTL(tr),
	}

	err = c.storage.update(id, td)
	if err != nil {
		return err
	}

	go c.listeners.updated(oldTD, td)

	return nil
}

func (c *Controller) delete(id string) error {
	oldTD, err := c.storage.get(id)
	if err != nil {
		return err
	}

	err = c.storage.delete(id)
	if err != nil {
		return err
	}

	go c.listeners.deleted(oldTD)

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

func (c *Controller) listAllBytes() ([]byte, error) {
	return c.storage.listAllBytes()
}

// Deprecated
// Note: filterJSONPath performs several (de-)serializations
// Use filterJSONPathBytes to query bytes directly
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
		return nil, 0, &BadRequestError{S: fmt.Sprintf("error evaluating jsonpath: %s", err)}
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
		return nil, 0, &BadRequestError{S: fmt.Sprintf("unable to paginate: %s", err)}
	}
	// return the requested page
	return results[offset : offset+limit], len(results), nil
}

func (c *Controller) filterJSONPathBytes(query string) ([]byte, error) {
	// query all items
	b, err := c.listAllBytes()
	if err != nil {
		return nil, err
	}

	// filter results with jsonpath
	b, err = jsonpath.Get(b, query)
	if err != nil {
		return nil, &BadRequestError{fmt.Sprintf("error evaluating jsonpath: %s", err)}
	}

	return b, nil
}

// Deprecated
// Note: filterXPath performs several (de-)serializations
// Use filterXPathBytes to query bytes directly
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
		return nil, 0, &BadRequestError{S: fmt.Sprintf("error filtering input with xpath: %s", err)}
	}
	for _, n := range nodes {
		results = append(results, getObjectFromXPathNode(n))
	}

	// paginate
	offset, limit, err := utils.GetPagingAttr(len(results), page, perPage, MaxPerPage)
	if err != nil {
		return nil, 0, &BadRequestError{S: fmt.Sprintf("unable to paginate: %s", err)}
	}
	// return the requested page
	return results[offset : offset+limit], len(results), nil
}

func (c *Controller) filterXPathBytes(path string) ([]byte, error) {
	// query all items
	b, err := c.listAllBytes()
	if err != nil {
		return nil, err
	}

	// parse the json document
	doc, err := xpath.Parse(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("error parsing serialized input for xpath filtering: %s", err)
	}

	// filter with xpath
	nodes, err := xpath.QueryAll(doc, path)
	if err != nil {
		return nil, &BadRequestError{S: fmt.Sprintf("error filtering input with xpath: %s", err)}
	}
	results := make([]interface{}, len(nodes))
	for i := range nodes {
		results[i] = getObjectFromXPathNode(nodes[i])
	}

	// serialize
	b, err = json.Marshal(results)
	if err != nil {
		return nil, fmt.Errorf("error serliazing results of xpath filtering: %s", err)
	}

	return b, nil
}

func (c *Controller) iterateBytes(ctx context.Context) <-chan []byte {
	return c.storage.iterateBytes(ctx)
}

// UTILITY FUNCTIONS

func ThingRegistration(td ThingDescription) *wot.ThingRegistration {
	_, found := td[wot.KeyThingRegistration]
	if found && td[wot.KeyThingRegistration] != nil {
		if trMap, ok := td[wot.KeyThingRegistration].(map[string]interface{}); ok {
			var tr wot.ThingRegistration
			parsedTime := func(t string) *time.Time {
				parsed, err := time.Parse(time.RFC3339, t)
				if err != nil {
					panic(err)
				}
				return &parsed
			}

			if created, ok := trMap[wot.KeyThingRegistrationCreated].(string); ok {
				tr.Created = parsedTime(created)
			}
			if modified, ok := trMap[wot.KeyThingRegistrationModified].(string); ok {
				tr.Modified = parsedTime(modified)
			}
			if expires, ok := trMap[wot.KeyThingRegistrationExpires].(string); ok {
				tr.Expires = parsedTime(expires)
			}
			if ttl, ok := trMap[wot.KeyThingRegistrationTTL].(float64); ok {
				tr.TTL = &ttl
			}

			return &tr
		}
	}
	// not found
	return nil
}

func computeExpiry(tr *wot.ThingRegistration, now time.Time) *time.Time {

	if tr != nil {
		if tr.TTL != nil {
			// calculate expiry as now+ttl
			expires := now.Add(time.Duration(*tr.TTL * 1e9))
			return &expires
		} else if tr.Expires != nil {
			return tr.Expires
		}
	}
	// no expiry
	return nil
}

func ThingExpires(tr *wot.ThingRegistration) *time.Time {
	if tr != nil {
		return tr.Expires
	}
	// no expiry
	return nil
}

func ThingTTL(tr *wot.ThingRegistration) *float64 {
	if tr != nil {
		return tr.TTL
	}
	// no TTL
	return nil
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

	if n.FirstChild != nil && n.FirstChild.Data == "" { // in case of array, there will be no wot.Key
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
			if expires := ThingExpires(ThingRegistration(td)); expires != nil {
				if t.After(*expires) {
					expiredServices = append(expiredServices, td)
				}
			}
		}

		for i := range expiredServices {
			id := expiredServices[i][wot.KeyThingID].(string)
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
