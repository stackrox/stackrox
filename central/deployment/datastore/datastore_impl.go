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
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/containerid"
	"github.com/stackrox/rox/pkg/debug"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/txn"
)

var (
	deploymentsSAC = sac.ForResource(resources.Deployment)
)

type datastoreImpl struct {
	deploymentStore    deploymentStore.Store
	deploymentIndexer  deploymentIndex.Indexer
	deploymentSearcher deploymentSearch.Searcher

	images                 imageDS.DataStore
	networkFlows           nfDS.ClusterDataStore
	indicators             piDS.DataStore
	whitelists             pwDS.DataStore
	risks                  riskDS.DataStore
	deletedDeploymentCache expiringcache.Cache
	processFilter          filter.Filter

	keyedMutex *concurrency.KeyedMutex

	ranker *ranking.Ranker
}

func newDatastoreImpl(storage deploymentStore.Store, indexer deploymentIndex.Indexer, searcher deploymentSearch.Searcher, images imageDS.DataStore, indicators piDS.DataStore, whitelists pwDS.DataStore, networkFlows nfDS.ClusterDataStore, risks riskDS.DataStore, deletedDeploymentCache expiringcache.Cache, processFilter filter.Filter) (*datastoreImpl, error) {
	ds := &datastoreImpl{
		deploymentStore:        storage,
		deploymentIndexer:      indexer,
		deploymentSearcher:     searcher,
		images:                 images,
		indicators:             indicators,
		whitelists:             whitelists,
		networkFlows:           networkFlows,
		risks:                  risks,
		keyedMutex:             concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize),
		deletedDeploymentCache: deletedDeploymentCache,
		processFilter:          processFilter,
	}
	if err := ds.buildIndex(); err != nil {
		return nil, err
	}
	return ds, nil
}

func (ds *datastoreImpl) initializeRanker() error {
	ds.ranker = ranking.DeploymentRanker()

	riskElevatedCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Risk),
		))

	risks, err := ds.risks.SearchRawRisks(riskElevatedCtx, pkgSearch.NewQueryBuilder().AddStrings(
		pkgSearch.RiskSubjectType, storage.RiskSubjectType_DEPLOYMENT.String()).ProtoQuery())
	if err != nil {
		return err
	}

	for _, risk := range risks {
		ds.ranker.Add(risk.GetSubject().GetId(), risk.GetScore())
	}
	return nil
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
	ds.updateListDeploymentPriority(deployment)
	return deployment, true, nil
}

func (ds *datastoreImpl) SearchListDeployments(ctx context.Context, q *v1.Query) ([]*storage.ListDeployment, error) {
	listDeployments, err := ds.deploymentSearcher.SearchListDeployments(ctx, q)
	if err != nil {
		return nil, err
	}
	ds.updateListDeploymentPriority(listDeployments...)
	return listDeployments, nil
}

// SearchDeployments
func (ds *datastoreImpl) SearchDeployments(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.deploymentSearcher.SearchDeployments(ctx, q)
}

// SearchRawDeployments
func (ds *datastoreImpl) SearchRawDeployments(ctx context.Context, q *v1.Query) ([]*storage.Deployment, error) {
	deployments, err := ds.deploymentSearcher.SearchRawDeployments(ctx, q)
	if err != nil {
		return nil, err
	}
	ds.updateDeploymentPriority(deployments...)
	return deployments, nil
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
	ds.updateDeploymentPriority(deployment)
	return deployment, true, nil
}

// GetDeployments
func (ds *datastoreImpl) GetDeployments(ctx context.Context, ids []string) ([]*storage.Deployment, error) {
	var deployments []*storage.Deployment
	var err error
	if ok, err := deploymentsSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return ds.SearchRawDeployments(ctx, pkgSearch.NewQueryBuilder().AddDocIDs(ids...).ProtoQuery())
	}

	deployments, _, err = ds.deploymentStore.GetDeploymentsWithIDs(ids...)
	if err != nil {
		return nil, err
	}
	ds.updateDeploymentPriority(deployments...)
	return deployments, nil
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
	return ds.upsertDeployment(ctx, deployment, true)
}

// UpsertDeployment inserts a deployment into deploymentStore
func (ds *datastoreImpl) UpsertDeploymentIntoStoreOnly(ctx context.Context, deployment *storage.Deployment) error {
	return ds.upsertDeployment(ctx, deployment, false)
}

// UpsertDeployment inserts a deployment into deploymentStore and into the deploymentIndexer
func (ds *datastoreImpl) upsertDeployment(ctx context.Context, deployment *storage.Deployment, indexingRequired bool) error {
	if ok, err := deploymentsSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	ds.processFilter.Update(deployment)

	ds.keyedMutex.Lock(deployment.GetId())
	defer ds.keyedMutex.Unlock(deployment.GetId())
	if err := ds.deploymentStore.UpsertDeployment(deployment); err != nil {
		return errors.Wrapf(err, "inserting deployment '%s' to store", deployment.GetId())
	}
	if indexingRequired {
		if err := ds.deploymentIndexer.AddDeployment(deployment); err != nil {
			return errors.Wrapf(err, "inserting deployment '%s' to index", deployment.GetId())
		}
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

// RemoveDeployment removes an alert from the deploymentStore and the deploymentIndexer
func (ds *datastoreImpl) RemoveDeployment(ctx context.Context, clusterID, id string) error {
	if ok, err := deploymentsSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}
	// Dedupe the removed deployments. This can happen because Pods have many completion states
	// and we may receive multiple Remove calls
	if ds.deletedDeploymentCache.Get(id) != nil {
		return nil
	}
	ds.deletedDeploymentCache.Add(id, true)
	ds.processFilter.Delete(id)

	ds.keyedMutex.Lock(id)
	defer ds.keyedMutex.Unlock(id)

	if err := ds.deploymentStore.RemoveDeployment(id); err != nil {
		return err
	}

	if err := ds.deploymentIndexer.DeleteDeployment(id); err != nil {
		return err
	}

	errorList := errorhelpers.NewErrorList("deleting related objects of deployments")
	deleteRelatedCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Indicator, resources.NetworkGraph, resources.ProcessWhitelist, resources.Risk),
		))

	if err := ds.risks.RemoveRisk(deleteRelatedCtx, id, storage.RiskSubjectType_DEPLOYMENT); err != nil {
		return err
	}
	if err := ds.whitelists.RemoveProcessWhitelistsByDeployment(deleteRelatedCtx, id); err != nil {
		errorList.AddError(err)
	}
	if err := ds.indicators.RemoveProcessIndicatorsByDeployment(deleteRelatedCtx, id); err != nil {
		errorList.AddError(err)
	}
	flowStore := ds.networkFlows.GetFlowStore(deleteRelatedCtx, clusterID)
	if err := flowStore.RemoveFlowsForDeployment(deleteRelatedCtx, id); err != nil {
		errorList.AddError(err)
	}

	return errorList.ToError()
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

func (ds *datastoreImpl) buildIndex() error {
	defer debug.FreeOSMemory()
	log.Infof("[STARTUP] Determining if deployment db/indexer reconciliation is needed")

	dbTxNum, err := ds.deploymentStore.GetTxnCount()
	if err != nil {
		return err
	}
	indexerTxNum := ds.deploymentIndexer.GetTxnCount()

	if !txn.ReconciliationNeeded(dbTxNum, indexerTxNum) {
		log.Infof("[STARTUP] Reconciliation for deployments is not needed")
		return nil
	}

	log.Info("[STARTUP] Indexing deployments")

	if err := ds.deploymentIndexer.ResetIndex(); err != nil {
		return err
	}

	deployments, err := ds.deploymentStore.GetDeployments()
	if err != nil {
		return err
	}
	if err := ds.deploymentIndexer.AddDeployments(deployments); err != nil {
		return err
	}

	if err := ds.deploymentStore.IncTxnCount(); err != nil {
		return err
	}
	if err := ds.deploymentIndexer.SetTxnCount(dbTxNum + 1); err != nil {
		return err
	}

	log.Info("[STARTUP] Successfully indexed deployments")
	return nil
}

func (ds *datastoreImpl) updateListDeploymentPriority(deployments ...*storage.ListDeployment) {
	for _, deployment := range deployments {
		deployment.Priority = ds.ranker.GetRankForID(deployment.GetId())
	}
}

func (ds *datastoreImpl) updateDeploymentPriority(deployments ...*storage.Deployment) {
	for _, deployment := range deployments {
		deployment.Priority = ds.ranker.GetRankForID(deployment.GetId())
	}
}
