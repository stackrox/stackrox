package watcher

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/baseimage/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
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
		pollInterval: env.BaseImagePollInterval.DurationSetting(),
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

// pollOnce executes a single poll cycle, processing all repositories.
func (w *watcherImpl) pollOnce() {
	log.Info("Starting base image watcher poll cycle")
	start := time.Now()

	ctx := concurrency.AsContext(w.stopper.LowLevel().GetStopRequestSignal())

	repos, err := w.datastore.ListRepositories(ctx)
	if err != nil {
		log.Errorf("Failed to list repositories: %v", err)
		recordPollError("list_repositories")
		return
	}

	if len(repos) == 0 {
		log.Info("No base image repositories configured, skipping poll cycle")
		recordRepositoryCount(0)
		return
	}

	log.Infof("Processing %d base image repositories", len(repos))
	recordRepositoryCount(len(repos))

	// Process repositories concurrently with bounded parallelism
	maxConcurrent := env.BaseImageMaxConcurrentRepositories.IntegerSetting()
	sem := semaphore.NewWeighted(int64(maxConcurrent))
	wg := &sync.WaitGroup{}

	for _, repo := range repos {
		wg.Add(1)

		// Acquire semaphore with context cancellation
		if err := sem.Acquire(ctx, 1); err != nil {
			log.Warnf("Poll cycle interrupted during semaphore acquire: %v", err)
			wg.Done()
			break
		}

		go func(r *storage.BaseImageRepository) {
			defer sem.Release(1)
			defer wg.Done()
			w.processRepository(ctx, r)
		}(repo)
	}

	wg.Wait()

	duration := time.Since(start)
	log.Infof("Poll cycle completed in %v", duration)
	recordPollDuration(duration.Seconds())
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
