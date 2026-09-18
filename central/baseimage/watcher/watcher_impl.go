package watcher

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/pkg/errors"
	baseImageDS "github.com/stackrox/rox/central/baseimage/datastore"
	repoDS "github.com/stackrox/rox/central/baseimage/datastore/repository"
	tagDS "github.com/stackrox/rox/central/baseimage/datastore/tag"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/backgroundworker"
	"github.com/stackrox/rox/pkg/baseimage/reposcan"
	"github.com/stackrox/rox/pkg/delegatedregistry"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
)

var log = logging.LoggerForModule()

type watcherImpl struct {
	repoDS       repoDS.DataStore
	tagDS        tagDS.DataStore
	baseImageDS  baseImageDS.DataStore
	delegator    delegatedregistry.Delegator
	localScanner reposcan.Scanner

	pollInterval  time.Duration
	batchSize     int
	tagLimit      int
	maxConcurrent int

	delegationEnabled bool

	worker    *backgroundworker.PeriodicWorker
	startOnce sync.Once
}

// New creates a new base image watcher.
func New(
	repositoryDS repoDS.DataStore,
	tagDS tagDS.DataStore,
	baseImageDS baseImageDS.DataStore,
	registries registries.Set,
	delegator delegatedregistry.Delegator,
	pollInterval time.Duration,
	schedulerCadence time.Duration,
	batchSize int,
	tagLimit int,
	maxConcurrent int,
	delegationEnabled bool,
) Watcher {
	w := &watcherImpl{
		repoDS:            repositoryDS,
		tagDS:             tagDS,
		baseImageDS:       baseImageDS,
		delegator:         delegator,
		localScanner:      reposcan.NewLocalScanner(registries),
		pollInterval:      pollInterval,
		batchSize:         batchSize,
		tagLimit:          tagLimit,
		maxConcurrent:     maxConcurrent,
		delegationEnabled: delegationEnabled,
	}

	w.worker = &backgroundworker.PeriodicWorker{
		Name:       "base-image-watcher",
		Interval:   schedulerCadence,
		RunOnStart: true,
		Run:        w.poll,
	}

	backgroundworker.Global.Register(w.worker)
	return w
}

// Start spawns the background polling goroutine.
// Subsequent calls are no-ops.
func (w *watcherImpl) Start() {
	w.startOnce.Do(func() {
		ctx := sac.WithAllAccess(context.Background())
		w.failInFlightScans(ctx)
		w.worker.Start(ctx)
		log.Info("Base image watcher started")
	})
}

// Stop signals shutdown and blocks until all goroutines exit.
// Safe to call multiple times or before Start.
func (w *watcherImpl) Stop() {
	w.worker.Stop()
	log.Info("Base image watcher stopped")
}

// poll runs a single poll cycle: list repos, claim due ones, scan concurrently.
func (w *watcherImpl) poll(ctx context.Context) error {
	start := time.Now()

	repos, err := w.repoDS.ListRepositories(ctx)
	if err != nil {
		recordPollDuration(time.Since(start).Seconds(), err)
		return fmt.Errorf("listing repositories: %w", err)
	}

	recordRepositoryCount(len(repos))

	if len(repos) == 0 {
		log.Debug("No base image repositories configured")
		recordPollDuration(time.Since(start).Seconds(), nil)
		return nil
	}

	// Sort by last_polled_at ascending so repos waiting longest get priority.
	slices.SortFunc(repos, func(a, b *storage.BaseImageRepository) int {
		return cmp.Compare(a.GetLastPolledAt().AsTime().UnixNano(), b.GetLastPolledAt().AsTime().UnixNano())
	})

	// Claim due repositories and scan with bounded concurrency.
	sem := make(chan struct{}, w.maxConcurrent)
	var wg sync.WaitGroup
	var claimed int

	for _, repo := range repos {
		if ctx.Err() != nil {
			break
		}
		if !isRepositoryDue(repo, w.pollInterval) {
			continue
		}
		r, err := w.repoDS.UpdateStatus(ctx, repo.GetId(), repoDS.StatusUpdate{
			Status: storage.BaseImageRepository_QUEUED,
			OnlyIfStatus: []storage.BaseImageRepository_Status{
				storage.BaseImageRepository_CREATED,
				storage.BaseImageRepository_READY,
				storage.BaseImageRepository_FAILED,
			},
		})
		if err != nil || r == nil {
			if err != nil {
				log.Errorf("Failed to claim repository %q: %v", repo.GetRepositoryPath(), err)
			}
			continue
		}
		claimed++

		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			w.scanRepository(ctx, r)
		}()
	}
	wg.Wait()

	recordPollDuration(time.Since(start).Seconds(), nil)
	if claimed > 0 {
		log.Infof("Base image watcher poll completed: duration=%v claimed=%d", time.Since(start), claimed)
	} else {
		log.Debugf("Base image watcher poll completed: duration=%v claimed=%d", time.Since(start), claimed)
	}
	return nil
}

// failInFlightScans marks repositories in QUEUED or IN_PROGRESS state as FAILED.
func (w *watcherImpl) failInFlightScans(ctx context.Context) {
	repos, err := w.repoDS.ListRepositories(ctx)
	if err != nil {
		log.Errorf("Failed to list repositories: %v", err)
		return
	}

	for _, repo := range repos {
		status := repo.GetStatus()
		if status != storage.BaseImageRepository_QUEUED && status != storage.BaseImageRepository_IN_PROGRESS {
			continue
		}
		msg := "scan interrupted by restart"
		_, err := w.repoDS.UpdateStatus(ctx, repo.GetId(), repoDS.StatusUpdate{
			Status:             storage.BaseImageRepository_FAILED,
			LastFailureMessage: &msg,
			FailureCountOp:     repoDS.FailureCountIncrement,
		})
		if err != nil {
			log.Errorf("Failed to update in-flight scan %q: %v", repo.GetRepositoryPath(), err)
		} else {
			log.Warnf("Marked in-flight scan as failed: %q was %s", repo.GetRepositoryPath(), status)
		}
	}
}

// scanRepository scans a claimed repository (QUEUED → IN_PROGRESS → READY/FAILED).
func (w *watcherImpl) scanRepository(ctx context.Context, repo *storage.BaseImageRepository) {
	log.Debugf("Scanning repository: repository=%q pattern=%q",
		repo.GetRepositoryPath(), repo.GetTagPattern())

	scanErr := w.doScan(ctx, repo)

	now := time.Now()
	update := repoDS.StatusUpdate{
		Status:         storage.BaseImageRepository_READY,
		LastPolledAt:   &now,
		FailureCountOp: repoDS.FailureCountReset,
	}
	if scanErr != nil {
		update.Status = storage.BaseImageRepository_FAILED
		s := scanErr.Error()
		update.LastFailureMessage = &s
		update.FailureCountOp = repoDS.FailureCountIncrement
	}

	if _, err := w.repoDS.UpdateStatus(ctx, repo.GetId(), update); err != nil {
		log.Errorf("Failed to save status for %q: %v", repo.GetRepositoryPath(), err)
	}
}

// doScan performs the actual scan of a repository. Returns an error if the scan fails.
func (w *watcherImpl) doScan(ctx context.Context, repo *storage.BaseImageRepository) error {
	// Transition to IN_PROGRESS.
	_, err := w.repoDS.UpdateStatus(ctx, repo.GetId(), repoDS.StatusUpdate{
		Status: storage.BaseImageRepository_IN_PROGRESS,
	})
	if err != nil {
		return fmt.Errorf("failed to set IN_PROGRESS: %w", err)
	}

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

	if ctx.Err() != nil {
		return fmt.Errorf("repository processing cancelled: repository=%q", repo.GetRepositoryPath())
	}

	// Determine scanner based on delegation.
	scanner := w.localScanner
	if w.delegationEnabled {
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
		if i < w.tagLimit {
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
	case storage.BaseImageRepository_CREATED:
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
