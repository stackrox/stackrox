package postgres

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/deployment/datastore/internal/store"
	"github.com/stackrox/rox/central/deployment/views"
	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
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
	listDeployments, missingIndices, err := f.GetManyListDeployments(ctx, id)
	if err != nil || len(missingIndices) > 0 || len(listDeployments) == 0 {
		return nil, false, err
	}
	return listDeployments[0], true, nil
}

// GetManyListDeployments returns the list deployments as specified by the passed IDs.
func (f *fullStoreImpl) GetManyListDeployments(ctx context.Context, ids ...string) ([]*storage.ListDeployment, []int, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	// Build query filtering by deployment IDs
	query := pkgSearch.NewQueryBuilder().
		AddExactMatches(pkgSearch.DeploymentID, ids...).
		ProtoQuery()

	// Set selects to only the columns needed for ListDeployment
	query.Selects = views.ListDeploymentViewSelects()

	// Fetch results using the search framework
	viewMap := make(map[string]*views.ListDeploymentView)
	err := pgSearch.RunSelectRequestForSchemaFn(queryCtx, f.db, pkgSchema.DeploymentsSchema, query,
		func(view *views.ListDeploymentView) error {
			viewMap[view.ID] = view
			return nil
		})
	if err != nil {
		return nil, nil, err
	}

	// It is important that the results are populated in the same order as the input IDs
	// slice, since some calling code relies on that to maintain order.
	listDeployments := make([]*storage.ListDeployment, 0, len(ids))
	var missingIndices []int
	for i, id := range ids {
		if view, ok := viewMap[id]; ok {
			listDeployments = append(listDeployments, view.ToListDeployment())
		} else {
			missingIndices = append(missingIndices, i)
		}
	}

	return listDeployments, missingIndices, nil
}

// SearchListDeployments returns list deployments matching the provided query.
func (f *fullStoreImpl) SearchListDeployments(ctx context.Context, q *v1.Query) ([]*storage.ListDeployment, error) {
	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	// Clone the query to avoid mutating the original
	query := q.CloneVT()
	if query == nil {
		query = pkgSearch.EmptyQuery()
	}

	// Set selects to only the columns needed for ListDeployment
	query.Selects = views.ListDeploymentViewSelects()

	var listDeployments []*storage.ListDeployment
	err := pgSearch.RunSelectRequestForSchemaFn(queryCtx, f.db, pkgSchema.DeploymentsSchema, query,
		func(view *views.ListDeploymentView) error {
			listDeployments = append(listDeployments, view.ToListDeployment())
			return nil
		})
	if err != nil {
		return nil, err
	}

	return listDeployments, nil
}

func (f *fullStoreImpl) GetContainerImageViews(ctx context.Context, q *v1.Query) ([]*views.ContainerImageView, error) {
	if err := common.ValidateQuery(q); err != nil {
		return nil, err
	}
	cloned := q.CloneVT()
	cloned.Selects = []*v1.QuerySelect{
		pkgSearch.NewQuerySelect(pkgSearch.ImageID).Proto(),
		pkgSearch.NewQuerySelect(pkgSearch.ImageSHA).Proto(),
		pkgSearch.NewQuerySelect(pkgSearch.ClusterID).Distinct().Proto(),
	}
	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{pkgSearch.ImageID.String(), pkgSearch.ImageSHA.String()},
	}

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	var results []*views.ContainerImageView
	err := pgSearch.RunSelectRequestForSchemaFn(queryCtx, f.db, pkgSchema.DeploymentsSchema, cloned, func(response *views.ContainerImageView) error {
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
