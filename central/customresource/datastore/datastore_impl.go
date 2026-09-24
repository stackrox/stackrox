package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/customresource/datastore/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	store store.Store
	db    postgres.DB
}

func (ds *datastoreImpl) GetCustomResource(ctx context.Context, id string) (*storage.CustomResource, bool, error) {
	return ds.store.Get(ctx, id)
}

func (ds *datastoreImpl) ListCustomResources(ctx context.Context, query *v1.Query) ([]*storage.CustomResource, error) {
	return ds.store.GetByQuery(ctx, query)
}

func (ds *datastoreImpl) Search(ctx context.Context, query *v1.Query) ([]search.Result, error) {
	return ds.store.Search(ctx, query)
}

func (ds *datastoreImpl) UpsertCustomResource(ctx context.Context, customResource *storage.CustomResource) error {
	if customResource == nil || customResource.GetId() == "" {
		return errors.New("custom resource id must be defined")
	}
	return ds.store.Upsert(ctx, customResource)
}

func (ds *datastoreImpl) DeleteCustomResource(ctx context.Context, id string) error {
	return ds.store.Delete(ctx, id)
}

func (ds *datastoreImpl) GetOwnedDeploymentIDs(ctx context.Context, id string) ([]string, error) {
	rows, err := ds.db.Query(ctx, `WITH RECURSIVE owned_custom_resources(id) AS (
    SELECT id FROM `+schema.CustomResourcesTableName+` WHERE id = $1
    UNION
    SELECT child.id
    FROM `+schema.CustomResourcesTableName+` child
    JOIN owned_custom_resources parent ON child.ownerid = parent.id
)
SELECT deployment.id
FROM `+schema.DeploymentsTableName+` deployment
JOIN owned_custom_resources custom_resource ON deployment.ownercustomresourceid = custom_resource.id`, pgutils.NilOrUUID(id))
	if err != nil {
		return nil, errors.Wrap(err, "querying custom resource deployments")
	}
	defer rows.Close()

	var deploymentIDs []string
	for rows.Next() {
		var deploymentID string
		if err := rows.Scan(&deploymentID); err != nil {
			return nil, errors.Wrap(err, "scanning custom resource deployment")
		}
		deploymentIDs = append(deploymentIDs, deploymentID)
	}
	return deploymentIDs, rows.Err()
}
