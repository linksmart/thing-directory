package notification

import "strconv"

type MemStorage struct {
	events   []Event
	latestID int64
}

func (m MemStorage) add(event Event) error {
	m.events = append(m.events, event)
	return nil
}

func (m MemStorage) getAllAfter(id string) ([]Event, error) {
	for i, v := range m.events {
		if v.Id == id {
			return m.events[i:], nil
		}
	}
	return []Event{}, nil
}

func (m MemStorage) getNewID() (string, error) {
	return strconv.FormatInt(m.latestID, 10), nil
}
