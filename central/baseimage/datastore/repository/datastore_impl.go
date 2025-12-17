package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/pkg/errors"
	repoStore "github.com/stackrox/rox/central/baseimage/store/repository/postgres"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
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

type datastoreImpl struct {
	store repoStore.Store
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
		// Generate ID if not provided
		repo.Id = uuid.NewV4().String()

		// Fill in the CreatedBy field.
		slimUser := authn.UserFromContext(ctx)
		if slimUser == nil {
			return nil, errors.New("Could not determine user identity from provided context")
		}
		repo.CreatedBy = slimUser
	}

	repo.UpdatedAt = timestamppb.Now()
	hash := sha256.Sum256([]byte(repo.GetTagPattern()))
	repo.PatternHash = hex.EncodeToString(hash[:])
	if err := d.store.Upsert(ctx, repo); err != nil {
		return nil, err
	}
	return repo, nil
}

func (d *datastoreImpl) DeleteRepository(ctx context.Context, id string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "DeleteRepository")

	if err := sac.VerifyAuthzOK(baseImageRepositorySAC.WriteAllowed(ctx)); err != nil {
		return err
	}
	return d.store.Delete(ctx, id)
}
