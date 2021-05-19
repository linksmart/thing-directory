package notification

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/linksmart/thing-directory/common"
)

type SSEAPI struct {
	controller  NotificationController
	contentType string
}

func NewHTTPAPI(controller NotificationController, version string) *SSEAPI {
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
		data, err := json.MarshalIndent(event.Data, "data: ", "    ")
		if err != nil {
			log.Printf("error marshaling event %v: %s", event, err)
		}
		fmt.Fprintf(w, "event: %s\n", event.Type)
		fmt.Fprintf(w, "id: %s\n", event.ID)
		fmt.Fprintf(w, "data: %s\n\n", data)

		flusher.Flush()
	}
}
