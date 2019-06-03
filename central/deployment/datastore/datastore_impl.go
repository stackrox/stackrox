package datastore

import (
	"context"

	"github.com/pkg/errors"
	deploymentSearch "github.com/stackrox/rox/central/deployment/datastore/internal/search"
	deploymentIndex "github.com/stackrox/rox/central/deployment/index"
	deploymentStore "github.com/stackrox/rox/central/deployment/store"
	"github.com/stackrox/rox/central/globaldb"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	nfDS "github.com/stackrox/rox/central/networkflow/datastore"
	piDS "github.com/stackrox/rox/central/processindicator/datastore"
	pwDS "github.com/stackrox/rox/central/processwhitelist/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/containerid"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

var (
	deploymentsSAC = sac.ForResource(resources.Deployment)
)

type datastoreImpl struct {
	deploymentStore    deploymentStore.Store
	deploymentIndexer  deploymentIndex.Indexer
	deploymentSearcher deploymentSearch.Searcher

	images       imageDS.DataStore
	networkFlows nfDS.ClusterDataStore
	indicators   piDS.DataStore
	whitelists   pwDS.DataStore

	keyedMutex *concurrency.KeyedMutex
}

func newDatastoreImpl(storage deploymentStore.Store, indexer deploymentIndex.Indexer, searcher deploymentSearch.Searcher, images imageDS.DataStore, indicators piDS.DataStore, whitelists pwDS.DataStore, networkFlows nfDS.ClusterDataStore) *datastoreImpl {
	return &datastoreImpl{
		deploymentStore:    storage,
		deploymentIndexer:  indexer,
		deploymentSearcher: searcher,
		images:             images,
		indicators:         indicators,
		whitelists:         whitelists,
		networkFlows:       networkFlows,
		keyedMutex:         concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize),
	}
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return ds.deploymentSearcher.Search(ctx, q)
}

func (ds *datastoreImpl) ListDeployment(ctx context.Context, id string) (*storage.ListDeployment, bool, error) {
	deployment, found, err := ds.deploymentStore.ListDeployment(id)
	if err != nil || !found {
		return nil, false, err
	}

	if ok, err := deploymentsSAC.ReadAllowed(ctx, sac.KeyForNSScopedObj(deployment)...); err != nil || !ok {
		return nil, false, err
	}
	return deployment, true, nil
}

func (ds *datastoreImpl) SearchListDeployments(ctx context.Context, q *v1.Query) ([]*storage.ListDeployment, error) {
	return ds.deploymentSearcher.SearchListDeployments(ctx, q)
}

// ListDeployments returns all deploymentStore in their minimal form
func (ds *datastoreImpl) ListDeployments(ctx context.Context) ([]*storage.ListDeployment, error) {
	if ok, err := deploymentsSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if ok {
		return ds.deploymentStore.ListDeployments()
	}

	return ds.SearchListDeployments(ctx, pkgSearch.EmptyQuery())
}

// SearchDeployments
func (ds *datastoreImpl) SearchDeployments(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.deploymentSearcher.SearchDeployments(ctx, q)
}

// SearchRawDeployments
func (ds *datastoreImpl) SearchRawDeployments(ctx context.Context, q *v1.Query) ([]*storage.Deployment, error) {
	return ds.deploymentSearcher.SearchRawDeployments(ctx, q)
}

// GetDeployment
func (ds *datastoreImpl) GetDeployment(ctx context.Context, id string) (*storage.Deployment, bool, error) {
	deployment, found, err := ds.deploymentStore.GetDeployment(id)
	if err != nil || !found {
		return nil, false, err
	}

	if ok, err := deploymentsSAC.ReadAllowed(ctx, sac.KeyForNSScopedObj(deployment)...); err != nil || !ok {
		return nil, false, err
	}

	return deployment, true, nil
}

// GetDeployments
func (ds *datastoreImpl) GetDeployments(ctx context.Context) ([]*storage.Deployment, error) {
	if ok, err := deploymentsSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return ds.deploymentStore.GetDeployments()
	}

	return ds.SearchRawDeployments(ctx, pkgSearch.EmptyQuery())
}

// CountDeployments
func (ds *datastoreImpl) CountDeployments(ctx context.Context) (int, error) {
	if ok, err := deploymentsSAC.ReadAllowed(ctx); err != nil {
		return 0, err
	} else if ok {
		return ds.deploymentStore.CountDeployments()
	}

	searchResults, err := ds.Search(ctx, pkgSearch.EmptyQuery())
	if err != nil {
		return 0, err
	}
	return len(searchResults), nil
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
	if ok, err := deploymentsSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	ds.keyedMutex.Lock(deployment.GetId())
	defer ds.keyedMutex.Unlock(deployment.GetId())
	if err := ds.deploymentStore.UpsertDeployment(deployment); err != nil {
		return errors.Wrapf(err, "inserting deployment '%s' to store", deployment.GetId())
	}
	if err := ds.deploymentIndexer.AddDeployment(deployment); err != nil {
		return errors.Wrapf(err, "inserting deployment '%s' to index", deployment.GetId())
	}

	if ds.indicators == nil {
		return nil
	}

	deleteIndicatorsCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Indicator)))

	if err := ds.indicators.RemoveProcessIndicatorsOfStaleContainers(deleteIndicatorsCtx, deployment.GetId(), containerIds(deployment)); err != nil {
		log.Errorf("Failed to remove stale process indicators for deployment %s/%s: %s",
			deployment.GetNamespace(), deployment.GetName(), err)
	}
	return nil
}

// UpdateDeployment updates a deployment in deploymentStore and in the deploymentIndexer
func (ds *datastoreImpl) UpdateDeployment(ctx context.Context, deployment *storage.Deployment) error {
	if ok, err := deploymentsSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	ds.keyedMutex.Lock(deployment.GetId())
	defer ds.keyedMutex.Unlock(deployment.GetId())
	if err := ds.deploymentStore.UpdateDeployment(deployment); err != nil {
		return err
	}
	return ds.deploymentIndexer.AddDeployment(deployment)
}

// RemoveDeployment removes an alert from the deploymentStore and the deploymentIndexer
func (ds *datastoreImpl) RemoveDeployment(ctx context.Context, clusterID, id string) error {
	if ok, err := deploymentsSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	ds.keyedMutex.Lock(id)
	defer ds.keyedMutex.Unlock(id)

	if err := ds.deploymentStore.RemoveDeployment(id); err != nil {
		return err
	}
	if err := ds.deploymentIndexer.DeleteDeployment(id); err != nil {
		return err
	}

	deleteRelatedCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Indicator, resources.NetworkGraph, resources.ProcessWhitelist),
		))

	if err := ds.whitelists.RemoveProcessWhitelistsByDeployment(deleteRelatedCtx, id); err != nil {
		return err
	}
	if err := ds.indicators.RemoveProcessIndicatorsByDeployment(deleteRelatedCtx, id); err != nil {
		return err
	}
	flowStore := ds.networkFlows.GetFlowStore(deleteRelatedCtx, clusterID)
	return flowStore.RemoveFlowsForDeployment(deleteRelatedCtx, id)
}

func (ds *datastoreImpl) GetImagesForDeployment(ctx context.Context, deployment *storage.Deployment) ([]*storage.Image, error) {
	imageIDs := make([]string, 0, len(deployment.GetContainers()))
	for _, c := range deployment.GetContainers() {
		if c.GetImage().GetId() != "" {
			imageIDs = append(imageIDs, c.GetImage().GetId())
		}
	}
	imgs, err := ds.images.GetImagesBatch(ctx, imageIDs)
	if err != nil {
		return nil, err
	}
	// Join the images to the container indices
	imageMap := make(map[string]*storage.Image)
	for _, i := range imgs {
		imageMap[i.GetId()] = i
	}
	images := make([]*storage.Image, 0, len(deployment.GetContainers()))
	for _, c := range deployment.GetContainers() {
		img, ok := imageMap[c.GetImage().GetId()]
		if ok {
			images = append(images, img)
		} else {
			images = append(images, types.ToImage(c.GetImage()))
		}
	}
	return images, nil
}
