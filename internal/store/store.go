// Package store is a tiny append-only event store for the companion app. It
// persists scan/alert events to a JSONL file so the tray UI can show history
// and stats. It is stdlib-only (no database dependency) and safe for concurrent
// use by the local API.
package store

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"
	"time"
)

type Event struct {
	Time      time.Time `json:"time"`
	Kind      string    `json:"kind"` // scan | alert | urlcheck | easm
	MessageID string    `json:"message_id,omitempty"`
	Status    string    `json:"status,omitempty"`
	Score     int       `json:"score,omitempty"`
	Summary   string    `json:"summary,omitempty"`
}

type Store struct {
	mu     sync.Mutex
	path   string
	events []Event
	max    int
}

// Open loads (or creates) a JSONL event store at path. A blank path makes an
// in-memory-only store.
func Open(path string) (*Store, error) {
	s := &Store{path: path, max: 5000}
	if path == "" {
		return s, nil
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		var e Event
		if json.Unmarshal(sc.Bytes(), &e) == nil {
			s.events = append(s.events, e)
		}
	}
	return s, sc.Err()
}

// Append records an event and persists it (best effort).
func (s *Store) Append(e Event) {
	if e.Time.IsZero() {
		e.Time = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, e)
	if len(s.events) > s.max {
		s.events = s.events[len(s.events)-s.max:]
	}
	if s.path == "" {
		return
	}
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	if b, err := json.Marshal(e); err == nil {
		f.Write(append(b, '\n'))
	}
}

// Recent returns the last n events, newest first.
func (s *Store) Recent(n int) []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	if n <= 0 || n > len(s.events) {
		n = len(s.events)
	}
	out := make([]Event, 0, n)
	for i := len(s.events) - 1; i >= 0 && len(out) < n; i-- {
		out = append(out, s.events[i])
	}
	return out
}

type Stats struct {
	Total      int            `json:"total"`
	ByStatus   map[string]int `json:"by_status"`
	ByKind     map[string]int `json:"by_kind"`
	LastUpdate time.Time      `json:"last_update"`
}

// Stats aggregates the current event set.
func (s *Store) Stats() Stats {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := Stats{ByStatus: map[string]int{}, ByKind: map[string]int{}}
	for _, e := range s.events {
		st.Total++
		if e.Status != "" {
			st.ByStatus[e.Status]++
		}
		st.ByKind[e.Kind]++
		if e.Time.After(st.LastUpdate) {
			st.LastUpdate = e.Time
		}
	}
	return st
}
