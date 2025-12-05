package repository

import (
	"context"

	repoStore "github.com/stackrox/rox/central/baseimage/store/repository/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	// TODO(ROX-32170): RBAC - review and finalize resource permissions
	baseImageRepositorySAC = sac.ForResource(resources.Administration)
)

type datastoreImpl struct {
	store repoStore.Store
}

func (d *datastoreImpl) GetRepository(ctx context.Context, id string) (*storage.BaseImageRepository, bool, error) {
	if ok, err := baseImageRepositorySAC.ReadAllowed(ctx); err != nil {
		return nil, false, err
	} else if !ok {
		return nil, false, sac.ErrResourceAccessDenied
	}
	return d.store.Get(ctx, id)
}

func (d *datastoreImpl) ListRepositories(ctx context.Context) ([]*storage.BaseImageRepository, error) {
	if ok, err := baseImageRepositorySAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, sac.ErrResourceAccessDenied
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
	if ok, err := baseImageRepositorySAC.WriteAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, sac.ErrResourceAccessDenied
	}

	if repo.GetId() == "" {
		// Generate ID if not provided
		repo.Id = uuid.NewV4().String()
	}

	if err := d.store.Upsert(ctx, repo); err != nil {
		return nil, err
	}
	return repo, nil
}

func (d *datastoreImpl) DeleteRepository(ctx context.Context, id string) error {
	if ok, err := baseImageRepositorySAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	return d.store.Delete(ctx, id)
}
