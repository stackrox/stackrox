package datastore

import (
	"context"
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/analystnotes"
	"github.com/stackrox/rox/central/deployment/datastore/internal/processtagsstore"
	deploymentSearch "github.com/stackrox/rox/central/deployment/datastore/internal/search"
	deploymentIndex "github.com/stackrox/rox/central/deployment/index"
	deploymentStore "github.com/stackrox/rox/central/deployment/store"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/metrics"
	nfDS "github.com/stackrox/rox/central/networkflow/datastore"
	piDS "github.com/stackrox/rox/central/processindicator/datastore"
	pwDS "github.com/stackrox/rox/central/processwhitelist/datastore"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/batcher"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/debug"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sliceutils"
)

var (
	deploymentsSAC = sac.ForResource(resources.Deployment)
	indicatorSAC   = sac.ForResource(resources.Indicator)
)

const deploymentBatchSize = 500

type datastoreImpl struct {
	deploymentStore    deploymentStore.Store
	deploymentIndexer  deploymentIndex.Indexer
	deploymentSearcher deploymentSearch.Searcher

	processTagsStore processtagsstore.Store

	images                 imageDS.DataStore
	networkFlows           nfDS.ClusterDataStore
	indicators             piDS.DataStore
	whitelists             pwDS.DataStore
	risks                  riskDS.DataStore
	deletedDeploymentCache expiringcache.Cache
	processFilter          filter.Filter

	keyedMutex *concurrency.KeyedMutex

	clusterRanker    *ranking.Ranker
	nsRanker         *ranking.Ranker
	deploymentRanker *ranking.Ranker
}

func newDatastoreImpl(storage deploymentStore.Store, processTagsStore processtagsstore.Store, indexer deploymentIndex.Indexer, searcher deploymentSearch.Searcher,
	images imageDS.DataStore, indicators piDS.DataStore, whitelists pwDS.DataStore, networkFlows nfDS.ClusterDataStore,
	risks riskDS.DataStore, deletedDeploymentCache expiringcache.Cache, processFilter filter.Filter,
	clusterRanker *ranking.Ranker, nsRanker *ranking.Ranker, deploymentRanker *ranking.Ranker, keyedMutex *concurrency.KeyedMutex) (*datastoreImpl, error) {

	ds := &datastoreImpl{
		deploymentStore:        storage,
		processTagsStore:       processTagsStore,
		deploymentIndexer:      indexer,
		deploymentSearcher:     searcher,
		images:                 images,
		indicators:             indicators,
		whitelists:             whitelists,
		networkFlows:           networkFlows,
		risks:                  risks,
		keyedMutex:             keyedMutex,
		deletedDeploymentCache: deletedDeploymentCache,
		processFilter:          processFilter,

		clusterRanker:    clusterRanker,
		nsRanker:         nsRanker,
		deploymentRanker: deploymentRanker,
	}
	if err := ds.buildIndex(); err != nil {
		return nil, err
	}
	return ds, nil
}

func (ds *datastoreImpl) initializeRanker() {
	readCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS), sac.ResourceScopeKeys(resources.Deployment)))

	results, err := ds.Search(readCtx, pkgSearch.EmptyQuery())
	if err != nil {
		log.Error(err)
		return
	}

	clusterScores := make(map[string]float32)
	nsScores := make(map[string]float32)
	for _, id := range pkgSearch.ResultsToIDs(results) {
		deployment, found, err := ds.deploymentStore.GetDeployment(id)
		if err != nil {
			log.Error(err)
			continue
		} else if !found {
			continue
		}

		riskScore := deployment.GetRiskScore()
		ds.deploymentRanker.Add(id, deployment.GetRiskScore())

		// aggregate deployment risk scores to get cluster risk score
		clusterScores[deployment.GetClusterId()] += riskScore

		// aggregate deployment risk scores to obtain namespace risk score
		nsScores[deployment.GetNamespaceId()] += riskScore
	}

	// update namespace risk scores
	for id, score := range nsScores {
		ds.nsRanker.Add(id, score)
	}

	// update cluster risk scores
	for id, score := range clusterScores {
		ds.clusterRanker.Add(id, score)
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
	ds.updateListDeploymentPriority(deployment)
	return deployment, true, nil
}

func (ds *datastoreImpl) SearchListDeployments(ctx context.Context, q *v1.Query) ([]*storage.ListDeployment, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Deployment", "SearchListDeployments")

	listDeployments, err := ds.deploymentSearcher.SearchListDeployments(ctx, q)
	if err != nil {
		return nil, err
	}
	ds.updateListDeploymentPriority(listDeployments...)
	return listDeployments, nil
}

// SearchDeployments
func (ds *datastoreImpl) SearchDeployments(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Deployment", "SearchDeployments")

	return ds.deploymentSearcher.SearchDeployments(ctx, q)
}

// SearchRawDeployments
func (ds *datastoreImpl) SearchRawDeployments(ctx context.Context, q *v1.Query) ([]*storage.Deployment, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Deployment", "SearchRawDeployments")

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

// UpsertDeployment inserts a deployment into deploymentStore and into the deploymentIndexer
func (ds *datastoreImpl) UpsertDeployment(ctx context.Context, deployment *storage.Deployment) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Deployment", "UpsertDeployment")

	return ds.upsertDeployment(ctx, deployment, true, true)
}

// UpsertDeployment inserts a deployment into deploymentStore
func (ds *datastoreImpl) UpsertDeploymentIntoStoreOnly(ctx context.Context, deployment *storage.Deployment) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Deployment", "UpsertDeploymentIntoStoreOnly")

	return ds.upsertDeployment(ctx, deployment, false, true)
}

// UpsertDeployment inserts a deployment into deploymentStore and into the deploymentIndexer
func (ds *datastoreImpl) upsertDeployment(ctx context.Context, deployment *storage.Deployment, indexingRequired bool, populateTagsFromExisting bool) error {
	if ok, err := deploymentsSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	// Update deployment with latest risk score
	deployment.RiskScore = ds.deploymentRanker.GetScoreForID(deployment.GetId())
	if !features.PodDeploymentSeparation.Enabled() {
		ds.processFilter.Update(deployment)
	}

	err := ds.keyedMutex.DoStatusWithLock(deployment.GetId(), func() error {
		if populateTagsFromExisting {
			existingDeployment, _, err := ds.deploymentStore.GetDeployment(deployment.GetId())
			// Best-effort, don't bother checking the error.
			if err == nil && existingDeployment != nil {
				deployment.Tags = existingDeployment.GetTags()
			}
		}
		if err := ds.deploymentStore.UpsertDeployment(deployment); err != nil {
			return errors.Wrapf(err, "inserting deployment '%s' to store", deployment.GetId())
		}
		if indexingRequired && !features.Dackbox.Enabled() {
			if err := ds.deploymentIndexer.AddDeployment(deployment); err != nil {
				return errors.Wrapf(err, "inserting deployment '%s' to index", deployment.GetId())
			}
			if err := ds.deploymentStore.AckKeysIndexed(deployment.GetId()); err != nil {
				return errors.Wrapf(err, "could not acknowledge indexing for %q", deployment.GetId())
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	if features.PodDeploymentSeparation.Enabled() {
		return nil
	}

	if ds.indicators == nil {
		return nil
	}

	deleteIndicatorsCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Indicator)))

	if err := ds.indicators.RemoveProcessIndicatorsOfStaleContainers(deleteIndicatorsCtx, deployment); err != nil {
		log.Errorf("Failed to remove stale process indicators for deployment %s/%s: %s",
			deployment.GetNamespace(), deployment.GetName(), err)
	}
	return nil
}

// RemoveDeployment removes an alert from the deploymentStore and the deploymentIndexer
func (ds *datastoreImpl) RemoveDeployment(ctx context.Context, clusterID, id string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Deployment", "RemoveDeployment")

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
	if !features.PodDeploymentSeparation.Enabled() {
		ds.processFilter.Delete(id)
	}

	err := ds.keyedMutex.DoStatusWithLock(id, func() error {
		if err := ds.deploymentStore.RemoveDeployment(id); err != nil {
			return err
		}

		if !features.Dackbox.Enabled() {
			if err := ds.deploymentIndexer.DeleteDeployment(id); err != nil {
				return err
			}
			if err := ds.deploymentStore.AckKeysIndexed(id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
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
	if !features.PodDeploymentSeparation.Enabled() {
		if err := ds.indicators.RemoveProcessIndicatorsByDeployment(deleteRelatedCtx, id); err != nil {
			errorList.AddError(err)
		}
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

func (ds *datastoreImpl) fullReindex() error {
	log.Info("[STARTUP] Reindexing all deployments")

	deploymentIDs, err := ds.deploymentStore.GetDeploymentIDs()
	if err != nil {
		return err
	}
	log.Infof("[STARTUP] Found %d deployments to index", len(deploymentIDs))
	deploymentBatcher := batcher.New(len(deploymentIDs), deploymentBatchSize)
	for start, end, valid := deploymentBatcher.Next(); valid; start, end, valid = deploymentBatcher.Next() {
		deployments, _, err := ds.deploymentStore.GetDeploymentsWithIDs(deploymentIDs[start:end]...)
		if err != nil {
			return err
		}
		if err := ds.deploymentIndexer.AddDeployments(deployments); err != nil {
			return err
		}
		if end%(deploymentBatchSize*2) == 0 {
			log.Infof("[STARTUP] Successfully indexed %d/%d deployments", end, len(deploymentIDs))
		}
	}
	log.Infof("[STARTUP] Successfully indexed %d deployments", len(deploymentIDs))

	// Clear the keys because we just re-indexed everything
	keys, err := ds.deploymentStore.GetKeysToIndex()
	if err != nil {
		return err
	}
	if err := ds.deploymentStore.AckKeysIndexed(keys...); err != nil {
		return err
	}

	// Write out that initial indexing is complete
	if err := ds.deploymentIndexer.MarkInitialIndexingComplete(); err != nil {
		return err
	}

	return nil
}

func (ds *datastoreImpl) buildIndex() error {
	if features.Dackbox.Enabled() {
		return nil
	}

	defer debug.FreeOSMemory()

	needsReindexing, err := ds.deploymentIndexer.NeedsInitialIndexing()
	if err != nil {
		return err
	}
	if needsReindexing {
		return ds.fullReindex()
	}

	log.Info("[STARTUP] Determining if deployment db/indexer reconciliation is needed")

	deploymentsToIndex, err := ds.deploymentStore.GetKeysToIndex()
	if err != nil {
		return errors.Wrap(err, "error retrieving keys to index")
	}

	log.Infof("[STARTUP] Found %d Deployments to index", len(deploymentsToIndex))

	deploymentBatcher := batcher.New(len(deploymentsToIndex), deploymentBatchSize)
	for start, end, valid := deploymentBatcher.Next(); valid; start, end, valid = deploymentBatcher.Next() {
		deployments, missingIndices, err := ds.deploymentStore.GetDeploymentsWithIDs(deploymentsToIndex[start:end]...)
		if err != nil {
			return err
		}
		if err := ds.deploymentIndexer.AddDeployments(deployments); err != nil {
			return err
		}
		if len(missingIndices) > 0 {
			idsToRemove := make([]string, 0, len(missingIndices))
			for _, missingIdx := range missingIndices {
				idsToRemove = append(idsToRemove, deploymentsToIndex[start:end][missingIdx])
			}
			if err := ds.deploymentIndexer.DeleteDeployments(idsToRemove); err != nil {
				return err
			}
		}

		// Ack keys so that even if central restarts, we don't need to reindex them again
		if err := ds.deploymentStore.AckKeysIndexed(deploymentsToIndex[start:end]...); err != nil {
			return err
		}
		log.Infof("[STARTUP] Successfully indexed %d/%d deployments", end, len(deploymentsToIndex))
	}

	log.Info("[STARTUP] Successfully indexed all out of sync deployments")
	return nil
}

func (ds *datastoreImpl) updateListDeploymentPriority(deployments ...*storage.ListDeployment) {
	for _, deployment := range deployments {
		deployment.Priority = ds.deploymentRanker.GetRankForID(deployment.GetId())
	}
}

func (ds *datastoreImpl) updateDeploymentPriority(deployments ...*storage.Deployment) {
	for _, deployment := range deployments {
		deployment.Priority = ds.deploymentRanker.GetRankForID(deployment.GetId())
	}
}

func (ds *datastoreImpl) GetDeploymentIDs() ([]string, error) {
	return ds.deploymentStore.GetDeploymentIDs()
}

func checkIndicatorWriteSAC(ctx context.Context) error {
	if ok, err := indicatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrPermissionDenied
	}
	return nil
}

func (ds *datastoreImpl) AddTagsToProcessKey(ctx context.Context, key *analystnotes.ProcessNoteKey, tags []string) error {
	if err := checkIndicatorWriteSAC(ctx); err != nil {
		return err
	}

	// This is a bit ugly -- users can comment on processes, which, as a side-effect, affects tags on deployments.
	// However, the tags being stored in the deployment is an internal indexing consistency mechanism for indexing only, and we do NOT
	// want to not do this just because the user has process indicator SAC but not deployment SAC.
	// Therefore, we use an elevated context for all deployment operations.
	elevatedCtx := sac.WithAllAccess(context.Background())
	deployment, _, err := ds.GetDeployment(elevatedCtx, key.DeploymentID)
	// It's possible to comment on tags for deployments that no longer exist.
	if err == nil && deployment != nil {
		existingTags := deployment.GetTags()
		unionTags := sliceutils.StringUnion(existingTags, tags)
		sort.Strings(unionTags)
		deployment.Tags = unionTags
		if err := ds.upsertDeployment(elevatedCtx, deployment, true, false); err != nil {
			return err
		}
	}

	return ds.processTagsStore.UpsertProcessTags(key, tags)
}

func (ds *datastoreImpl) RemoveTagsFromProcessKey(ctx context.Context, key *analystnotes.ProcessNoteKey, tags []string) error {
	if err := checkIndicatorWriteSAC(ctx); err != nil {
		return err
	}

	return ds.processTagsStore.RemoveProcessTags(key, tags)
}

func (ds *datastoreImpl) GetTagsForProcessKey(ctx context.Context, key *analystnotes.ProcessNoteKey) ([]string, error) {
	if ok, err := indicatorSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}
	return ds.processTagsStore.GetTagsForProcessKey(key)
}
