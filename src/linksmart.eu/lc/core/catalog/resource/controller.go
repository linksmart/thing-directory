package resource

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	avl "github.com/ancientlore/go-avltree"
	"linksmart.eu/lc/core/catalog"
)

type Controller struct {
	wg sync.WaitGroup
	sync.RWMutex
	storage     CatalogStorage
	apiLocation string

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
		rid_did:   avl.New(stringKeys, 0),
		exp_did:   avl.New(timeKeys, avl.AllowDuplicates),
		startTime:   time.Now().UTC().Unix(),
	}

	err := c.initIndices()
	if err != nil {
		return nil, err
	}

	go c.cleanExpired(5 * time.Second)

	return &c, nil
}

// DEVICES

func (c *Controller) add(d *Device) error {
	if err := d.validate(); err != nil {
		return fmt.Errorf("Invalid Device registration: %s", err)
	}

	c.Lock()
	defer c.Unlock()

	if d.Id == "" {
		// System generated id
		d.Id = fmt.Sprintf("urn:is_device:%s", c.newID())
	}
	d.URL = fmt.Sprintf("%s/%s/%s", c.apiLocation, FTypeDevices, d.Id)
	d.Type = ApiDeviceType
	d.Created = time.Now().UTC()
	d.Updated = d.Created
	if d.Ttl == nil {
		d.Expires = nil
	} else {
		expires := d.Created.Add(time.Duration(*d.Ttl) * time.Second)
		d.Expires = &expires
	}

	for i := range d.Resources {

		if d.Resources[i].Id == "" {
			// System generated id
			d.Resources[i].Id = fmt.Sprintf("urn:is_resource:%s", c.newID())
		} else {
			// User-defined id, check for uniqueness
			if match := c.rid_did.Find(Map{key: d.Resources[i].Id}); match != nil {
				return &ConflictError{fmt.Sprintf("Resource id %s is not unique", d.Resources[i].Id)}
			}
		}
		d.Resources[i].URL = fmt.Sprintf("%s/%s/%s", c.apiLocation, FTypeResources, d.Resources[i].Id)
		d.Resources[i].Type = ApiResourceType
		d.Resources[i].Device = d.URL
	}

	sort.Sort(d.Resources)

	err := c.storage.add(d)
	if err != nil {
		return err
	}

	// Add secondary indices
	c.addIndices(d)

	return nil
}

func (c *Controller) get(id string) (*SimpleDevice, error) {
	d, err := c.storage.get(id)
	if err != nil {
		return nil, err
	}

	return c.simplifyDevice(d), nil
}

func (c *Controller) update(id string, d *Device) error {
	if err := d.validate(); err != nil {
		return fmt.Errorf("Invalid Device registration: %s", err)
	}

	c.Lock()
	defer c.Unlock()

	// Get the stored device
	sd, err := c.storage.get(id)
	if err != nil {
		return err
	}

	// Remove existing secondary indices
	c.removeIndices(sd)

	sd.Type = ApiDeviceType
	sd.Name = d.Name
	sd.Description = d.Description
	sd.Meta = d.Meta
	sd.Ttl = d.Ttl
	sd.Updated = time.Now().UTC()
	if sd.Ttl == nil {
		sd.Expires = nil
	} else {
		expires := sd.Updated.Add(time.Duration(*sd.Ttl) * time.Second)
		sd.Expires = &expires
	}

	sd.Resources = d.Resources
	for i := range sd.Resources {
		if sd.Resources[i].Id == "" {
			// System generated id
			sd.Resources[i].Id = fmt.Sprintf("urn:is_resource:%s", c.newID())
		} else {
			// User-defined id
			if match := c.rid_did.Find(Map{key: sd.Resources[i].Id}); match != nil {
				return &ConflictError{fmt.Sprintf("Resource id %s is not unique", sd.Resources[i].Id)}
			}
		}
		sd.Resources[i].URL = fmt.Sprintf("%s/%s/%s", c.apiLocation, FTypeResources, sd.Resources[i].Id)
		sd.Resources[i].Type = ApiResourceType
		sd.Resources[i].Device = sd.URL
	}

	sort.Sort(sd.Resources)

	err = c.storage.update(id, sd)
	if err != nil {
		return err
	}

	// Add new secondary indices
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

	return c.simplifyDevices(devices), total, nil
}

func (c *Controller) filter(path, op, value string, page, perPage int) ([]SimpleDevice, int, error) {
	c.RLock()
	defer c.RUnlock()

	var matches []SimpleDevice
	pp := 100
	for p := 1; ; p++ {
		slice, t, err := c.storage.list(p, pp)
		if err != nil {
			return nil, 0, err
		}

		simplified := c.simplifyDevices(slice)
		for i := range simplified {
			matched, err := catalog.MatchObject(simplified[i], strings.Split(path, "."), op, value)
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
	offset, limit := catalog.GetPagingAttr(len(matches), page, perPage, MaxPerPage)
	// Return the page
	return matches[offset : offset+limit], len(matches), nil
}

func (c *Controller) total() (int, error) {
	return c.storage.total()
}

func (c *Controller) cleanExpired(d time.Duration) {
	t := time.Tick(d)
	for timestamp := range t {
		c.Lock()

		var expiredList []Map
		for m := range c.exp_did.Iter() {
			if !m.(Map).key.(time.Time).After(timestamp.UTC()) {
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
				logger.Println("cleanExpired() Error retrieving device %v:", id, err.Error())
			}
			err = c.storage.delete(id)
			if err != nil {
				logger.Println("cleanExpired() Error removing device %v:", id, err.Error())
			}
			// Remove secondary indices
			c.removeIndices(oldDevice)
		}

		c.Unlock()
	}
}

// RESOURCES

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
	offset, limit := catalog.GetPagingAttr(total, page, perPage, MaxPerPage)

	// Blank page
	if limit == 0 {
		return []Resource{}, total, nil
	}

	// Retrieve resources from devices
	devices := make(map[string]*Device)
	var resources []Resource
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

func (c *Controller) filterResources(path, op, value string, page, perPage int) ([]Resource, int, error) {
	c.RLock()
	defer c.RUnlock()

	// Retrieve resources from devices
	devices := make(map[string]*Device)
	var matches []Resource
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

				matched, err := catalog.MatchObject(d.Resources[i], strings.Split(path, "."), op, value)
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
	offset, limit := catalog.GetPagingAttr(len(matches), page, perPage, MaxPerPage)
	// Return the page
	return matches[offset : offset+limit], len(matches), nil
}

func (c *Controller) totalResources() (int, error) {
	c.RLock()
	defer c.RUnlock()

	return c.rid_did.Len(), nil
}

// UTILITY FUNCTIONS

// Sorting operators
func (s Resources) Len() int           { return len(s) }
func (s Resources) Less(i, j int) bool { return s[i].Id < s[j].Id }
func (s Resources) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// Converts a Device into SimpleDevice
func (c *Controller) simplifyDevice(d *Device) *SimpleDevice {
	resourceIDs := make([]string, len(d.Resources))
	for i := 0; i < len(d.Resources); i++ {
		resourceIDs[i] = fmt.Sprintf("%s/resources/%s", c.apiLocation, d.Resources[i].Id)
	}
	return &SimpleDevice{d, resourceIDs}
}

// Converts []Device into []SimpleDevice
func (c *Controller) simplifyDevices(devices []Device) []SimpleDevice {
	simpleDevices := make([]SimpleDevice, len(devices))
	for i := 0; i < len(devices); i++ {
		simpleDevices[i] = *c.simplifyDevice(&devices[i])
	}
	return simpleDevices
}

// Generate a new unique id
// ID is the timestamp(seconds) of the controller startTime + counter in hex
func (c *Controller) newID() string {
	c.counter++
	return fmt.Sprintf("%x", c.startTime+c.counter)
}

// Initialize secondary indices
func (c *Controller) initIndices() error {
	perPage := 100
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
	if d.Ttl != nil {
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
	if d.Ttl != nil {
		var temp []Map // Keep duplicates in temp and store them back
		for m := range c.exp_did.Iter() {
			id := m.(Map).value.(string)
			if id == d.Id {
				for { // go through all duplicates
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
