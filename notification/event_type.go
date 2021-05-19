package notification

type EventType string

const (
	createEvent = "create"
	updateEvent = "update"
	deleteEvent = "delete"
)

func (e EventType) IsValid() bool {
	switch e {
	case createEvent, updateEvent, deleteEvent:
		return true
	default:
		return false
	}
}
