package inmem

import (
	"context"
	"sync"
	"time"
)

// MetadataStore is an ephemeral, in-process implementation of
// scanner/datastore/postgres.MatcherMetadataStore.
//
// Because the in-memory vulnerability snapshot is not persisted across restarts,
// metadata is deliberately ephemeral so the updater reports "never updated" on
// boot and re-imports the vulnerability bundle every start.
type MetadataStore struct {
	mu        sync.Mutex
	timestamp map[string]time.Time
}

// NewMetadataStore returns an empty, ready-to-use ephemeral metadata store.
func NewMetadataStore() *MetadataStore {
	return &MetadataStore{timestamp: make(map[string]time.Time)}
}

// GetLastVulnerabilityUpdate returns the oldest per-bundle update timestamp, or
// the zero time when no bundle has been recorded yet.
func (m *MetadataStore) GetLastVulnerabilityUpdate(_ context.Context) (time.Time, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var oldest time.Time
	for _, t := range m.timestamp {
		if oldest.IsZero() || t.Before(oldest) {
			oldest = t
		}
	}
	return oldest.UTC(), nil
}

// SetLastVulnerabilityUpdate records the update timestamp for one bundle.
func (m *MetadataStore) SetLastVulnerabilityUpdate(_ context.Context, bundle string, lastUpdate time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.timestamp[bundle] = lastUpdate.UTC()
	return nil
}

// GetOrSetLastVulnerabilityUpdate returns the recorded timestamp for the bundle,
// inserting lastUpdate if the bundle has no entry yet.
func (m *MetadataStore) GetOrSetLastVulnerabilityUpdate(_ context.Context, bundle string, lastUpdate time.Time) (time.Time, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.timestamp[bundle]; ok {
		return t.UTC(), nil
	}
	t := lastUpdate.UTC()
	m.timestamp[bundle] = t
	return t, nil
}

// GCVulnerabilityUpdates drops entries for bundles that are neither active nor
// newer than lastUpdate.
func (m *MetadataStore) GCVulnerabilityUpdates(_ context.Context, activeUpdaters []string, lastUpdate time.Time) error {
	active := make(map[string]struct{}, len(activeUpdaters))
	for _, u := range activeUpdaters {
		active[u] = struct{}{}
	}
	cutoff := lastUpdate.UTC()
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, t := range m.timestamp {
		if _, ok := active[k]; ok {
			continue
		}
		if t.Before(cutoff) {
			delete(m.timestamp, k)
		}
	}
	return nil
}
