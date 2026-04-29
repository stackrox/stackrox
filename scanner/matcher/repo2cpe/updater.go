package repo2cpe

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/quay/zlog"
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
	getter          Getter
	done            chan struct{}
	defaultInterval time.Duration
	failedInterval  time.Duration

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
		getter:          getter,
		done:            make(chan struct{}),
		defaultInterval: defaultRefreshInterval,
		failedInterval:  failedRefreshInterval,
	}
}

// init performs the initial fetch and starts the background refresh loop.
// It is called once on first access via Get.
func (u *Updater) init() {
	ctx := zlog.ContextWithValues(context.Background(), "component", "matcher/repo2cpe/Updater")

	err := retry.WithRetry(
		func() error { return u.fetch(ctx, "") },
		retry.Tries(initMaxRetries),
		retry.WithExponentialBackoff(),
		retry.OnFailedAttempts(func(err error) {
			zlog.Warn(ctx).Err(err).Msg("failed to fetch repo-to-CPE mapping; retrying")
		}),
	)
	if err != nil {
		zlog.Error(ctx).Err(err).Msg("failed initial fetch of repo-to-CPE mapping after retries")
	}

	go u.refreshLoop(ctx)
}

// refreshLoop periodically refreshes the mapping using conditional fetches.
// It uses a shorter interval after a failed refresh.
func (u *Updater) refreshLoop(ctx context.Context) {
	interval := u.defaultInterval
	if u.lastFailed.Load() {
		interval = u.failedInterval
	}
	timer := time.NewTimer(interval)
	defer timer.Stop()

	for {
		select {
		case <-u.done:
			return
		case <-timer.C:
			lastMod := concurrency.WithRLock1(&u.mu, func() string {
				return u.lastModified
			})

			if err := u.fetch(ctx, lastMod); err != nil {
				zlog.Warn(ctx).Err(err).Msg("failed to refresh repo-to-CPE mapping")
				timer.Reset(u.failedInterval)
			} else {
				timer.Reset(u.defaultInterval)
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
	ctx = zlog.ContextWithValues(ctx, "component", "matcher/repo2cpe/Updater.fetch")
	zlog.Debug(ctx).Str("if_modified_since", ifModifiedSince).Msg("fetching repo-to-CPE mapping from indexer")

	result, err := u.getter.GetRepositoryToCPEMapping(ctx, ifModifiedSince)
	if err != nil {
		u.lastFailed.Store(true)
		return err
	}

	u.lastFailed.Store(false)

	concurrency.WithLock(&u.mu, func() {
		u.lastModified = result.LastModified
	})

	if !result.Modified {
		zlog.Debug(ctx).Msg("repo-to-CPE mapping not modified")
		return nil
	}

	u.value.Store(result.Data)
	zlog.Info(ctx).Int("entries", len(result.Data.Data)).Msg("updated repo-to-CPE mapping cache")
	return nil
}

// Get returns the cached repository-to-CPE mapping.
// On first call, it triggers initialization: fetching the mapping and starting
// the background refresh goroutine. Returns an error if no successful fetch has
// ever occurred.
func (u *Updater) Get(_ context.Context) (*repositorytocpe.MappingFile, error) {
	u.initOnce.Do(u.init)

	if v := u.value.Load(); v != nil {
		return v, nil
	}

	return nil, errNoSuccessfulFetch
}
