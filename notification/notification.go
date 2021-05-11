package notification

import (
	"github.com/linksmart/thing-directory/catalog"
)

type Event struct {
	Id   string                   `json:"id"`
	Type EventType                `json:"notification"`
	Data catalog.ThingDescription `json:"data"`
}

// EventController interface
type EventController interface {
	subscribe(c chan Event, eventTypes []EventType) error
	unsubscribe(c chan Event) error
	catalog.EventListener
}

// Storage interface
type Storage interface {
	add(event Event) error
	getAllAfter(id string) ([]Event, error)
	getNewID() (string, error)
}
