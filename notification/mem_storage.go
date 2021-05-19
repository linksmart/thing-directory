package notification

import (
	"math/rand"
	"strconv"
)

type MemStorage struct {
	events   []Event
	latestID uint64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{latestID: rand.Uint64()}
}

func (m MemStorage) add(event Event) error {
	m.events = append(m.events, event)
	return nil
}

func (m MemStorage) getAllAfter(ID string) ([]Event, error) {
	for i, v := range m.events {
		if v.ID == ID {
			return m.events[i:], nil
		}
	}
	return []Event{}, nil
}

func (m MemStorage) getNewID() (string, error) {
	m.latestID += 1
	return strconv.FormatUint(m.latestID, 10), nil
}
