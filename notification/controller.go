package notification

import (
	"encoding/json"
	"fmt"
	"log"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/linksmart/thing-directory/catalog"
	"github.com/linksmart/thing-directory/wot"
)

type Controller struct {
	s EventQueue
	// Events are pushed to this channel by the main events-gathering routine
	Notifier chan Event

	// New client connections
	subscribingClients chan subscriber

	// Closed client connections
	unsubscribingClients chan chan Event

	// Client connections registry
	activeClients map[chan Event]subscriber

	// shutdown
	shutdown chan bool
}

type subscriber struct {
	client      chan Event
	eventTypes  []wot.EventType
	diff        bool
	lastEventID string
}

func NewController(s EventQueue) *Controller {
	c := &Controller{
		s:                    s,
		Notifier:             make(chan Event, 1),
		subscribingClients:   make(chan subscriber),
		unsubscribingClients: make(chan chan Event),
		activeClients:        make(map[chan Event]subscriber),
		shutdown:             make(chan bool),
	}
	go c.handler()
	return c
}

func (c *Controller) subscribe(client chan Event, eventTypes []wot.EventType, diff bool, lastEventID string) error {
	s := subscriber{client: client,
		eventTypes:  eventTypes,
		diff:        diff,
		lastEventID: lastEventID,
	}
	c.subscribingClients <- s
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
	err = c.s.addRotate(event)
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
		Type: wot.EventTypeCreate,
		Data: new,
	}

	err := c.storeAndNotify(event)
	return err
}

func (c *Controller) UpdateHandler(old catalog.ThingDescription, new catalog.ThingDescription) error {
	oldJson, err := json.Marshal(old)
	if err != nil {
		return fmt.Errorf("error marshalling old TD")
	}
	newJson, err := json.Marshal(new)
	if err != nil {
		return fmt.Errorf("error marshalling new TD")
	}
	patch, err := jsonpatch.CreateMergePatch(oldJson, newJson)
	if err != nil {
		return fmt.Errorf("error merging new TD")
	}
	var td catalog.ThingDescription
	if err := json.Unmarshal(patch, &td); err != nil {
		return fmt.Errorf("error unmarshalling the patch TD")
	}
	td[wot.KeyThingID] = old[wot.KeyThingID]
	event := Event{
		Type: wot.EventTypeUpdate,
		Data: td,
	}
	err = c.storeAndNotify(event)
	return err
}

func (c *Controller) DeleteHandler(old catalog.ThingDescription) error {
	deleted := catalog.ThingDescription{
		wot.KeyThingID: old[wot.KeyThingID],
	}
	event := Event{
		Type: wot.EventTypeDelete,
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
			c.activeClients[s.client] = s
			log.Printf("New subscription. %d active clients", len(c.activeClients))

			// Send the missed events
			if s.lastEventID != "" {
				missedEvents, err := c.s.getAllAfter(s.lastEventID)
				if err != nil {
					log.Printf("error getting the events after ID %s: %s", s.lastEventID, err)
					continue loop
				}
				for _, event := range missedEvents {
					sendToSubscriber(s, event)
				}
			}
		case clientChan := <-c.unsubscribingClients:
			delete(c.activeClients, clientChan)
			close(clientChan)
			log.Printf("Unsubscribed. %d active clients", len(c.activeClients))
		case event := <-c.Notifier:
			for _, s := range c.activeClients {
				sendToSubscriber(s, event)
			}
		case <-c.shutdown:
			log.Println("Shutting down notification controller")
			break loop
		}
	}

}

func sendToSubscriber(s subscriber, event Event) {
	for _, eventType := range s.eventTypes {
		// Send the notification if the type matches
		if eventType == event.Type {
			toSend := event
			if !s.diff {
				toSend.Data = catalog.ThingDescription{wot.KeyThingID: toSend.Data[wot.KeyThingID]}
			}
			s.client <- toSend
			break
		}
	}
}
