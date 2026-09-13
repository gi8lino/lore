package plugin

import (
	"fmt"
	"github.com/gi8lino/lore/internal/pluginpackage"
	"slices"
	"sync"
)

// A lifetime belongs to one contribution version. Snapshots acquired by a render
// retain it, so retirement cannot close a reactor while that render still uses it.
type lifetime struct {
	mu      sync.Mutex
	refs    int
	retired bool
	done    chan struct{}
}

func newLifetime() *lifetime { return &lifetime{done: make(chan struct{})} }
func (l *lifetime) acquire() { l.mu.Lock(); defer l.mu.Unlock(); l.refs++ }
func (l *lifetime) release() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.refs--
	if l.retired && l.refs == 0 {
		close(l.done)
	}
}
func (l *lifetime) retire() <-chan struct{} {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.retired = true
	if l.refs == 0 {
		close(l.done)
	}
	return l.done
}

// Acquire pins a consistent contribution set for one complete render. Release
// must be called once, including on error. No registry lock covers plugin calls.
func (r *Registry) Acquire() (Snapshot, func()) {
	r.mu.RLock()
	snapshot := Snapshot{Entries: make([]Entry, len(r.entries))}
	lives := make([]*lifetime, 0, len(r.entries))
	for index, entry := range r.entries {
		entry.lifetime.acquire()
		lives = append(lives, entry.lifetime)
		snapshot.Entries[index] = cloneEntry(entry)
	}
	r.mu.RUnlock()
	var once sync.Once
	return snapshot, func() {
		once.Do(func() {
			for _, life := range lives {
				life.release()
			}
		})
	}
}

// transition validates a candidate before committing persistence or publication.
// Replacement keeps contribution order and is never observable as remove/add.
func (r *Registry) transition(id string, replacement *Entry, replace bool, commit func() error) (*lifetime, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	index := slices.IndexFunc(r.entries, func(e Entry) bool { return e.Descriptor.ID == id })
	if replacement != nil && (index >= 0) != replace {
		if index >= 0 {
			return nil, fmt.Errorf("plugin %s is already registered", id)
		}
		return nil, fmt.Errorf("plugin %s is no longer registered", id)
	}
	candidate := &Registry{entries: slices.Clone(r.entries)}
	if replacement == nil {
		if err := candidate.Unregister(id); err != nil {
			return nil, err
		}
	} else {
		if index >= 0 {
			candidate.entries = slices.Delete(candidate.entries, index, index+1)
		}
		if err := candidate.Register(replacement.Descriptor, replacement.Contributions); err != nil {
			return nil, err
		}
		if index >= 0 {
			added := candidate.entries[len(candidate.entries)-1]
			candidate.entries = candidate.entries[:len(candidate.entries)-1]
			candidate.entries = slices.Insert(candidate.entries, index, added)
		}
	}
	catalog := make(map[string]managedPlugin, len(candidate.entries))
	for _, entry := range candidate.entries {
		catalog[entry.Descriptor.ID] = managedPlugin{metadata: LoadedPlugin{Enabled: true, Manifest: pluginpackage.Manifest{Requires: entry.Descriptor.Requires}}}
	}
	if _, err := dependencyOrder(catalog); err != nil {
		return nil, err
	}
	if commit != nil {
		if err := commit(); err != nil {
			return nil, err
		}
	}
	var old *lifetime
	if index >= 0 {
		old = r.entries[index].lifetime
	}
	r.entries = candidate.entries
	return old, nil
}

func (r *Registry) initialize(entries []Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.entries) != 0 {
		return fmt.Errorf("registry is not empty")
	}
	r.entries = entries
	return nil
}

// detach removes all manager-owned versions together during shutdown. Normal
// lifecycle transitions still enforce dependency rules.
func (r *Registry) detach(ids map[string]managedPlugin) map[string]*lifetime {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make(map[string]*lifetime)
	r.entries = slices.DeleteFunc(r.entries, func(entry Entry) bool {
		if _, owned := ids[entry.Descriptor.ID]; !owned {
			return false
		}
		result[entry.Descriptor.ID] = entry.lifetime
		return true
	})
	return result
}
