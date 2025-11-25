package watcher

import (
	"context"
	"fmt"
	"time"

	"github.com/stackrox/rox/central/baseimage/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/sync/semaphore"
)

var log = logging.LoggerForModule()

type watcherImpl struct {
	datastore    datastore.DataStore
	pollInterval time.Duration

	stopper     concurrency.Stopper
	startedOnce sync.Once
	stoppedOnce sync.Once
}

// New creates a new base image watcher.
func New(ds datastore.DataStore) Watcher {
	return &watcherImpl{
		datastore:    ds,
		pollInterval: env.BaseImageWatcherPollInterval.DurationSetting(),
		stopper:      concurrency.NewStopper(),
	}
}

// Start spawns the background polling goroutine.
// Subsequent calls are no-ops.
func (w *watcherImpl) Start() {
	w.startedOnce.Do(func() {
		go w.run()
	})
}

// Stop signals shutdown and blocks until polling goroutine exits.
// Subsequent calls are no-ops.
func (w *watcherImpl) Stop() {
	w.stoppedOnce.Do(func() {
		w.stopper.Client().Stop()
		_ = w.stopper.Client().Stopped().Wait()
	})
}

// run is the main polling loop, runs until Stop() is called.
func (w *watcherImpl) run() {
	defer w.stopper.Flow().ReportStopped()

	log.Info("Base image watcher started")

	// Poll immediately on start
	w.pollOnce()

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.pollOnce()
		case <-w.stopper.Flow().StopRequested():
			log.Info("Base image watcher stopped")
			return
		}
	}
}

// pollOnce executes a single poll cycle with metric tracking.
func (w *watcherImpl) pollOnce() {
	start := time.Now()
	err := w.doPoll()
	recordPollDuration(time.Since(start).Seconds(), err)
	if err != nil {
		log.Errorf("Base image watcher poll cycle failed in %v: %v", time.Since(start), err)
	} else {
		log.Infof("Base image watcher poll cycle completed in %v", time.Since(start))
	}
}

// doPoll contains the core poll logic.
func (w *watcherImpl) doPoll() error {
	log.Info("Starting base image watcher poll cycle")

	ctx := concurrency.AsContext(w.stopper.LowLevel().GetStopRequestSignal())

	// Wrap context with SAC scope checker for Administration resource.
	// Base image repositories are administration-level resources requiring READ_ACCESS.
	ctx = sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Administration),
		))

	repos, err := w.datastore.ListRepositories(ctx)
	if err != nil {
		return fmt.Errorf("listing repositories: %w", err)
	}

	if len(repos) == 0 {
		log.Info("No base image repositories configured, skipping poll cycle")
		recordRepositoryCount(0)
		return nil
	}

	log.Infof("Processing %d base image repositories", len(repos))
	recordRepositoryCount(len(repos))

	// Process repositories concurrently with bounded parallelism.
	maxConcurrent := env.BaseImageWatcherMaxConcurrentRepositories.IntegerSetting()
	sem := semaphore.NewWeighted(int64(maxConcurrent))
	wg := &sync.WaitGroup{}

	for _, repo := range repos {
		wg.Add(1)
		if err := sem.Acquire(ctx, 1); err != nil {
			wg.Done()
			return fmt.Errorf("interrupted during semaphore acquire: %w", err)
		}

		go func(r *storage.BaseImageRepository) {
			defer sem.Release(1)
			defer wg.Done()
			w.processRepository(ctx, r)
		}(repo)
	}

	wg.Wait()
	return nil
}

// processRepository processes a single repository.
func (w *watcherImpl) processRepository(ctx context.Context, repo *storage.BaseImageRepository) {
	log.Infof("Processing repository: %s (pattern: %s)",
		repo.GetRepositoryPath(),
		repo.GetTagPattern())

	// Check for context cancellation (shutdown during processing)
	select {
	case <-ctx.Done():
		log.Warnf("Repository processing cancelled: %s", repo.GetRepositoryPath())
		return
	default:
	}

	// TODO(ROX-31921): Add tag listing and pattern matching
	// TODO(ROX-31922): Add metadata fetching

	log.Infof("Repository processed successfully: %s", repo.GetRepositoryPath())
}
