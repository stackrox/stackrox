package watcher

import (
	"context"
	"fmt"
	"time"

	repoDS "github.com/stackrox/rox/central/baseimage/datastore/repository"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/delegatedregistry"
	"github.com/stackrox/rox/pkg/env"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/sync/semaphore"
)

var log = logging.LoggerForModule()

type watcherImpl struct {
	datastore    repoDS.DataStore
	registries   registries.Set
	delegator    delegatedregistry.Delegator
	pollInterval time.Duration

	stopper     concurrency.Stopper
	startedOnce sync.Once
	stoppedOnce sync.Once
}

// New creates a new base image watcher.
func New(ds repoDS.DataStore, registries registries.Set, delegator delegatedregistry.Delegator) Watcher {
	return &watcherImpl{
		datastore:    ds,
		registries:   registries,
		delegator:    delegator,
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

	// Use all access since the watcher is an internal Central component.
	ctx = sac.WithAllAccess(ctx)

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
	log.Infof("Processing repository: %q: pattern: %q",
		repo.GetRepositoryPath(),
		repo.GetTagPattern())

	name, _, err := imageUtils.GenerateImageNameFromString(repo.GetRepositoryPath())
	if err != nil {
		log.Errorf("Failed to parse repository path %q: %v", repo.GetRepositoryPath(), err)
		return
	}

	// Check for context cancellation (shutdown during processing)
	select {
	case <-ctx.Done():
		log.Warnf("Repository processing cancelled: %s", repo.GetRepositoryPath())
		return
	default:
	}

	// Check if scanning should be delegated to a secured cluster.  On error, default
	// to Central (same behavior as image enricher).
	clusterID, shouldDelegate, err := w.delegator.GetDelegateClusterID(ctx, name)
	if err != nil {
		log.Warnf("Error checking delegation for %s: %v (continuing with Central-based processing)",
			repo.GetRepositoryPath(), err)
	} else if shouldDelegate {
		log.Infof("Repository %s would be delegated to cluster %s (delegation not implemented yet, skipping)",
			repo.GetRepositoryPath(), clusterID)
		// TODO(ROX-31926/ROX-31927): Implement delegated tag listing via Sensor.
		return
	}

	// Find matching registry integration for this repository.
	var reg types.Registry
	for _, r := range w.registries.GetAll() {
		if r.Match(name) {
			reg = r
			break
		}
	}
	if reg == nil {
		log.Errorf("No matching image integration found for repository %s", repo.GetRepositoryPath())
		return
	}

	// List and filter tags on the repository.
	start := time.Now()
	tags, err := listAndFilterTags(ctx, reg, name.GetRemote(), repo.GetTagPattern())
	recordTagListDuration(name.GetRegistry(), repo.GetRepositoryPath(), start, len(tags), err)
	if err != nil {
		log.Errorf("list and filter tags: repository %q: pattern %q: %v",
			repo.GetRepositoryPath(), repo.GetTagPattern(), err)
		return
	}

	log.Infof("Found %d matching tags for repository %s with pattern %q",
		len(tags), repo.GetRepositoryPath(), repo.GetTagPattern())

	// TODO(ROX-31922): Add metadata fetching for discovered tags

	log.Infof("Repository processed successfully: %s", repo.GetRepositoryPath())
}
