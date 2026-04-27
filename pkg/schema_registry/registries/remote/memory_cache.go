package remote

import (
	"context"
	"sync"
	"time"

	"github.com/kaptinlin/jsonschema"

	"github.com/ChargePi/chargeflow/pkg/ocpp"
)

type cachedSchema struct {
	schema   *jsonschema.Schema
	cachedAt time.Time
}

// MemoryCache is a thread-safe in-memory Cache with TTL-based expiry.
type MemoryCache struct {
	mu      sync.RWMutex
	entries map[ocpp.Version]map[string]*cachedSchema
	ttl     time.Duration
}

func NewMemoryCache(ttl time.Duration) *MemoryCache {
	return &MemoryCache{
		entries: make(map[ocpp.Version]map[string]*cachedSchema),
		ttl:     ttl,
	}
}

func (m *MemoryCache) Get(_ context.Context, version ocpp.Version, action string) (*jsonschema.Schema, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	actions, ok := m.entries[version]
	if !ok {
		return nil, false
	}

	entry, ok := actions[action]
	if !ok {
		return nil, false
	}

	if time.Since(entry.cachedAt) >= m.ttl {
		return nil, false
	}

	return entry.schema, true
}

func (m *MemoryCache) Set(_ context.Context, version ocpp.Version, action string, schema *jsonschema.Schema) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.entries[version]; !ok {
		m.entries[version] = make(map[string]*cachedSchema)
	}

	m.entries[version][action] = &cachedSchema{
		schema:   schema,
		cachedAt: time.Now(),
	}
}

func (m *MemoryCache) Delete(_ context.Context, version ocpp.Version, action string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if actions, ok := m.entries[version]; ok {
		delete(actions, action)
	}
}
