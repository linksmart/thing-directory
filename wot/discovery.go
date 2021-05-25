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
	// TD event types
	EventTypeCreate = "create"
	EventTypeUpdate = "update"
	EventTypeDelete = "delete"
)

type EnrichedTD struct {
	*ThingDescription
	Registration *ThingRegistration `json:"registration,omitempty"`
}

// ThingRegistration contains the registration information
// alphabetically sorted to match the TD map serialization
type ThingRegistration struct {
	Created   *time.Time `json:"created,omitempty"`
	Expires   *time.Time `json:"expires,omitempty"`
	Modified  *time.Time `json:"modified,omitempty"`
	Retrieved *time.Time `json:"retrieved,omitempty"`
	TTL       *float64   `json:"ttl,omitempty"`
}

type EventType string

func (e EventType) IsValid() bool {
	switch e {
	case EventTypeCreate, EventTypeUpdate, EventTypeDelete:
		return true
	default:
		return false
	}
}
