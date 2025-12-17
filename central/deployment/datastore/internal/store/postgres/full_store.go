package postgres

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/deployment/datastore/internal/store"
	"github.com/stackrox/rox/central/deployment/datastore/internal/store/types"
	"github.com/stackrox/rox/central/deployment/views"
	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac/resources"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"gorm.io/gorm"
)

var (
	queryTimeout = env.PostgresVMStatementTimeout.DurationSetting()
)

// NewFullStore augments the generated store with ListDeployment functions.
func NewFullStore(db postgres.DB) store.Store {
	return &fullStoreImpl{
		Store: New(db),
		db:    db,
	}
}

// FullStoreWrap augments the wrapped store with ListDeployment functions.
func FullStoreWrap(wrapped Store, db postgres.DB) store.Store {
	return &fullStoreImpl{
		Store: wrapped,
		db:    db,
	}
}

type fullStoreImpl struct {
	Store
	db postgres.DB
}

// GetListDeployment returns the list deployment of the passed ID.
func (f *fullStoreImpl) GetListDeployment(ctx context.Context, id string) (*storage.ListDeployment, bool, error) {
	dep, exists, err := f.Get(ctx, id)
	if err != nil || !exists {
		return nil, false, err
	}
	return types.ConvertDeploymentToDeploymentList(dep), true, nil
}

// GetManyListDeployments returns the list deployments as specified by the passed IDs.
func (f *fullStoreImpl) GetManyListDeployments(ctx context.Context, ids ...string) ([]*storage.ListDeployment, []int, error) {
	deployments, missing, err := f.GetMany(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	listDeployments := make([]*storage.ListDeployment, 0, len(deployments))
	for _, d := range deployments {
		listDeployments = append(listDeployments, types.ConvertDeploymentToDeploymentList(d))
	}
	return listDeployments, missing, nil
}

func (f *fullStoreImpl) GetContainerImageViews(ctx context.Context, q *v1.Query) ([]*views.ContainerImageView, error) {
	if err := common.ValidateQuery(q); err != nil {
		return nil, err
	}
	q, err := common.WithSACFilter(ctx, resources.Deployment, q)
	if err != nil {
		return nil, err
	}
	q.Selects = []*v1.QuerySelect{
		pkgSearch.NewQuerySelect(pkgSearch.ImageID).Proto(),
		pkgSearch.NewQuerySelect(pkgSearch.ImageSHA).Proto(),
		pkgSearch.NewQuerySelect(pkgSearch.ClusterID).Distinct().Proto(),
	}
	q.GroupBy = &v1.QueryGroupBy{
		Fields: []string{pkgSearch.ImageID.String(), pkgSearch.ImageSHA.String()},
	}

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	var results []*views.ContainerImageView
	err = pgSearch.RunSelectRequestForSchemaFn(queryCtx, f.db, pkgSchema.DeploymentsSchema, q, func(response *views.ContainerImageView) error {
		results = append(results, response)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return results, nil
}

// NewFullTestStore is used for testing.
func NewFullTestStore(ctx context.Context, _ testing.TB, store Store, db postgres.DB, gormDB *gorm.DB) store.Store {
	pkgSchema.ApplySchemaForTable(ctx, gormDB, baseTable)
	return &fullStoreImpl{
		db:    db,
		Store: store,
	}
}
