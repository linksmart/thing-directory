package notification

import (
	"github.com/linksmart/thing-directory/catalog"
)

type Event struct {
	Id   string                   `json:"id"`
	Type EventType                `json:"notification"`
	Data catalog.ThingDescription `json:"data"`
}

// NotificationController interface
type NotificationController interface {
	subscribe(c chan Event, eventTypes []EventType) error
	unsubscribe(c chan Event) error
	Stop()
	catalog.EventListener
}

// Storage interface
type Storage interface {
	add(event Event) error
	getAllAfter(id string) ([]Event, error)
	getNewID() (string, error)
}
