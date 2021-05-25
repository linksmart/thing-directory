package notification

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// LevelDB storage
type LevelDBEventQueue struct {
	db       *leveldb.DB
	wg       sync.WaitGroup
	latestID uint64
	capacity uint64
}

func NewLevelDBEventQueue(dsn string, opts *opt.Options, capacity uint64) (EventQueue, error) {
	url, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}

	// Open the database file
	db, err := leveldb.OpenFile(url.Path, opts)
	if err != nil {
		return nil, err
	}

	ldbEventQueue := &LevelDBEventQueue{db: db, capacity: capacity}
	ldbEventQueue.latestID, err = ldbEventQueue.fetchLatestID()
	if err != nil {
		return nil, fmt.Errorf("error fetching the latest ID from storage: %w", err)
	}
	return ldbEventQueue, nil
}

func (s *LevelDBEventQueue) addRotate(event Event) error {
	s.wg.Add(1)
	defer s.wg.Done()

	// add new data
	bytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("error marshalling event: %w", err)
	}
	uintID, err := strconv.ParseUint(event.ID, 16, 64)
	if err != nil {
		return fmt.Errorf("error parsing event ID: %w", err)
	}
	batch := new(leveldb.Batch)

	batch.Put(uint64ToByte(uintID), bytes)

	// cleanup the older data
	if s.latestID > s.capacity {
		cleanBefore := s.latestID - s.capacity + 1 // adding 1 as Range is  is not inclusive the limit.
		iter := s.db.NewIterator(&util.Range{Limit: uint64ToByte(cleanBefore)}, nil)
		for iter.Next() {
			// log.Println("deleting older entry: ", byteToUint64(iter.Key()))
			batch.Delete(iter.Key())
		}
		iter.Release()
		err = iter.Error()
		if err != nil {
			return err
		}
	}
	err = s.db.Write(batch, nil)
	if err != nil {
		return fmt.Errorf("error cleaning up: %w", err)
	}
	return nil
}

func (s *LevelDBEventQueue) getAllAfter(id string) ([]Event, error) {
	intID, err := strconv.ParseUint(id, 16, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing latest ID: %w", err)
	}

	// start from the last missing event.
	// If the leveldb does not have the requested ID,
	// then the iterator starts with oldest available entry
	iter := s.db.NewIterator(&util.Range{Start: uint64ToByte(intID + 1)}, nil)
	var events []Event
	for iter.Next() {
		var event Event
		err = json.Unmarshal(iter.Value(), &event)
		if err != nil {
			iter.Release()
			return nil, fmt.Errorf("error unmarshalling event: %w", err)
		}
		events = append(events, event)
	}
	iter.Release()
	err = iter.Error()
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (s *LevelDBEventQueue) getNewID() (string, error) {
	s.latestID += 1
	return strconv.FormatUint(s.latestID, 16), nil
}

func (s *LevelDBEventQueue) Close() {
	s.wg.Wait()
	err := s.db.Close()
	if err != nil {
		log.Printf("Error closing SSE storage: %s", err)
	}
	if flag.Lookup("test.v") == nil {
		log.Println("Closed SSE leveldb.")
	}
}

func (s *LevelDBEventQueue) fetchLatestID() (uint64, error) {
	var latestID uint64
	s.wg.Add(1)
	defer s.wg.Done()
	iter := s.db.NewIterator(nil, nil)
	exists := iter.Last()
	if exists {
		latestID = byteToUint64(iter.Key())
	} else {
		// Start from 0
		latestID = 0
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return 0, err
	}
	return latestID, nil
}

//byte to unint64 conversion functions and vice versa
func byteToUint64(input []byte) uint64 {
	return binary.BigEndian.Uint64(input)
}
func uint64ToByte(input uint64) []byte {
	output := make([]byte, 8)
	binary.BigEndian.PutUint64(output, input)
	return output
}
