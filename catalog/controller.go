// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	avl "github.com/ancientlore/go-avltree"
)

type Controller struct {
	wg sync.WaitGroup
	sync.RWMutex
	storage     CatalogStorage
	apiLocation string
	ticker      *time.Ticker

	// startTime and counter for ID generation
	startTime int64
	counter   int64

	// sorted resourceID->deviceID maps
	rid_did *avl.Tree
	// sorted expiryTime->deviceID maps
	exp_did *avl.Tree
}

func NewController(storage CatalogStorage, apiLocation string) (CatalogController, error) {
	c := Controller{
		storage:     storage,
		apiLocation: apiLocation,
		rid_did:     avl.New(stringKeys, 0),
		exp_did:     avl.New(timeKeys, avl.AllowDuplicates), // allows more than one device with the same expiry time
		startTime:   time.Now().UTC().Unix(),
	}

	// Initialize secondary indices (if a persistent storage backend is present)
	err := c.initIndices()
	if err != nil {
		return nil, err
	}

	c.ticker = time.NewTicker(5 * time.Second)
	go c.cleanExpired()

	return &c, nil
}

// DEVICES

func (c *Controller) add(d Device) (string, error) {
	if err := d.validate(); err != nil {
		return "", &BadRequestError{err.Error()}
	}

	c.Lock()
	defer c.Unlock()

	if d.Id == "" {
		// System generated id
		d.Id = c.newDeviceURN()
	}
	d.URL = fmt.Sprintf("%s/%s/%s", c.apiLocation, TypeDevices, d.Id)
	d.Type = ApiDeviceType
	d.Created = time.Now().UTC()
	d.Updated = d.Created
	if d.Ttl == 0 {
		d.Expires = nil
	} else {
		expires := d.Created.Add(time.Duration(d.Ttl) * time.Second)
		d.Expires = &expires
	}

	for i := range d.Resources {

		if d.Resources[i].Id == "" {
			// System generated id
			d.Resources[i].Id = c.newResourceURN()
		} else {
			// User-defined id, check for uniqueness
			if match := c.rid_did.Find(Map{key: d.Resources[i].Id}); match != nil {
				return "", &ConflictError{fmt.Sprintf("Resource id %s is not unique", d.Resources[i].Id)}
			}
		}
		d.Resources[i].URL = fmt.Sprintf("%s/%s/%s", c.apiLocation, TypeResources, d.Resources[i].Id)
		d.Resources[i].Type = ApiResourceType
		d.Resources[i].Device = d.URL
	}
	sort.Sort(d.Resources)

	err := c.storage.add(&d)
	if err != nil {
		return "", err
	}

	// Add secondary indices
	c.addIndices(&d)

	return d.Id, nil
}

func (c *Controller) get(id string) (*SimpleDevice, error) {
	d, err := c.storage.get(id)
	if err != nil {
		return nil, err
	}

	return d.simplify(), nil
}

func (c *Controller) update(id string, d Device) error {
	if err := d.validate(); err != nil {
		return &BadRequestError{err.Error()}
	}

	c.Lock()
	defer c.Unlock()

	// Check uniqueness of resource IDs
	for _, r := range d.Resources {
		// User-defined
		if r.Id != "" {
			if match := c.rid_did.Find(Map{key: r.Id}); match != nil {
				if match.(Map).value.(string) != id {
					return &ConflictError{fmt.Sprintf("Resource id %s is not unique", r.Id)}
				}
			}
		}
	}

	// Get the stored device
	sd, err := c.storage.get(id)
	if err != nil {
		return err
	}

	// Partially deep copy
	var cp Device = *sd
	cp.Resources = make([]Resource, len(sd.Resources))
	copy(cp.Resources, sd.Resources)

	sd.Type = ApiDeviceType
	sd.Name = d.Name
	sd.Description = d.Description
	sd.Meta = d.Meta
	sd.Ttl = d.Ttl
	sd.Updated = time.Now().UTC()
	if sd.Ttl == 0 {
		sd.Expires = nil
	} else {
		expires := sd.Updated.Add(time.Duration(sd.Ttl) * time.Second)
		sd.Expires = &expires
	}
	sd.Resources = d.Resources

	for i := range sd.Resources {
		// System generated resource id
		if sd.Resources[i].Id == "" {
			sd.Resources[i].Id = c.newResourceURN()
		}
		sd.Resources[i].URL = fmt.Sprintf("%s/%s/%s", c.apiLocation, TypeResources, sd.Resources[i].Id)
		sd.Resources[i].Type = ApiResourceType
		sd.Resources[i].Device = sd.URL
	}
	sort.Sort(sd.Resources)

	err = c.storage.update(id, sd)
	if err != nil {
		return err
	}

	// Update secondary indices
	c.removeIndices(&cp)
	c.addIndices(sd)

	return nil
}

func (c *Controller) delete(id string) error {
	c.Lock()
	defer c.Unlock()

	oldDevice, err := c.storage.get(id)
	if err != nil {
		return err
	}

	err = c.storage.delete(id)
	if err != nil {
		return err
	}

	// Remove secondary indices
	c.removeIndices(oldDevice)

	return nil
}

func (c *Controller) list(page, perPage int) ([]SimpleDevice, int, error) {
	devices, total, err := c.storage.list(page, perPage)
	if err != nil {
		return nil, 0, err
	}

	return devices.simplify(), total, nil
}

func (c *Controller) filter(path, op, value string, page, perPage int) ([]SimpleDevice, int, error) {
	c.RLock()
	defer c.RUnlock()

	matches := make([]SimpleDevice, 0)
	pp := MaxPerPage
	for p := 1; ; p++ {
		slice, t, err := c.storage.list(p, pp)
		if err != nil {
			return nil, 0, err
		}

		simplified := slice.simplify()
		for i := range simplified {
			matched, err := MatchObject(simplified[i], strings.Split(path, "."), op, value)
			if err != nil {
				return nil, 0, err
			}
			if matched {
				matches = append(matches, simplified[i])
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
	for t := range c.ticker.C {
		c.Lock()

		var expiredList []Map
		for m := range c.exp_did.Iter() {
			if !m.(Map).key.(time.Time).After(t.UTC()) {
				expiredList = append(expiredList, m.(Map))
			} else {
				// exp_did is sorted by time ascendingly: its elements expire in order
				break
			}
		}

		for _, m := range expiredList {
			id := m.value.(string)
			logger.Printf("cleanExpired() Registration %v has expired\n", id)

			oldDevice, err := c.storage.get(id)
			if err != nil {
				logger.Printf("cleanExpired() Error retrieving device %v: %v\n", id, err.Error())
				break
			}

			err = c.storage.delete(id)
			if err != nil {
				logger.Printf("cleanExpired() Error removing device %v: %v\n", id, err.Error())
				break
			}
			// Remove secondary indices
			c.removeIndices(oldDevice)
		}

		c.Unlock()
	}
}

// RESOURCES

func (c *Controller) getResource(id string) (*Resource, error) {
	c.RLock()
	defer c.RUnlock()

	res := c.rid_did.Find(Map{key: id})
	if res == nil {
		return nil, &NotFoundError{"Resource not found"}
	}
	deviceID := res.(Map).value.(string)

	device, err := c.storage.get(deviceID)
	if err != nil {
		return nil, err
	}

	for _, res := range device.Resources {
		if res.Id == id {
			return &res, nil
		}
	}

	return nil, &NotFoundError{"Parent device not found"} // should never happen

}

func (c *Controller) listResources(page, perPage int) ([]Resource, int, error) {
	c.RLock()
	defer c.RUnlock()

	total := c.rid_did.Len()

	// Retrieve resourceID->deviceID (s) from the tree
	resourceIDs := make([]string, total)
	deviceIDs := make([]string, total)
	for i, x := range c.rid_did.Data() {
		resourceIDs[i] = x.(Map).key.(string)
		deviceIDs[i] = x.(Map).value.(string)
	}
	// Pagination
	offset, limit, err := GetPagingAttr(total, page, perPage, MaxPerPage)
	if err != nil {
		return nil, 0, &BadRequestError{fmt.Sprintf("Unable to paginate: %s", err)}
	}

	// Blank page
	if limit == 0 {
		return []Resource{}, total, nil
	}

	// Retrieve resources from devices
	devices := make(map[string]*Device)
	resources := make([]Resource, 0)
	for i := offset; i < offset+limit; i++ {
		did := deviceIDs[i]
		rid := resourceIDs[i]

		var err error
		d, exists := devices[did]
		if !exists {
			d, err = c.storage.get(did)
			if err != nil {
				return nil, total, err
			}
			devices[did] = d
		}

		for r := range d.Resources {
			if d.Resources[r].Id == rid {
				resources = append(resources, d.Resources[r])
				break
			}
		}
	}
	return resources, total, nil
}

func (c *Controller) filterResources(path, op, value string, page, perPage int) ([]Resource, int, error) {
	c.RLock()
	defer c.RUnlock()

	// Retrieve resources from devices
	devices := make(map[string]*Device)
	matches := make([]Resource, 0)
	for x := range c.rid_did.Iter() {
		resourceID := x.(Map).key.(string)
		deviceID := x.(Map).value.(string)

		var err error
		d, exists := devices[deviceID]
		if !exists {
			d, err = c.storage.get(deviceID)
			if err != nil {
				return nil, 0, err
			}
			devices[deviceID] = d
		}

		for i := range d.Resources {
			if d.Resources[i].Id == resourceID {

				matched, err := MatchObject(d.Resources[i], strings.Split(path, "."), op, value)
				if err != nil {
					return nil, 0, err
				}
				if matched {
					matches = append(matches, d.Resources[i])
				}
			}
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

func (c *Controller) totalResources() (int, error) {
	c.RLock()
	defer c.RUnlock()

	return c.rid_did.Len(), nil
}

// Stop the controller
func (c *Controller) Stop() error {
	c.ticker.Stop()
	return c.storage.Close()
}

// UTILITY FUNCTIONS

// Sorting operators
func (s Resources) Len() int           { return len(s) }
func (s Resources) Less(i, j int) bool { return s[i].Id < s[j].Id }
func (s Resources) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// Generate a new unique urn for device
// Format: urn:ls_device:id, where id is the timestamp(s) of the controller startTime+counter in hex
// WARNING: the caller must obtain the lock before calling
func (c *Controller) newDeviceURN() string {
	c.counter++
	return fmt.Sprintf("urn:ls_device:%x", c.startTime+c.counter)
}

// Generate a new unique urn for resource
// Format: urn:ls_resource:id, where id is the timestamp(s) of the controller startTime+counter in hex
// WARNING: the caller must obtain the lock before calling
func (c *Controller) newResourceURN() string {
	c.counter++
	return fmt.Sprintf("urn:ls_resource:%x", c.startTime+c.counter)
}

// Initialize secondary indices (from a persistent storage backend)
func (c *Controller) initIndices() error {
	perPage := MaxPerPage
	for page := 1; ; page++ {
		devices, total, err := c.storage.list(page, perPage)
		if err != nil {
			return err
		}

		for i, _ := range devices {
			c.addIndices(&devices[i])
		}

		if page*perPage >= total {
			break
		}
	}
	return nil
}

// Creates secondary indices
// WARNING: the caller must obtain the lock before calling
func (c *Controller) addIndices(d *Device) {
	for _, r := range d.Resources {
		c.rid_did.Add(Map{r.Id, d.Id})
	}

	// Add expiry time index
	if d.Ttl != 0 {
		c.exp_did.Add(Map{*d.Expires, d.Id})
	}
}

// Removes secondary indices
// WARNING: the caller must obtain the lock before calling
func (c *Controller) removeIndices(d *Device) {
	// Remove resource indices
	for _, r := range d.Resources {
		c.rid_did.Remove(Map{key: r.Id})
	}

	// Remove the expiry time index
	// INFO:
	// More than one device can have the same expiry time (i.e. map's key)
	//	which leads to non-unique keys in the maps.
	// This code removes keys with that expiry time (keeping them in a temp) until the
	// 	desired target is reached. It then adds the items in the temp back to the tree.
	if d.Ttl != 0 {
		var temp []Map
		for m := range c.exp_did.Iter() {
			id := m.(Map).value.(string)
			if id == d.Id {
				for { // go through all duplicates (same expiry times)
					r := c.exp_did.Remove(m)
					if r == nil {
						break
					}
					if id != d.Id {
						temp = append(temp, r.(Map))
					}
				}
				break
			}
		}
		for _, r := range temp {
			c.exp_did.Add(r)
		}
	}
}

// AVL Tree: sorted nodes according to keys
//
// A node of the AVL Tree (go-avltree)
type Map struct {
	key   interface{}
	value interface{}
}

// Operator for string-type key
func stringKeys(a interface{}, b interface{}) int {
	if a.(Map).key.(string) < b.(Map).key.(string) {
		return -1
	} else if a.(Map).key.(string) > b.(Map).key.(string) {
		return 1
	}
	return 0
}

// Operator for Time-type key
func timeKeys(a interface{}, b interface{}) int {
	if a.(Map).key.(time.Time).Before(b.(Map).key.(time.Time)) {
		return -1
	} else if a.(Map).key.(time.Time).After(b.(Map).key.(time.Time)) {
		return 1
	}
	return 0
}
