package notification

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/linksmart/thing-directory/common"
)

type HTTPAPI struct {
	controller  EventController
	contentType string
}

func NewHTTPAPI(controller EventController, version string) *HTTPAPI {
	contentType := "text/notification-stream"
	if version != "" {
		contentType += ";version=" + version
	}
	return &HTTPAPI{
		controller:  controller,
		contentType: contentType,
	}

}

func (a *HTTPAPI) subscribeEvent(w http.ResponseWriter, req *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		common.ErrorResponse(w, http.StatusInternalServerError, "Streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", a.contentType)

	messageChan := make(chan Event)
	eventTypes := []EventType{createEvent, updateEvent, deleteEvent}
	a.controller.subscribe(messageChan, eventTypes)

	go func() {
		<-req.Context().Done()
		a.controller.unsubscribe(messageChan)
	}()

	for event := range messageChan {
		toSend, err := json.Marshal(event)
		if err != nil {
			log.Printf("error marshaling notification %v: %s", event, err)
		}
		w.Write(toSend)

		flusher.Flush()
	}
}
