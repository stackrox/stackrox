package watcher

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	baseImageDS "github.com/stackrox/rox/central/baseimage/datastore"
	repoDS "github.com/stackrox/rox/central/baseimage/datastore/repository"
	tagDS "github.com/stackrox/rox/central/baseimage/datastore/tag"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/baseimage/reposcan"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/delegatedregistry"
	"github.com/stackrox/rox/pkg/env"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"golang.org/x/sync/semaphore"
)

var log = logging.LoggerForModule()

type watcherImpl struct {
	repoDS       repoDS.DataStore
	tagDS        tagDS.DataStore
	baseImageDS  baseImageDS.DataStore
	delegator    delegatedregistry.Delegator
	localScanner reposcan.Scanner

	stopper     concurrency.Stopper
	startedOnce sync.Once
	stoppedOnce sync.Once

	pollInterval     time.Duration
	schedulerCadence time.Duration
	batchSize        int
	tagLimit         int

	delegationEnabled bool
}

// New creates a new base image watcher.
func New(
	repoDS repoDS.DataStore,
	tagDS tagDS.DataStore,
	baseImageDS baseImageDS.DataStore,
	registries registries.Set,
	delegator delegatedregistry.Delegator,
	pollInterval time.Duration,
	schedulerCadence time.Duration,
	batchSize int,
	tagLimit int,
	delegationEnabled bool,
) Watcher {
	return &watcherImpl{
		repoDS:           repoDS,
		tagDS:            tagDS,
		baseImageDS:      baseImageDS,
		delegator:        delegator,
		localScanner:     reposcan.NewLocalScanner(registries),
		stopper:          concurrency.NewStopper(),
		pollInterval:     pollInterval,
		schedulerCadence: schedulerCadence,
		batchSize:        batchSize,
		tagLimit:         tagLimit,

		delegationEnabled: delegationEnabled,
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

// run is the main scheduling loop, runs until Stop() is called.
func (w *watcherImpl) run() {
	defer w.stopper.Flow().ReportStopped()

	log.Info("Base image watcher started")

	// Use scheduler cadence for the ticker, not poll interval.
	// The scheduler checks for due repositories on each tick.
	ticker := time.NewTicker(w.schedulerCadence)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.schedulerPass()
		case <-w.stopper.Flow().StopRequested():
			log.Info("Base image watcher stopped")
			return
		}
	}
}

// schedulerPass executes a single scheduler pass with metric tracking.
func (w *watcherImpl) schedulerPass() {
	start := time.Now()
	claimed, err := w.doSchedulerPass()
	recordPollDuration(time.Since(start).Seconds(), err)
	if err != nil {
		log.Errorf("Base image watcher scheduler pass failed: duration=%v: %v", time.Since(start), err)
	} else if claimed > 0 {
		log.Infof("Base image watcher scheduler pass completed: duration=%v claimed=%d", time.Since(start), claimed)
	} else {
		log.Debugf("Base image watcher scheduler pass completed: duration=%v claimed=%d", time.Since(start), claimed)
	}
}

// doSchedulerPass lists repositories, claims due ones, and scans them.
func (w *watcherImpl) doSchedulerPass() (int, error) {
	log.Debug("Starting base image watcher scheduler pass")

	ctx := concurrency.AsContext(w.stopper.LowLevel().GetStopRequestSignal())
	ctx = sac.WithAllAccess(ctx)

	repos, err := w.repoDS.ListRepositories(ctx)
	if err != nil {
		return 0, fmt.Errorf("listing repositories: %w", err)
	}

	recordRepositoryCount(len(repos))

	if len(repos) == 0 {
		log.Debug("No base image repositories configured")
		return 0, nil
	}

	// Claim due repositories.
	var claimed []*storage.BaseImageRepository
	for _, repo := range repos {
		if !isRepositoryDue(repo, w.pollInterval) {
			continue
		}
		claimedRepo, err := w.repoDS.UpdateStatus(ctx, repo.GetId(), repoDS.StatusUpdate{
			Status: storage.BaseImageRepository_QUEUED,
		})
		if err != nil {
			log.Errorf("Failed to claim repository %q: %v", repo.GetRepositoryPath(), err)
			continue
		}
		if claimedRepo != nil {
			claimed = append(claimed, claimedRepo)
		}
	}

	if len(claimed) == 0 {
		log.Debug("No repositories due for scanning")
		return 0, nil
	}

	log.Debugf("Claimed repositories for scanning: count=%d", len(claimed))

	// Process claimed repositories concurrently with bounded parallelism.
	maxConcurrent := env.BaseImageWatcherMaxConcurrentRepositories.IntegerSetting()
	sem := semaphore.NewWeighted(int64(maxConcurrent))
	wg := &sync.WaitGroup{}
	// Block until all scans complete. This prevents overlapping scheduler passes
	// and ensures predictable scheduling. Status transitions (QUEUED → IN_PROGRESS)
	// prevent the same repository from being claimed by multiple goroutines.
	defer wg.Wait()

	for _, repo := range claimed {
		if err := sem.Acquire(ctx, 1); err != nil {
			return len(claimed), fmt.Errorf("interrupted during semaphore acquire: %w", err)
		}
		wg.Add(1)
		go func(r *storage.BaseImageRepository) {
			defer sem.Release(1)
			defer wg.Done()
			w.scanRepository(ctx, r)
		}(repo)
	}

	return len(claimed), nil
}

// scanRepository scans a single repository that has been claimed (status=QUEUED).
// It transitions the repository to IN_PROGRESS, performs the scan, and then
// sets the final status (READY or FAILED) with last_polled_at.
func (w *watcherImpl) scanRepository(ctx context.Context, repo *storage.BaseImageRepository) {
	log.Debugf("Scanning repository: repository=%q pattern=%q",
		repo.GetRepositoryPath(),
		repo.GetTagPattern())

	_, err := w.repoDS.UpdateStatus(ctx, repo.GetId(), repoDS.StatusUpdate{
		Status: storage.BaseImageRepository_IN_PROGRESS,
	})
	if err != nil {
		log.Errorf("Failed to set IN_PROGRESS for repository %q: %v", repo.GetRepositoryPath(), err)
		return
	}

	// Perform the scan and track success/failure.
	scanErr := w.doScan(ctx, repo)

	// Update final status and last_polled_at.
	finalStatus := storage.BaseImageRepository_READY
	var failureMsg *string
	failureCountOp := repoDS.FailureCountReset
	if scanErr != nil {
		finalStatus = storage.BaseImageRepository_FAILED
		msg := scanErr.Error()
		failureMsg = &msg
		failureCountOp = repoDS.FailureCountIncrement
	}

	now := time.Now()
	_, err = w.repoDS.UpdateStatus(ctx, repo.GetId(), repoDS.StatusUpdate{
		Status:             finalStatus,
		LastPolledAt:       &now,
		LastFailureMessage: failureMsg,
		FailureCountOp:     failureCountOp,
	})
	if err != nil {
		log.Errorf("Failed to update final status for repository %q: %v", repo.GetRepositoryPath(), err)
	}
}

// doScan performs the actual scan of a repository. Returns an error if the scan fails.
func (w *watcherImpl) doScan(ctx context.Context, repo *storage.BaseImageRepository) error {
	// Validate repository ID is a valid UUID.
	if _, err := uuid.FromString(repo.GetId()); err != nil {
		return fmt.Errorf("repository ID is not a valid UUID: id=%q repository=%q: %w",
			repo.GetId(), repo.GetRepositoryPath(), err)
	}

	name, _, err := imageUtils.GenerateImageNameFromString(repo.GetRepositoryPath())
	if err != nil {
		return fmt.Errorf("failed to parse repository path %q: %w", repo.GetRepositoryPath(), err)
	}

	if repo.GetTagPattern() == "" {
		return fmt.Errorf("tag pattern is empty: repository: %q", repo.GetRepositoryPath())
	}

	// Check for context cancellation (shutdown during processing)
	select {
	case <-ctx.Done():
		return fmt.Errorf("repository processing cancelled: repository=%q", repo.GetRepositoryPath())
	default:
	}

	// Determine scanner based on delegation.
	scanner := w.localScanner
	if w.delegationEnabled {
		// Check if scanning should be delegated to a secured cluster. On error, default
		// to Central (same behavior as image enricher).
		clusterID, shouldDelegate, err := w.delegator.GetDelegateClusterID(ctx, name)
		if err != nil {
			log.Warnf("Error checking delegation for repository=%q: %v (continuing with Central-based processing)",
				repo.GetRepositoryPath(), err)
			shouldDelegate = false
		}
		if shouldDelegate {
			scanner = NewDelegatedScanner(w.delegator, clusterID)
		}
	}

	// Fetch existing tags from cache (sorted by created timestamp, newest first).
	tags, err := w.tagDS.ListTagsByRepository(ctx, repo.GetId())
	if err != nil {
		return fmt.Errorf("failed to list tags: repository=%q: %w", repo.GetRepositoryPath(), err)
	}

	// Build scan request.
	req := reposcan.ScanRequest{
		Pattern:   repo.GetTagPattern(),
		CheckTags: make(map[string]*storage.BaseImageTag),
		SkipTags:  make(map[string]struct{}),
	}
	for i, t := range tags {
		if i < env.BaseImageWatcherPerRepoTagLimit.IntegerSetting() {
			req.CheckTags[t.GetTag()] = t
		} else {
			req.SkipTags[t.GetTag()] = struct{}{}
		}
	}
	log.Debugf("Repository scan request: repo=%v check=%d skip=%d",
		repo, len(req.CheckTags), len(req.SkipTags))

	// Scan repository: list tags, fetch metadata, and emit events.
	start := time.Now()

	// Batch accumulators for tags.
	var metadataCount, errorCount, deleteCount int
	var adds []*storage.BaseImageTag
	var dels []string
	var scanErr error

	for event, err := range scanner.ScanRepository(ctx, repo, req) {
		log.Debugf("Processing repository: scan event: err=%v event=%v repo=%v", err, event, repo)

		if err != nil {
			log.Errorf("Error during repository scan: repository=%q: %v", repo.GetRepositoryPath(), err)
			scanErr = err
			break
		}

		if err := validate(event); err != nil {
			log.Errorf("Skipping invalid scan event: repository=%q tag=%q: %v", repo.GetRepositoryPath(), event.Tag, err)
			continue
		}

		// For error tag events this is irrelevant, but harmless.
		tagID, err := tagUUID(repo.GetId(), event.Tag)
		if err != nil {
			utils.Should(fmt.Errorf("failed to generate tag UUID: repository=%q tag=%q: %w", repo.GetRepositoryPath(), event.Tag, err))
			continue
		}

		switch event.Type {
		case reposcan.TagEventMetadata:
			metadata := event.Metadata
			tag := &storage.BaseImageTag{
				Id:                    tagID,
				BaseImageRepositoryId: repo.GetId(),
				Tag:                   event.Tag,
				ManifestDigest:        metadata.ManifestDigest,
				Created:               protocompat.ConvertTimeToTimestampOrNil(metadata.Created),
				LayerDigests:          metadata.LayerDigests,
			}
			adds = append(adds, tag)
			if len(adds) >= w.batchSize {
				if err := w.tagDS.UpsertMany(ctx, adds); err != nil {
					log.Errorf("Failed to flush %d tags: repository=%q: %v", len(adds), repo.GetRepositoryPath(), err)
					errorCount += len(adds)
				} else {
					metadataCount += len(adds)
				}
				adds = adds[:0]
			}

		case reposcan.TagEventDeleted:
			dels = append(dels, tagID)
			if len(dels) >= w.batchSize {
				if err := w.tagDS.DeleteMany(ctx, dels); err != nil {
					log.Errorf("Failed to delete %d tags: repository=%q: %v", len(dels), repo.GetRepositoryPath(), err)
					errorCount += len(dels)
					dels = dels[:0]
					continue
				}
				deleteCount += len(dels)
				dels = dels[:0]
			}

		case reposcan.TagEventError:
			log.Warnf("Tag scan failed: repository=%q tag=%q: %v", repo.GetRepositoryPath(), event.Tag, event.Error)
			errorCount++
		}

	}

	// Final flush of remaining batches.
	if len(adds) > 0 {
		if err := w.tagDS.UpsertMany(ctx, adds); err != nil {
			log.Errorf("Failed to flush %d tags: repository=%q: %v", len(adds), repo.GetRepositoryPath(), err)
			errorCount += len(adds)
		} else {
			metadataCount += len(adds)
		}
	}
	if len(dels) > 0 {
		if err := w.tagDS.DeleteMany(ctx, dels); err != nil {
			log.Errorf("Failed to delete %d tags: repository=%q: %v", len(dels), repo.GetRepositoryPath(), err)
			errorCount += len(dels)
		} else {
			deleteCount += len(dels)
		}
	}

	if err := w.promoteTags(ctx, repo); err != nil {
		log.Errorf("Failed to promote top-%d tags: repository=%q: %v", w.tagLimit, repo.GetRepositoryPath(), err)
	}

	recordScanDuration(name.GetRegistry(), repo.GetRepositoryPath(), scanner.Name(), start, metadataCount, errorCount, nil)

	log.Infof("Repository scan completed: repository=%q pattern=%q processed=%d metadata=%d errors=%d deletes=%d",
		repo.GetRepositoryPath(), repo.GetTagPattern(), metadataCount+errorCount+deleteCount, metadataCount, errorCount, deleteCount)

	return scanErr
}

// promoteTags promotes the top-N tags by created timestamp from cache to base_images.
// This replaces all base_images entries for the repository with the current top-N from cache.
func (w *watcherImpl) promoteTags(
	ctx context.Context,
	repo *storage.BaseImageRepository,
) error {
	// Get top-N tags from cache ordered by created DESC.
	tags, err := w.tagDS.ListTagsByRepository(ctx, repo.GetId())
	if err != nil {
		return errors.Wrap(err, "listing tags from cache")
	}

	if len(tags) > w.tagLimit {
		tags = tags[:w.tagLimit]
	}

	// Build base images from cached tags.
	imgs := make(map[*storage.BaseImage][]string, len(tags))
	for _, tag := range tags {
		bi := &storage.BaseImage{
			Id:                    tag.GetId(),
			BaseImageRepositoryId: tag.GetBaseImageRepositoryId(),
			Repository:            repo.GetRepositoryPath(),
			Tag:                   tag.GetTag(),
			ManifestDigest:        tag.GetManifestDigest(),
			DiscoveredAt:          protocompat.TimestampNow(),
			Active:                true,
			Created:               tag.GetCreated(),
		}
		imgs[bi] = tag.GetLayerDigests()
	}

	// Atomically replace base images for this repository.
	return w.baseImageDS.ReplaceByRepository(ctx, repo.GetId(), imgs)
}

func validate(event reposcan.TagEvent) error {
	if event.Tag == "" {
		return errors.New("tag is empty")
	}
	switch event.Type {
	// Error events have error but no metadata.
	case reposcan.TagEventError:
		if event.Error == nil {
			return errors.New("error event without error")
		}
		if event.Metadata != nil {
			return errors.New("error event containing metadata")
		}
	// Deletion events have no metadata or error.
	case reposcan.TagEventDeleted:
		if event.Metadata != nil {
			return errors.New("deletion event containing metadata")
		}
		if event.Error != nil {
			return errors.New("deletion event containing error")
		}
	// Metadata events must have complete metadata.
	case reposcan.TagEventMetadata:
		if event.Error != nil {
			return errors.New("metadata event containing error")
		}
		if event.Metadata == nil {
			return errors.New("metadata is empty")
		}
		if event.Metadata.Tag != event.Tag {
			return fmt.Errorf("metadata tag %q is different from event tag %q", event.Metadata.Tag, event.Tag)
		}
		if event.Metadata.ManifestDigest == "" {
			return errors.New("metadata manifest digest is empty")
		}
		if len(event.Metadata.LayerDigests) == 0 {
			return errors.New("metadata layers are empty")
		}
		if event.Metadata.Created == nil {
			return errors.New("metadata created timestamp is empty")
		}
	default:
		return fmt.Errorf("unknown event type: %d", event.Type)
	}
	return nil

}

// tagUUID creates a deterministic ID for a tag cache entry.
// Requires repoID to be a valid UUID (validated by processRepository).
func tagUUID(repoID, tag string) (string, error) {
	repoUUID, err := uuid.FromString(repoID)
	if err != nil {
		return "", err
	}
	return uuid.NewV5(repoUUID, tag).String(), nil
}

// isRepositoryDue returns true if a repository is eligible for polling.
func isRepositoryDue(repo *storage.BaseImageRepository, pollInterval time.Duration) bool {
	switch repo.GetStatus() {
	case storage.BaseImageRepository_CREATED,
		storage.BaseImageRepository_QUEUED,
		storage.BaseImageRepository_IN_PROGRESS:
		return true
	case storage.BaseImageRepository_READY, storage.BaseImageRepository_FAILED:
		lastPolled := repo.GetLastPolledAt()
		if lastPolled == nil {
			return true
		}
		return lastPolled.AsTime().Add(pollInterval).Before(time.Now())
	default:
		return false
	}
}
