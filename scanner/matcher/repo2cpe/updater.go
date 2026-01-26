package repo2cpe

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/scannerv4/repositorytocpe"
	"github.com/stackrox/rox/scanner/indexer"
)

// defaultRefreshInterval is how often we attempt to refresh the mapping.
var defaultRefreshInterval = 24 * time.Hour

// Getter provides access to the repository-to-CPE mapping with conditional fetch support.
type Getter interface {
	GetRepositoryToCPEMapping(ctx context.Context, ifModifiedSince string) (*indexer.FetchResult, error)
}

// Updater fetches and caches the repository-to-CPE mapping from an indexer.
// It periodically refreshes the cached value using conditional fetches.
type Updater struct {
	getter Getter

	mu           sync.RWMutex
	value        atomic.Pointer[repositorytocpe.MappingFile]
	lastModified string
}

// NewUpdater creates a new Updater that fetches the repository-to-CPE mapping
// from the given Getter and caches it locally.
func NewUpdater(getter Getter) *Updater {
	return &Updater{
		getter: getter,
	}
}

// Start begins the background refresh loop. It performs an initial fetch
// and then periodically refreshes the mapping using conditional fetches.
//
// Start blocks until the context is cancelled.
func (u *Updater) Start(ctx context.Context) error {
	ctx = zlog.ContextWithValues(ctx, "component", "matcher/repo2cpe/Updater.Start")

	// Initial fetch (unconditional).
	if err := u.fetch(ctx, ""); err != nil {
		zlog.Warn(ctx).Err(err).Msg("failed initial fetch of repo-to-CPE mapping; will retry")
	}

	ticker := time.NewTicker(defaultRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			u.mu.RLock()
			lastMod := u.lastModified
			u.mu.RUnlock()

			if err := u.fetch(ctx, lastMod); err != nil {
				zlog.Warn(ctx).Err(err).Msg("failed to refresh repo-to-CPE mapping")
			}
		}
	}
}

// fetch retrieves the mapping from the indexer using a conditional fetch.
func (u *Updater) fetch(ctx context.Context, ifModifiedSince string) error {
	ctx = zlog.ContextWithValues(ctx, "component", "matcher/repo2cpe/Updater.fetch")
	zlog.Debug(ctx).Str("if_modified_since", ifModifiedSince).Msg("fetching repo-to-CPE mapping from indexer")

	result, err := u.getter.GetRepositoryToCPEMapping(ctx, ifModifiedSince)
	if err != nil {
		return err
	}

	u.mu.Lock()
	u.lastModified = result.LastModified
	u.mu.Unlock()

	if !result.Modified {
		zlog.Debug(ctx).Msg("repo-to-CPE mapping not modified")
		return nil
	}

	u.value.Store(result.Data)
	zlog.Info(ctx).Int("entries", len(result.Data.Data)).Msg("updated repo-to-CPE mapping cache")
	return nil
}

// Get returns the cached repository-to-CPE mapping.
// If no mapping has been fetched yet, it attempts to fetch one synchronously.
// Returns an empty MappingFile if the getter is nil or fetch fails.
func (u *Updater) Get(ctx context.Context) *repositorytocpe.MappingFile {
	// Try to return cached value first.
	if v := u.value.Load(); v != nil {
		return v
	}

	// No cached value yet - try a synchronous fetch.
	if u.getter != nil {
		ctx = zlog.ContextWithValues(ctx, "component", "matcher/repo2cpe/Updater.Get")
		if err := u.fetch(ctx, ""); err != nil {
			zlog.Warn(ctx).Err(err).Msg("failed to fetch repo-to-CPE mapping on first access")
		}
		if v := u.value.Load(); v != nil {
			return v
		}
	}

	// Return empty mapping as fallback.
	return &repositorytocpe.MappingFile{}
}
