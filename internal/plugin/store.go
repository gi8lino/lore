package plugin

import (
	"bytes"
	"context"
	"sort"
	"sync"
)

// Record is durable installation state. Bundled records contain no archive:
// their distribution bytes always come from the running Lore binary.
type Record struct {
	ID      string
	Source  Source
	Enabled bool
	Package []byte
}
type Store interface {
	ListPlugins(context.Context) ([]Record, error)
	SavePlugin(context.Context, Record) error
	DeletePlugin(context.Context, string) error
}
type ManagerOption func(*Manager)

func WithStore(store Store) ManagerOption { return func(m *Manager) { m.store = store } }

type memoryStore struct {
	mu      sync.Mutex
	records map[string]Record
}

func (s *memoryStore) ListPlugins(context.Context) ([]Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]Record, 0, len(s.records))
	for _, record := range s.records {
		result = append(result, cloneRecord(record))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}
func (s *memoryStore) SavePlugin(_ context.Context, record Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[record.ID] = cloneRecord(record)
	return nil
}
func (s *memoryStore) DeletePlugin(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, id)
	return nil
}
func cloneRecord(record Record) Record { record.Package = bytes.Clone(record.Package); return record }
