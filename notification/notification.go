package notification

import (
	"github.com/linksmart/thing-directory/catalog"
)

type Event struct {
	ID   string                   `json:"id"`
	Type EventType                `json:"event"`
	Data catalog.ThingDescription `json:"data"`
}

// NotificationController interface
type NotificationController interface {
	subscribe(client chan Event, eventTypes []EventType, full bool, lastEventID string) error
	unsubscribe(client chan Event) error
	Stop()
	catalog.EventListener
}

// Storage interface
type Storage interface {
	add(event Event) error
	getAllAfter(id string) ([]Event, error)
	getNewID() (string, error)
	Close()
}
