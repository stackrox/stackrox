package datastore

import (
	"context"

	"github.com/pkg/errors"
	deploymentIndex "github.com/stackrox/rox/central/deployment/index"
	deploymentSearch "github.com/stackrox/rox/central/deployment/search"
	deploymentStore "github.com/stackrox/rox/central/deployment/store"
	nfDS "github.com/stackrox/rox/central/networkflow/datastore"
	piDS "github.com/stackrox/rox/central/processindicator/datastore"
	pwDS "github.com/stackrox/rox/central/processwhitelist/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/containerid"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	deploymentStore    deploymentStore.Store
	deploymentIndexer  deploymentIndex.Indexer
	deploymentSearcher deploymentSearch.Searcher

	networkFlows nfDS.ClusterDataStore
	indicators   piDS.DataStore
	whitelists   pwDS.DataStore

	keyedMutex *concurrency.KeyedMutex
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return ds.deploymentIndexer.Search(q)
}

func (ds *datastoreImpl) ListDeployment(ctx context.Context, id string) (*storage.ListDeployment, bool, error) {
	return ds.deploymentStore.ListDeployment(id)
}

func (ds *datastoreImpl) SearchListDeployments(ctx context.Context, q *v1.Query) ([]*storage.ListDeployment, error) {
	return ds.deploymentSearcher.SearchListDeployments(q)
}

// ListDeployments returns all deploymentStore in their minimal form
func (ds *datastoreImpl) ListDeployments(ctx context.Context) ([]*storage.ListDeployment, error) {
	return ds.deploymentStore.ListDeployments()
}

// SearchDeployments
func (ds *datastoreImpl) SearchDeployments(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.deploymentSearcher.SearchDeployments(q)
}

// SearchRawDeployments
func (ds *datastoreImpl) SearchRawDeployments(ctx context.Context, q *v1.Query) ([]*storage.Deployment, error) {
	return ds.deploymentSearcher.SearchRawDeployments(q)
}

// GetDeployment
func (ds *datastoreImpl) GetDeployment(ctx context.Context, id string) (*storage.Deployment, bool, error) {
	return ds.deploymentStore.GetDeployment(id)
}

// GetDeployments
func (ds *datastoreImpl) GetDeployments(ctx context.Context) ([]*storage.Deployment, error) {
	return ds.deploymentStore.GetDeployments()
}

// CountDeployments
func (ds *datastoreImpl) CountDeployments(ctx context.Context) (int, error) {
	return ds.deploymentStore.CountDeployments()
}

func containerIds(deployment *storage.Deployment) (ids []string) {
	for _, container := range deployment.GetContainers() {
		for _, instance := range container.GetInstances() {
			containerID := containerid.ShortContainerIDFromInstance(instance)
			if containerID != "" {
				ids = append(ids, containerID)
			}
		}
	}
	return
}

// UpsertDeployment inserts a deployment into deploymentStore and into the deploymentIndexer
func (ds *datastoreImpl) UpsertDeployment(ctx context.Context, deployment *storage.Deployment) error {
	ds.keyedMutex.Lock(deployment.GetId())
	defer ds.keyedMutex.Unlock(deployment.GetId())
	if err := ds.deploymentStore.UpsertDeployment(deployment); err != nil {
		return errors.Wrapf(err, "inserting deployment '%s' to store", deployment.GetId())
	}
	if err := ds.deploymentIndexer.AddDeployment(deployment); err != nil {
		return errors.Wrapf(err, "inserting deployment '%s' to index", deployment.GetId())
	}

	if err := ds.indicators.RemoveProcessIndicatorsOfStaleContainers(ctx, deployment.GetId(), containerIds(deployment)); err != nil {
		log.Errorf("Failed to remove stale process indicators for deployment %s/%s: %s",
			deployment.GetNamespace(), deployment.GetName(), err)
	}
	return nil
}

// UpdateDeployment updates a deployment in deploymentStore and in the deploymentIndexer
func (ds *datastoreImpl) UpdateDeployment(ctx context.Context, deployment *storage.Deployment) error {
	ds.keyedMutex.Lock(deployment.GetId())
	defer ds.keyedMutex.Unlock(deployment.GetId())
	if err := ds.deploymentStore.UpdateDeployment(deployment); err != nil {
		return err
	}
	return ds.deploymentIndexer.AddDeployment(deployment)
}

// RemoveDeployment removes an alert from the deploymentStore and the deploymentIndexer
func (ds *datastoreImpl) RemoveDeployment(ctx context.Context, clusterID, id string) error {
	ds.keyedMutex.Lock(id)
	defer ds.keyedMutex.Unlock(id)

	if err := ds.deploymentStore.RemoveDeployment(id); err != nil {
		return err
	}
	if err := ds.deploymentIndexer.DeleteDeployment(id); err != nil {
		return err
	}

	if err := ds.whitelists.RemoveProcessWhitelistsByDeployment(ctx, id); err != nil {
		return err
	}
	if err := ds.indicators.RemoveProcessIndicatorsByDeployment(ctx, id); err != nil {
		return err
	}
	flowStore := ds.networkFlows.GetFlowStore(ctx, clusterID)
	return flowStore.RemoveFlowsForDeployment(ctx, id)
}
