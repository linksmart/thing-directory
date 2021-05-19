package notification

import (
	"fmt"
	"log"

	"github.com/linksmart/thing-directory/catalog"
)

type Controller struct {
	s Storage
	// Events are pushed to this channel by the main events-gathering routine
	Notifier chan Event

	// New client connections
	subscribingClients chan subscriber

	// Closed client connections
	unsubscribingClients chan chan Event

	// Client connections registry
	activeClients map[chan Event][]EventType

	// shutdown
	shutdown chan bool
}

type subscriber struct {
	client     chan Event
	eventTypes []EventType
}

func NewController(s Storage) *Controller {
	c := &Controller{
		s:                    s,
		Notifier:             make(chan Event, 1),
		subscribingClients:   make(chan subscriber),
		unsubscribingClients: make(chan chan Event),
		activeClients:        make(map[chan Event][]EventType),
		shutdown:             make(chan bool),
	}
	go c.handler()
	return c
}

func (c *Controller) subscribe(client chan Event, eventType []EventType) error {
	c.subscribingClients <- subscriber{client: client, eventTypes: eventType}
	return nil
}

func (c *Controller) unsubscribe(client chan Event) error {
	c.unsubscribingClients <- client
	return nil
}

func (c *Controller) storeAndNotify(event Event) error {
	var err error
	event.ID, err = c.s.getNewID()
	if err != nil {
		return fmt.Errorf("error generating ID : %v", err)
	}

	// Notify
	c.Notifier <- event

	// Store
	err = c.s.add(event)
	if err != nil {
		return fmt.Errorf("error storing the notification : %v", err)
	}

	return nil
}

func (c *Controller) Stop() {
	c.shutdown <- true
}

func (c *Controller) CreateHandler(new catalog.ThingDescription) error {
	event := Event{
		Type: createEvent,
		Data: new,
	}

	err := c.storeAndNotify(event)
	return err
}

func (c *Controller) UpdateHandler(old catalog.ThingDescription, new catalog.ThingDescription) error {
	event := Event{
		Type: updateEvent,
		Data: new,
	}
	// Todo: store only changes
	err := c.storeAndNotify(event)
	return err
}

func (c *Controller) DeleteHandler(old catalog.ThingDescription) error {
	deleted := catalog.ThingDescription{
		"id": old["id"],
	}
	event := Event{
		Type: deleteEvent,
		Data: deleted,
	}
	err := c.storeAndNotify(event)
	return err
}

func (c *Controller) handler() {
loop:
	for {
		select {
		case s := <-c.subscribingClients:
			c.activeClients[s.client] = s.eventTypes
			log.Printf("New subscription. %d active clients", len(c.activeClients))
		case s := <-c.unsubscribingClients:

			delete(c.activeClients, s)
			log.Printf("Unsubscribed. %d active clients", len(c.activeClients))
		case event := <-c.Notifier:
			for clientMessageChan, eventTypes := range c.activeClients {
				for _, eventType := range eventTypes {
					// Send the notification if the type matches
					if eventType == event.Type {
						clientMessageChan <- event
						break
					}
				}

			}
		case <-c.shutdown:
			log.Println("Shutting down notification controller")
			break loop
		}
	}

}
