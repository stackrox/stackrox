package manifest

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/stackrox/rox/clair-adapter/datastore"
)

// ClairClient defines the interface for interacting with Clair's indexer API.
type ClairClient interface {
	DeleteIndexReport(ctx context.Context, digest string) error
}

// Manager garbage-collects expired manifests from both the adapter database and Clair.
type Manager struct {
	metadataStore datastore.IndexerMetadataStore
	clair         ClairClient
	gcInterval    time.Duration
	gcThrottle    int
	cancel        context.CancelFunc
}

// ManagerOption configures the Manager.
type ManagerOption func(*Manager)

// WithGCInterval sets the interval between garbage collection cycles.
// Default is 1 hour.
func WithGCInterval(d time.Duration) ManagerOption {
	return func(m *Manager) {
		m.gcInterval = d
	}
}

// WithGCThrottle sets the maximum number of manifests to garbage collect per cycle.
// Default is 100.
func WithGCThrottle(n int) ManagerOption {
	return func(m *Manager) {
		m.gcThrottle = n
	}
}

// NewManager creates a new manifest garbage collection manager.
func NewManager(metadataStore datastore.IndexerMetadataStore, clair ClairClient, opts ...ManagerOption) *Manager {
	m := &Manager{
		metadataStore: metadataStore,
		clair:         clair,
		gcInterval:    time.Hour,
		gcThrottle:    100,
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// StartGC starts the garbage collection loop and blocks until the context is canceled.
// It runs GC immediately on start, then runs at the configured interval.
func (m *Manager) StartGC(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	defer cancel()

	// Run GC immediately on start
	if err := m.runGC(ctx); err != nil {
		slog.ErrorContext(ctx, "Initial GC failed", "error", err)
	}

	ticker := time.NewTicker(m.gcInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := m.runGC(ctx); err != nil {
				slog.ErrorContext(ctx, "GC cycle failed", "error", err)
			}
		}
	}
}

// StopGC stops the garbage collection loop.
func (m *Manager) StopGC() {
	if m.cancel != nil {
		m.cancel()
	}
}

// runGC performs a single garbage collection cycle.
func (m *Manager) runGC(ctx context.Context) error {
	now := time.Now()

	// Get expired manifest IDs
	expiredIDs, err := m.metadataStore.GCManifests(ctx, now, m.gcThrottle)
	if err != nil {
		return fmt.Errorf("failed to get expired manifests: %w", err)
	}

	if len(expiredIDs) == 0 {
		return nil
	}

	slog.InfoContext(ctx, "Starting GC cycle", "expired_count", len(expiredIDs))

	// Delete each manifest from Clair
	deletedCount := 0
	for _, manifestID := range expiredIDs {
		if err := m.clair.DeleteIndexReport(ctx, manifestID); err != nil {
			slog.WarnContext(ctx, "Failed to delete index report from Clair",
				"manifest_id", manifestID,
				"error", err)
			continue
		}
		deletedCount++
	}

	slog.InfoContext(ctx, "GC cycle completed",
		"deleted", deletedCount,
		"failed", len(expiredIDs)-deletedCount)

	return nil
}
