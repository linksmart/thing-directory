package wot

import (
	"time"
)

const (
	// DNS-SD Types
	DNSSDServiceType             = "_wot._tcp"
	DNSSDServiceSubtypeThing     = "_thing"     // _thing._sub._wot._tcp
	DNSSDServiceSubtypeDirectory = "_directory" // _directory._sub._wot._tcp
	// Media Types
	MediaTypeJSONLD = "application/ld+json"
	MediaTypeJSON   = "application/json"
	// TD keys used by directory
	KeyThingID                   = "id"
	KeyThingRegistration         = "registration"
	KeyThingRegistrationCreated  = "created"
	KeyThingRegistrationModified = "modified"
	KeyThingRegistrationExpires  = "expires"
	KeyThingRegistrationTTL      = "ttl"
)

type EnrichedTD struct {
	*ThingDescription
	Registration *ThingRegistration `json:"registration,omitempty"`
}

type ThingRegistration struct {
	Created  *time.Time `json:"created,omitempty"`
	Modified *time.Time `json:"modified,omitempty"`
	Expires  *time.Time `json:"expires,omitempty"`
	TTL      *float64   `json:"ttl,omitempty"`
}
