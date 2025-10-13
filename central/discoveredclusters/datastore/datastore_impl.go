package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/convert/typetostorage"
	"github.com/stackrox/rox/central/discoveredclusters/datastore/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudsources/discoveredclusters"
	"github.com/stackrox/rox/pkg/errox"
)

var _ DataStore = (*datastoreImpl)(nil)

type datastoreImpl struct {
	store store.Store
}

func (ds *datastoreImpl) CountDiscoveredClusters(ctx context.Context, query *v1.Query) (int, error) {
	count, err := ds.store.Count(ctx, query)
	if err != nil {
		return 0, errors.Wrap(err, "failed to count discovered clusters")
	}
	return count, nil
}

func (ds *datastoreImpl) GetDiscoveredCluster(ctx context.Context, id string) (*storage.DiscoveredCluster, error) {
	discoveredCluster, exists, err := ds.store.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get discovered cluster")
	}
	if !exists {
		return nil, errox.NotFound.Newf("discovered cluster %q not found", id)
	}
	return discoveredCluster, nil
}

func (ds *datastoreImpl) ListDiscoveredClusters(ctx context.Context, query *v1.Query) ([]*storage.DiscoveredCluster, error) {
	discoveredClusters, err := ds.store.GetByQuery(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list discovered clusters")
	}
	return discoveredClusters, nil
}

func (ds *datastoreImpl) UpsertDiscoveredClusters(ctx context.Context,
	discoveredClusters ...*discoveredclusters.DiscoveredCluster,
) error {
	upsertList := []*storage.DiscoveredCluster{}
	for _, dc := range discoveredClusters {
		if err := dc.Validate(); err != nil {
			return errox.InvalidArgs.CausedBy(err)
		}
		upsertList = append(upsertList, typetostorage.DiscoveredCluster(dc))
	}
	if err := ds.store.UpsertMany(ctx, upsertList); err != nil {
		return errors.Wrap(err, "failed to upsert discovered clusters")
	}
	return nil
}

func (ds *datastoreImpl) DeleteDiscoveredClusters(ctx context.Context, query *v1.Query) error {
	err := ds.store.DeleteByQuery(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to delete discovered clusters")
	}
	return nil
}
