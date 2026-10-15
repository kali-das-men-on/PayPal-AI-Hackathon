package dispute

import (
	"errors"
	"sort"
	"sync"
	"time"
)

var ErrNotFound = errors.New("dispute not found")

// Store is an in-memory database. HTTP handlers run concurrently (one goroutine
// per request), so every access to the map goes through a mutex.
type Store struct {
	mu       sync.RWMutex // RWMutex: many readers OR one writer
	disputes map[string]*Dispute
	audit    []AuditEntry
}

func NewStore() *Store { return &Store{disputes: make(map[string]*Dispute)} }

func (s *Store) Put(d Dispute) {
	s.mu.Lock()
	defer s.mu.Unlock() // defer runs when the function returns, even on early return
	s.disputes[d.ID] = &d
}

// Get returns a copy, so callers can't modify stored data without Update.
func (s *Store) Get(id string) (Dispute, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.disputes[id]
	if !ok {
		return Dispute{}, ErrNotFound
	}
	return *d, nil
}

func (s *Store) List() []Dispute {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Dispute, 0, len(s.disputes))
	for _, d := range s.disputes {
		out = append(out, *d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DueAt.Before(out[j].DueAt) })
	return out
}

// Update applies fn to the stored dispute while holding the write lock.
func (s *Store) Update(id string, fn func(*Dispute)) (Dispute, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.disputes[id]
	if !ok {
		return Dispute{}, ErrNotFound
	}
	fn(d)
	return *d, nil
}

func (s *Store) Log(id, actor, event, detail string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audit = append(s.audit, AuditEntry{At: time.Now(), DisputeID: id, Actor: actor, Event: event, Detail: detail})
}

// Audit returns entries newest first.
func (s *Store) Audit() []AuditEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]AuditEntry, len(s.audit))
	for i, e := range s.audit {
		out[len(s.audit)-1-i] = e
	}
	return out
}
