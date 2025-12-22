package watcher

import (
	"context"
	"fmt"
	"time"

	repoDS "github.com/stackrox/rox/central/baseimage/datastore/repository"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/baseimage/reposcan"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/delegatedregistry"
	"github.com/stackrox/rox/pkg/env"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"golang.org/x/sync/semaphore"
)

var log = logging.LoggerForModule()

type watcherImpl struct {
	datastore    repoDS.DataStore
	delegator    delegatedregistry.Delegator
	pollInterval time.Duration
	localScanner *reposcan.LocalScanner

	stopper     concurrency.Stopper
	startedOnce sync.Once
	stoppedOnce sync.Once
}

// New creates a new base image watcher.
func New(ds repoDS.DataStore, registries registries.Set, delegator delegatedregistry.Delegator) Watcher {
	return &watcherImpl{
		datastore:    ds,
		delegator:    delegator,
		pollInterval: env.BaseImageWatcherPollInterval.DurationSetting(),
		localScanner: reposcan.NewLocalScanner(registries),
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
		utils.Should(fmt.Errorf("failed to parse repository path %q: %w", repo.GetRepositoryPath(), err))
		return
	}

	if repo.GetTagPattern() == "" {
		utils.Should(fmt.Errorf("tag pattern is empty: repository: %q", repo.GetRepositoryPath()))
		return
	}

	// Check for context cancellation (shutdown during processing)
	select {
	case <-ctx.Done():
		log.Warnf("Repository processing cancelled: %s", repo.GetRepositoryPath())
		return
	default:
	}

	// Check if scanning should be delegated to a secured cluster. On error, default
	// to Central (same behavior as image enricher).
	clusterID, shouldDelegate, err := w.delegator.GetDelegateClusterID(ctx, name)
	if err != nil {
		log.Warnf("Error checking delegation for %s: %v (continuing with Central-based processing)",
			repo.GetRepositoryPath(), err)
		shouldDelegate = false
	}

	// Determine scanner based on delegation.
	var scanner reposcan.Scanner
	if shouldDelegate {
		scanner = NewDelegatedScanner(w.delegator, clusterID)
	} else {
		scanner = w.localScanner
	}

	// Build scan request.
	// TODO(ROX-31923): Populate CheckTags and SkipTags from cache for incremental updates.
	req := reposcan.ScanRequest{
		Pattern:   repo.GetTagPattern(),
		CheckTags: make(map[string]*storage.BaseImageTag),
		SkipTags:  make(map[string]struct{}),
	}

	// Scan repository: list tags, fetch metadata, and emit events.
	start := time.Now()
	var metadataCount, errorCount int
	for event, err := range scanner.ScanRepository(ctx, repo, req) {
		if err != nil {
			log.Errorf("scanning repository %q: %v", repo.GetRepositoryPath(), err)
			recordScanDuration(name.GetRegistry(), repo.GetRepositoryPath(), scanner.Name(), start, 0, 0, err)
			return
		}

		switch event.Type {
		case reposcan.TagEventMetadata:
			metadataCount++
			log.Debugf("Tag %s: digest=%s, created=%v",
				event.Tag, event.Metadata.ManifestDigest, event.Metadata.Created)
			// TODO(ROX-31923): Store tag metadata in cache.

		case reposcan.TagEventDeleted:
			log.Infof("Tag %s was deleted from registry", event.Tag)
			// TODO(ROX-31923): Remove tag from cache.

		case reposcan.TagEventError:
			errorCount++
			log.Warnf("Failed to fetch metadata for tag %s: %v", event.Tag, event.Error)
		}
	}

	recordScanDuration(name.GetRegistry(), repo.GetRepositoryPath(), scanner.Name(), start, metadataCount, errorCount, nil)
	log.Infof("Repository %s: processed %d tags (%d metadata, %d errors) with pattern %q",
		repo.GetRepositoryPath(), metadataCount+errorCount, metadataCount, errorCount, repo.GetTagPattern())
}
