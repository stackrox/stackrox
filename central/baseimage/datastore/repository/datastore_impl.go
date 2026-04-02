package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	repoStore "github.com/stackrox/rox/central/baseimage/store/repository/postgres"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	resourceType = "BaseImage"
)

var (
	baseImageRepositorySAC = sac.ForResource(resources.ImageAdministration)
)

// ErrScanInProgress is returned when a config update is attempted while repository scanning is in progress.
var ErrScanInProgress = errors.New("repository scanning in progress")

type datastoreImpl struct {
	store      repoStore.Store
	writeFence concurrency.KeyFence
}

func (d *datastoreImpl) withWriteLock(id string, fn func() error) error {
	return d.writeFence.DoStatusWithLock(concurrency.DiscreteKeySet([]byte(id)), fn)
}

func (d *datastoreImpl) withWriteLockedRepository(ctx context.Context, id string, fn func(repo *storage.BaseImageRepository) error) error {
	return d.withWriteLock(id, func() error {
		repo, found, err := d.store.Get(ctx, id)
		if err != nil {
			return fmt.Errorf("fetching repository %q: %w", id, err)
		}
		if !found {
			return fn(nil)
		}
		return fn(repo)
	})
}

func resetTimestampAndHash(repo *storage.BaseImageRepository) {
	repo.UpdatedAt = timestamppb.Now()
	hash := sha256.Sum256([]byte(repo.GetTagPattern()))
	repo.PatternHash = hex.EncodeToString(hash[:])
}

func resetRepositoryState(repo *storage.BaseImageRepository) {
	resetTimestampAndHash(repo)
	repo.Status = storage.BaseImageRepository_CREATED
	repo.FailureCount = 0
	repo.HealthStatus = storage.BaseImageRepository_HEALTHY
	repo.LastFailureMessage = ""
}

func (d *datastoreImpl) GetRepository(ctx context.Context, id string) (*storage.BaseImageRepository, bool, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "GetRepository")

	if err := sac.VerifyAuthzOK(baseImageRepositorySAC.ReadAllowed(ctx)); err != nil {
		return nil, false, err
	}
	return d.store.Get(ctx, id)
}

func (d *datastoreImpl) ListRepositories(ctx context.Context) ([]*storage.BaseImageRepository, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "ListRepositories")

	if err := sac.VerifyAuthzOK(baseImageRepositorySAC.ReadAllowed(ctx)); err != nil {
		return nil, err
	}

	var repos []*storage.BaseImageRepository
	walkFn := func() error {
		repos = repos[:0]
		return d.store.Walk(ctx, func(obj *storage.BaseImageRepository) error {
			repos = append(repos, obj)
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(ctx, walkFn); err != nil {
		return nil, err
	}
	return repos, nil
}

func (d *datastoreImpl) UpsertRepository(ctx context.Context, repo *storage.BaseImageRepository) (*storage.BaseImageRepository, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "UpsertRepository")

	if err := sac.VerifyAuthzOK(baseImageRepositorySAC.WriteAllowed(ctx)); err != nil {
		return nil, err
	}

	if repo.GetId() == "" {
		repo.Id = uuid.NewV4().String()

		slimUser := authn.UserFromContext(ctx)
		if slimUser == nil {
			return nil, errors.New("could not determine user identity from provided context")
		}
		repo.CreatedBy = slimUser
	}

	resetTimestampAndHash(repo)

	if err := d.store.Upsert(ctx, repo); err != nil {
		return nil, err
	}
	return repo, nil
}

func (d *datastoreImpl) UpdateConfiguration(ctx context.Context, id string, update ConfigUpdate) (*storage.BaseImageRepository, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "UpdateConfiguration")

	if err := sac.VerifyAuthzOK(baseImageRepositorySAC.WriteAllowed(ctx)); err != nil {
		return nil, err
	}

	var updated *storage.BaseImageRepository
	err := d.withWriteLockedRepository(ctx, id, func(existing *storage.BaseImageRepository) error {
		if existing == nil {
			return errox.NotFound.Newf("base image repository %q not found", id)
		}
		switch existing.GetStatus() {
		case storage.BaseImageRepository_QUEUED, storage.BaseImageRepository_IN_PROGRESS:
			return ErrScanInProgress
		}

		if update.RepositoryPath != nil {
			existing.RepositoryPath = *update.RepositoryPath
		}
		if update.TagPattern != nil {
			existing.TagPattern = *update.TagPattern
		}
		resetRepositoryState(existing)

		if err := d.store.Upsert(ctx, existing); err != nil {
			return fmt.Errorf("updating repository: %w", err)
		}
		updated = existing
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (d *datastoreImpl) UpdateStatus(ctx context.Context, id string, update StatusUpdate) (*storage.BaseImageRepository, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "UpdateStatus")

	if err := sac.VerifyAuthzOK(baseImageRepositorySAC.WriteAllowed(ctx)); err != nil {
		return nil, err
	}

	var updated *storage.BaseImageRepository
	err := d.withWriteLockedRepository(ctx, id, func(existing *storage.BaseImageRepository) error {
		if existing == nil {
			return nil
		}

		existing.Status = update.Status
		if update.LastPolledAt != nil {
			existing.LastPolledAt = timestamppb.New(*update.LastPolledAt)
		}
		if update.LastFailureMessage != nil {
			existing.LastFailureMessage = *update.LastFailureMessage
		}
		switch update.FailureCountOp {
		case FailureCountReset:
			existing.FailureCount = 0
		case FailureCountIncrement:
			existing.FailureCount++
		}

		if err := d.store.Upsert(ctx, existing); err != nil {
			return fmt.Errorf("updating repository: %w", err)
		}
		updated = existing
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (d *datastoreImpl) DeleteRepository(ctx context.Context, id string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "DeleteRepository")

	if err := sac.VerifyAuthzOK(baseImageRepositorySAC.WriteAllowed(ctx)); err != nil {
		return err
	}
	return d.withWriteLock(id, func() error {
		return d.store.Delete(ctx, id)
	})
}
