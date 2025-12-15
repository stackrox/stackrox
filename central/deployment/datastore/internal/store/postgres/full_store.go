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

func (f *fullStoreImpl) GetContainerImageResponses(ctx context.Context) ([]*views.ContainerImagesResponse, error) {
	q, err := common.WithSACFilter(ctx, resources.Deployment, pkgSearch.EmptyQuery())
	if err != nil {
		return nil, err
	}
	q.Selects = []*v1.QuerySelect{
		pkgSearch.NewQuerySelect(pkgSearch.ImageID).Proto(),
		pkgSearch.NewQuerySelect(pkgSearch.ClusterID).Distinct().Proto(),
	}
	q.GroupBy = &v1.QueryGroupBy{
		Fields: []string{pkgSearch.ImageID.String()},
	}

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	var results []*views.ContainerImagesResponse
	err = pgSearch.RunSelectRequestForSchemaFn(queryCtx, f.db, pkgSchema.DeploymentsSchema, q, func(response *views.ContainerImagesResponse) error {
		results = append(results, response)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return results, nil
}

const (
	// Query to get the serialized deployment that contains a container with the given image ID.
	// The deployments_containers table doesn't have a serialized column, but the parent
	// deployments table does, so we join to get the full deployment data.
	getDeploymentByContainerImageIDStmt = `
		SELECT d.serialized
		FROM deployments d
		INNER JOIN deployments_containers dc ON d.id = dc.deployments_id
		WHERE dc.image_idv2 = $1
		LIMIT 1
	`
)

// GetDeploymentContainer queries the deployments table by image ID and returns
// the container with all fields populated from the serialized deployment data.
func (f *fullStoreImpl) GetDeploymentContainer(ctx context.Context, imageID string) (*storage.Container, error) {
	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	row := f.db.QueryRow(queryCtx, getDeploymentByContainerImageIDStmt, imageID)

	var data []byte
	if err := row.Scan(&data); err != nil {
		return nil, err
	}

	var deployment storage.Deployment
	if err := deployment.UnmarshalVTUnsafe(data); err != nil {
		return nil, err
	}

	// Find the container with the matching image ID
	for _, container := range deployment.GetContainers() {
		if container.GetImage().GetIdV2() == imageID {
			return container, nil
		}
	}

	return nil, nil
}

// NewFullTestStore is used for testing.
func NewFullTestStore(ctx context.Context, _ testing.TB, store Store, db postgres.DB, gormDB *gorm.DB) store.Store {
	pkgSchema.ApplySchemaForTable(ctx, gormDB, baseTable)
	return &fullStoreImpl{
		db:    db,
		Store: store,
	}
}
