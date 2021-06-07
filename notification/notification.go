package notification

import (
	"github.com/linksmart/thing-directory/catalog"
	"github.com/linksmart/thing-directory/wot"
)

type Event struct {
	ID   string                   `json:"id"`
	Type wot.EventType            `json:"event"`
	Data catalog.ThingDescription `json:"data"`
}

// NotificationController interface
type NotificationController interface {
	// subscribe to the events. the caller will get events through the channel 'client' starting from 'lastEventID'
	subscribe(client chan Event, eventTypes []wot.EventType, diff bool, lastEventID string) error

	// unsubscribe and close the channel 'client'
	unsubscribe(client chan Event) error

	// Stop the controller
	Stop()

	catalog.EventListener
}

// EventQueue interface
type EventQueue interface {
	//addRotate adds new and delete the old event if the event queue is full
	addRotate(event Event) error

	// getAllAfter gets the events after the event ID
	getAllAfter(id string) ([]Event, error)

	// getNewID creates a new ID for the event
	getNewID() (string, error)

	// Close all the resources acquired by the queue implementation
	Close()
}
