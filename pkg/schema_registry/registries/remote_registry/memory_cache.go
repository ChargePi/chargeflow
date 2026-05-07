package remote_registry

import (
	"context"
	"sync"
	"time"

	"github.com/kaptinlin/jsonschema"
)

type cachedSchema struct {
	schema   *jsonschema.Schema
	cachedAt time.Time
}

// MemoryCache is a thread-safe in-memory Cache with TTL-based expiry, keyed by subject name.
type MemoryCache struct {
	mu      sync.RWMutex
	entries map[string]*cachedSchema
	ttl     time.Duration
}

func NewMemoryCache(ttl time.Duration) *MemoryCache {
	return &MemoryCache{
		entries: make(map[string]*cachedSchema),
		ttl:     ttl,
	}
}

func (m *MemoryCache) Get(_ context.Context, subject string) (*jsonschema.Schema, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.entries[subject]
	if !ok {
		return nil, false
	}

	if time.Since(entry.cachedAt) >= m.ttl {
		return nil, false
	}

	return entry.schema, true
}

func (m *MemoryCache) Set(_ context.Context, subject string, schema *jsonschema.Schema) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries[subject] = &cachedSchema{
		schema:   schema,
		cachedAt: time.Now(),
	}
}

func (m *MemoryCache) Delete(_ context.Context, subject string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.entries, subject)
}
