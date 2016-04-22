package resource

import (
	"sync"

	"time"

	"fmt"
	"sort"

	avl "github.com/ancientlore/go-avltree"
	"linksmart.eu/lc/core/catalog"
)

type Controller struct {
	wg sync.WaitGroup
	sync.RWMutex
	storage     CatalogStorage
	apiLocation string

	// sorted resourceID->deviceID map
	resDevice *avl.Tree
	// sorted expiryTime->deviceID map
	expDevice *avl.Tree
}

func NewController(storage CatalogStorage, apiLocation string) (CatalogController, error) {
	c := Controller{
		storage:     storage,
		apiLocation: apiLocation,
		resDevice:   avl.New(stringKeys, 0),
		expDevice:   avl.New(timeKeys, avl.AllowDuplicates),
	}

	err := c.initIndices()
	if err != nil {
		return nil, err
	}

	// schedule cleaner
	//t := time.Tick(time.Duration(5) * time.Second)
	//go func() {
	//	for now := range t {
	//		storage.cleanExpired(now)
	//	}
	//}()

	return &c, nil
}

func (c *Controller) listDevices(page, perPage int) ([]SimpleDevice, int, error) {
	devices, total, err := c.storage.list(page, perPage)
	if err != nil {
		return nil, 0, err
	}

	return c.simplifyDevices(devices), total, nil
}

func (c *Controller) addDevice(d *Device) error {
	if err := d.validate(); err != nil {
		return fmt.Errorf("Invalid Device registration: %s", err)
	}

	// System generated id
	c.Lock()
	d.Id = fmt.Sprintf("urn:is_device:%x", time.Now().UnixNano())
	c.Unlock()

	d.URL = fmt.Sprintf("%s/devices/%s", c.apiLocation, d.Id)
	d.Created = time.Now()
	d.Updated = d.Created
	if d.Ttl >= 0 {
		expires := d.Created.Add(time.Duration(d.Ttl) * time.Second)
		d.Expires = &expires
	} else {
		d.Expires = nil
	}

	for i := range d.Resources {
		// System generated id
		c.Lock()
		d.Resources[i].Id = fmt.Sprintf("urn:is_resource:%x", time.Now().UnixNano())
		c.Unlock()

		d.Resources[i].URL = fmt.Sprintf("%s/resources/%s", c.apiLocation, d.Resources[i].Id)
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

func (c *Controller) getDevice(id string) (*SimpleDevice, error) {
	d, err := c.storage.get(id)
	if err != nil {
		return nil, err
	}

	return c.simplifyDevice(d), nil
}

func (c *Controller) updateDevice(id string, d *Device) error {
	if err := d.validate(); err != nil {
		return fmt.Errorf("Invalid Device registration: %s", err)
	}

	// Get the stored device
	sd, err := c.storage.get(id)
	if err != nil {
		switch err.(type) {
		case *NotFoundError:
			// Create a device with user-defined id
			sd.Id = d.Id
			sd.URL = fmt.Sprintf("%s/devices/%s", c.apiLocation, d.Id)
		default:
			return err
		}
	} else {
		// Remove old secondary indices
		c.removeIndices(sd)
	}

	sd.Type = d.Type
	sd.Name = d.Name
	sd.Description = d.Description
	sd.Meta = d.Meta
	sd.Resources = d.Resources
	sd.Ttl = d.Ttl
	sd.Updated = time.Now()
	if sd.Ttl >= 0 {
		expires := sd.Updated.Add(time.Duration(sd.Ttl) * time.Second)
		sd.Expires = &expires
	} else {
		sd.Expires = nil
	}

	for i := range sd.Resources {
		if sd.Resources[i].Id == "" {
			// System generated id
			c.Lock()
			sd.Resources[i].Id = fmt.Sprintf("urn:is_resource:%x", time.Now().UnixNano())
			c.Unlock()
		} else {
			// User-defined id
			if match := c.resDevice.Find(SortedMap{key: d.Resources[i].Id}); match != nil {
				return &NotUniqueError{fmt.Sprintf("Resource id %s is not unique", d.Resources[i].Id)}
			}
			sd.Resources[i].Id = d.Resources[i].Id
		}
		d.Resources[i].URL = fmt.Sprintf("%s/resources/%s", c.apiLocation, d.Resources[i].Id)
		sd.Resources[i].Device = d.URL
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

func (c *Controller) deleteDevice(id string) error {
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

func (c *Controller) filterDevices(path, op, value string, page, perPage int) ([]SimpleDevice, int, error) {
	return nil, 0, nil
}

func (c *Controller) totalDevices() (int, error) {
	c.RLock()
	defer c.RUnlock()

	return c.storage.total()
}

func (c *Controller) deviceCleaner() {

}

func (c *Controller) listResources(page, perPage int) ([]Resource, int, error) {

	total, _ := c.totalResources() // already mutex locked

	c.RLock()
	defer c.RUnlock()

	// Slice resources map
	keys := make([]string, total)
	for i, x := range c.resDevice.Data() {
		keys[i] = x.(SortedMap).key.(string)
	}
	slice := catalog.GetPageOfSlice(keys, page, perPage, MaxPerPage)

	if len(slice) == 0 {
		return nil, total, nil
	}

	// Retrieve devices that are in the slice
	sliceMap := make(map[string]bool)
	for _, x := range slice {
		sliceMap[x] = true
	}
	devices := make(map[string]*Device)
	var resources []Resource
	for _, k := range slice {
		deviceID := c.resDevice.Find(SortedMap{key: k}).(SortedMap).value.(string)

		d, exists := devices[deviceID]
		if !exists {
			d, err := c.storage.get(deviceID)
			if err != nil {
				return nil, total, err
			}
			devices[deviceID] = d
		}

		ri := sort.Search(len(d.Resources), func(i int) bool { return d.Resources[i].Id == k })
		resources = append(resources, d.Resources[ri])
	}

	return resources, total, nil
}

func (c *Controller) getResource(id string) (*Resource, error) {

	res := c.resDevice.Find(SortedMap{key: id})
	if res == nil {
		return nil, ErrorNotFound
	}
	deviceID := res.(SortedMap).value.(string)

	device, err := c.storage.get(deviceID)
	if err != nil {
		return nil, err
	}

	for _, res := range device.Resources {
		if res.Id == id {
			return &res, nil
		}
	}

	return nil, ErrorNotFound

}

func (c *Controller) filterResources(path, op, value string, page, perPage int) ([]Resource, int, error) {
	return nil, 0, nil
}

func (c *Controller) totalResources() (int, error) {
	c.RLock()
	l := c.resDevice.Len()
	c.RUnlock()
	return l, nil
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
	c.Lock()
	defer c.Unlock()

	for _, r := range d.Resources {
		c.resDevice.Add(SortedMap{r.Id, d.Id})
	}

	// Add expiry time index
	if d.Ttl >= 0 {
		c.expDevice.Add(SortedMap{*d.Expires, d.Id})
	}
}

// Removes secondary indices
// WARNING: the caller must obtain the lock before calling
func (c *Controller) removeIndices(d *Device) {
	c.Lock()
	defer c.Unlock()

	// Remove resource indices
	for _, r := range d.Resources {
		c.resDevice.Remove(SortedMap{key: r.Id})
	}

	// Remove the expiry time index
	if d.Ttl >= 0 {
		var temp []SortedMap // Keep duplicates in temp and store them back
		for m := range c.expDevice.Iter() {
			id := m.(SortedMap).value.(string)
			if id == d.Id {
				for { // go through all duplicates
					r := c.expDevice.Remove(m)
					if r == nil {
						break
					}
					if id != d.Id {
						temp = append(temp, r.(SortedMap))
					}
				}
				break
			}
		}
		for _, r := range temp {
			c.expDevice.Add(r)
		}
	}
}

//// Clean all remote registrations which expire time is larger than the given timestamp
//func (s *LevelDBStorage) cleanExpired(timestamp time.Time) {
//	s.Lock()
//
//	var expiredList []SortedMap
//	for m := range s.expDevice.Iter() {
//		if !m.(SortedMap).key.(time.Time).After(timestamp) {
//			expiredList = append(expiredList, m.(SortedMap))
//		} else {
//			// expDevice is sorted by time ascendingly: its elements expire in order
//			break
//		}
//	}
//
//	for _, m := range expiredList {
//		// Remove expiry index
//		id := s.expDevice.Remove(m).(SortedMap).value.(string)
//		logger.Printf("LevelDBStorage.cleanExpired() Registration %v has expired\n", id)
//
//		oldDevice, err := s.get(id)
//		if err != nil {
//			logger.Println("LevelDBStorage.cleanExpired()", err.Error())
//			continue
//		}
//
//		// Remove the device from db
//		err = s.db.Delete([]byte(id), nil)
//		if err != nil {
//			logger.Println("LevelDBStorage.cleanExpired()", err.Error())
//			continue
//		}
//
//		// Remove resource indices
//		for _, r := range oldDevice.Resources {
//			s.resDevice.Remove(SortedMap{key: r.Id})
//		}
//	}
//
//	s.Unlock()
//}
