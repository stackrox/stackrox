package repo2cpe

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/scannerv4/repositorytocpe"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/scanner/indexer"
)

var (
	defaultRefreshInterval = 24 * time.Hour
	failedRefreshInterval  = 5 * time.Minute
	initMaxRetries         = 3
)

var errNoSuccessfulFetch = errors.New("repo-to-CPE mapping has never been successfully fetched")

// Getter provides access to the repository-to-CPE mapping with conditional fetch support.
//
//go:generate mockgen-wrapper
type Getter interface {
	GetRepositoryToCPEMapping(ctx context.Context, ifModifiedSince string) (*indexer.FetchResult, error)
}

// Updater fetches and caches the repository-to-CPE mapping from an indexer.
// It periodically refreshes the cached value using conditional fetches.
// Initialization is lazy: the first call to Get triggers the initial fetch
// and starts the background refresh goroutine.
type Updater struct {
	getter Getter
	done   chan struct{}

	initOnce     sync.Once
	mu           sync.RWMutex
	value        atomic.Pointer[repositorytocpe.MappingFile]
	lastModified string
	lastFailed   atomic.Bool
}

// NewUpdater creates a new Updater that fetches the repository-to-CPE mapping
// from the given Getter and caches it locally.
func NewUpdater(getter Getter) *Updater {
	return &Updater{
		getter: getter,
		done:   make(chan struct{}),
	}
}

// init performs the initial fetch and starts the background refresh loop.
// It is called once on first access via Get.
func (u *Updater) init() {
	ctx := context.Background()

	err := retry.WithRetry(
		func() error { return u.fetch(ctx, "") },
		retry.Tries(initMaxRetries),
		retry.WithExponentialBackoff(),
		retry.OnFailedAttempts(func(err error) {
			slog.WarnContext(ctx, "failed to fetch repo-to-CPE mapping; retrying", "reason", err)
		}),
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed initial fetch of repo-to-CPE mapping after retries", "reason", err)
	}

	go u.refreshLoop(ctx)
}

// refreshLoop periodically refreshes the mapping using conditional fetches.
func (u *Updater) refreshLoop(ctx context.Context) {
	ticker := time.NewTicker(defaultRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-u.done:
			return
		case <-ticker.C:
			lastMod := concurrency.WithRLock1(&u.mu, func() string {
				return u.lastModified
			})

			if err := u.fetch(ctx, lastMod); err != nil {
				slog.WarnContext(ctx, "failed to refresh repo-to-CPE mapping", "reason", err)
			}
		}
	}
}

// Close stops the background refresh goroutine.
func (u *Updater) Close() {
	close(u.done)
}

// fetch retrieves the mapping from the indexer using a conditional fetch.
func (u *Updater) fetch(ctx context.Context, ifModifiedSince string) error {
	slog.DebugContext(ctx, "fetching repo-to-CPE mapping from indexer", "if_modified_since", ifModifiedSince)

	result, err := u.getter.GetRepositoryToCPEMapping(ctx, ifModifiedSince)
	if err != nil {
		return err
	}

	concurrency.WithLock(&u.mu, func() {
		u.lastModified = result.LastModified
	})

	if !result.Modified {
		slog.DebugContext(ctx, "repo-to-CPE mapping not modified")
		return nil
	}

	u.value.Store(result.Data)
	slog.InfoContext(ctx, "updated repo-to-CPE mapping cache", "entries", len(result.Data.Data))
	return nil
}

// Get returns the cached repository-to-CPE mapping.
// On first call, it triggers initialization: fetching the mapping and starting
// the background refresh goroutine. Returns an empty MappingFile if fetch fails.
func (u *Updater) Get(_ context.Context) *repositorytocpe.MappingFile {
	// Lazy initialization on first access.
	u.initOnce.Do(u.init)

	if v := u.value.Load(); v != nil {
		return v
	}

	// Return empty mapping as fallback.
	return &repositorytocpe.MappingFile{}
}
