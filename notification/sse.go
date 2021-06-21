package notification

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/linksmart/thing-directory/catalog"
	"github.com/linksmart/thing-directory/wot"
)

const (
	QueryParamType    = "type"
	QueryParamFull    = "diff"
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
	diff, err := parseQueryParameters(req)
	if err != nil {
		catalog.ErrorResponse(w, http.StatusBadRequest, err)
		return
	}
	eventTypes, err := parsePath(req)
	if err != nil {
		catalog.ErrorResponse(w, http.StatusBadRequest, err)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		catalog.ErrorResponse(w, http.StatusInternalServerError, "Streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", a.contentType)

	messageChan := make(chan Event)

	lastEventID := req.Header.Get(HeaderLastEventID)
	a.controller.subscribe(messageChan, eventTypes, diff, lastEventID)

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

func parseQueryParameters(req *http.Request) (bool, error) {
	diff := false
	req.ParseForm()
	// Parse diff or just ID
	if strings.EqualFold(req.Form.Get(QueryParamFull), "true") {
		diff = true
	}
	return diff, nil
}

func parsePath(req *http.Request) ([]wot.EventType, error) {
	// Parse event type to be subscribed to
	params := mux.Vars(req)
	event := params[QueryParamType]
	if event == "" {
		return []wot.EventType{wot.EventTypeCreate, wot.EventTypeUpdate, wot.EventTypeDelete}, nil
	}

	eventType := wot.EventType(event)
	if !eventType.IsValid() {
		return nil, fmt.Errorf("invalid type in path")
	}

	return []wot.EventType{eventType}, nil

}
