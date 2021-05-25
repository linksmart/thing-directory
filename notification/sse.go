package notification

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/linksmart/thing-directory/catalog"
	"github.com/linksmart/thing-directory/wot"
)

const (
	QueryParamType    = "type"
	QueryParamFull    = "full"
	HeaderLastEventID = "Last-Event-ID"
)

type SSEAPI struct {
	controller  NotificationController
	contentType string
}

func NewSSEAPI(controller NotificationController, version string) *SSEAPI {
	contentType := "text/event-stream"
	if version != "" {
		contentType += ";version=" + version
	}
	return &SSEAPI{
		controller:  controller,
		contentType: contentType,
	}

}

func (a *SSEAPI) SubscribeEvent(w http.ResponseWriter, req *http.Request) {
	eventTypes, full, err := parseQueryParameters(req)
	if err != nil {
		catalog.ErrorResponse(w, http.StatusBadRequest, err)
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		catalog.ErrorResponse(w, http.StatusInternalServerError, "Streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", a.contentType)

	messageChan := make(chan Event)

	lastEventID := req.Header.Get(HeaderLastEventID)
	a.controller.subscribe(messageChan, eventTypes, full, lastEventID)

	go func() {
		<-req.Context().Done()
		// unsubscribe to events and close the messageChan
		a.controller.unsubscribe(messageChan)
	}()

	for event := range messageChan {
		//data, err := json.MarshalIndent(event.Data, "data: ", "")
		data, err := json.Marshal(event.Data)
		if err != nil {
			log.Printf("error marshaling event %v: %s", event, err)
		}
		fmt.Fprintf(w, "event: %s\n", event.Type)
		fmt.Fprintf(w, "id: %s\n", event.ID)
		fmt.Fprintf(w, "data: %s\n\n", data)

		flusher.Flush()
	}
}

func parseQueryParameters(req *http.Request) ([]wot.EventType, bool, error) {

	full := false
	req.ParseForm()

	// Parse full or partial events
	if strings.EqualFold(req.Form.Get(QueryParamFull), "true") {
		full = true
	}

	// Parse event type to be subscribed to
	queriedTypes := req.Form[QueryParamType]
	if queriedTypes == nil {
		return []wot.EventType{wot.EventTypeCreate, wot.EventTypeUpdate, wot.EventTypeDelete}, full, nil
	}

	var eventTypes []wot.EventType
loopQueriedTypes:
	for _, v := range queriedTypes {
		eventType := wot.EventType(v)
		if !eventType.IsValid() {
			return nil, false, fmt.Errorf("invalid type parameter")
		}
		for _, existing := range eventTypes {
			if existing == eventType {
				continue loopQueriedTypes
			}
		}
		eventTypes = append(eventTypes, eventType)
	}

	return eventTypes, full, nil
}
