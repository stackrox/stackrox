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
	"github.com/stackrox/rox/central/globaldb"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/metrics"
	nfDS "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	pwDS "github.com/stackrox/rox/central/processbaseline/datastore"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sliceutils"
)

var (
	deploymentsSAC = sac.ForResource(resources.Deployment)
	indicatorSAC   = sac.ForResource(resources.Indicator)
)

type datastoreImpl struct {
	deploymentStore    deploymentStore.Store
	deploymentIndexer  deploymentIndex.Indexer
	deploymentSearcher deploymentSearch.Searcher

	processTagsStore processtagsstore.Store

	images                 imageDS.DataStore
	networkFlows           nfDS.ClusterDataStore
	baselines              pwDS.DataStore
	risks                  riskDS.DataStore
	deletedDeploymentCache expiringcache.Cache
	processFilter          filter.Filter

	keyedMutex *concurrency.KeyedMutex

	clusterRanker    *ranking.Ranker
	nsRanker         *ranking.Ranker
	deploymentRanker *ranking.Ranker
}

func newDatastoreImpl(storage deploymentStore.Store, processTagsStore processtagsstore.Store, indexer deploymentIndex.Indexer, searcher deploymentSearch.Searcher,
	images imageDS.DataStore, baselines pwDS.DataStore, networkFlows nfDS.ClusterDataStore,
	risks riskDS.DataStore, deletedDeploymentCache expiringcache.Cache, processFilter filter.Filter,
	clusterRanker *ranking.Ranker, nsRanker *ranking.Ranker, deploymentRanker *ranking.Ranker) *datastoreImpl {

	ds := &datastoreImpl{
		deploymentStore:        storage,
		processTagsStore:       processTagsStore,
		deploymentIndexer:      indexer,
		deploymentSearcher:     searcher,
		images:                 images,
		baselines:              baselines,
		networkFlows:           networkFlows,
		risks:                  risks,
		keyedMutex:             concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize),
		deletedDeploymentCache: deletedDeploymentCache,
		processFilter:          processFilter,

		clusterRanker:    clusterRanker,
		nsRanker:         nsRanker,
		deploymentRanker: deploymentRanker,
	}
	return ds
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
		deployment, found, err := ds.deploymentStore.Get(readCtx, id)
		if err != nil {
			log.Error(err)
			continue
		} else if !found {
			continue
		}

		riskScore := deployment.GetRiskScore()
		ds.deploymentRanker.Add(id, deployment.GetRiskScore())

		// TODO: ROX-6235: account for nodes in cluster risk
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

// Count returns the number of search results from the query
func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.deploymentSearcher.Count(ctx, q)
}

func (ds *datastoreImpl) ListDeployment(ctx context.Context, id string) (*storage.ListDeployment, bool, error) {
	deployment, found, err := ds.deploymentStore.GetListDeployment(ctx, id)
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
	deployment, found, err := ds.deploymentStore.Get(ctx, id)
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

	deployments, _, err = ds.deploymentStore.GetMany(ctx, ids)
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
		return ds.deploymentStore.Count(ctx)
	}

	return ds.Count(ctx, pkgSearch.EmptyQuery())
}

// UpsertDeployment inserts a deployment into deploymentStore and into the deploymentIndexer
func (ds *datastoreImpl) UpsertDeployment(ctx context.Context, deployment *storage.Deployment) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Deployment", "UpsertDeployment")

	return ds.upsertDeployment(ctx, deployment, true)
}

// upsertDeployment inserts a deployment into deploymentStore and into the deploymentIndexer
func (ds *datastoreImpl) upsertDeployment(ctx context.Context, deployment *storage.Deployment, populateTagsFromExisting bool) error {
	if ok, err := deploymentsSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	// Update deployment with latest risk score
	deployment.RiskScore = ds.deploymentRanker.GetScoreForID(deployment.GetId())

	if features.PostgresDatastore.Enabled() {
		if err := ds.deploymentStore.Upsert(ctx, deployment); err != nil {
			return errors.Wrapf(err, "inserting deployment '%s' to store", deployment.GetId())
		}
		return nil
	}
	return ds.keyedMutex.DoStatusWithLock(deployment.GetId(), func() error {
		if populateTagsFromExisting {
			existingDeployment, _, err := ds.deploymentStore.Get(ctx, deployment.GetId())
			// Best-effort, don't bother checking the error.
			if err == nil && existingDeployment != nil {
				deployment.ProcessTags = existingDeployment.GetProcessTags()
			}
		}
		if err := ds.deploymentStore.Upsert(ctx, deployment); err != nil {
			return errors.Wrapf(err, "inserting deployment '%s' to store", deployment.GetId())
		}
		return nil
	})
}

// RemoveDeployment removes an alert from the deploymentStore and the deploymentIndexer
func (ds *datastoreImpl) RemoveDeployment(ctx context.Context, clusterID, id string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Deployment", "RemoveDeployment")

	if ok, err := deploymentsSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	// Dedupe the removed deployments. This can happen because Pods have many completion states
	// and we may receive multiple Remove calls
	if ds.deletedDeploymentCache.Get(id) != nil {
		return nil
	}
	ds.deletedDeploymentCache.Add(id, true)
	// Though the filter is updated upon pod update,
	// We still want to ensure it is properly cleared when the deployment is deleted.
	ds.processFilter.Delete(id)

	err := ds.keyedMutex.DoStatusWithLock(id, func() error {
		if err := ds.deploymentStore.Delete(ctx, id); err != nil {
			return err
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
			sac.ResourceScopeKeys(resources.NetworkGraph, resources.ProcessWhitelist, resources.Risk),
		))

	if err := ds.risks.RemoveRisk(deleteRelatedCtx, id, storage.RiskSubjectType_DEPLOYMENT); err != nil {
		errorList.AddError(err)
	}

	if err := ds.baselines.RemoveProcessBaselinesByDeployment(deleteRelatedCtx, id); err != nil {
		errorList.AddError(err)
	}

	flowStore, err := ds.networkFlows.GetFlowStore(deleteRelatedCtx, clusterID)
	if err != nil {
		errorList.AddError(err)
		return errorList.ToError()
	}

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

func (ds *datastoreImpl) GetDeploymentIDs(ctx context.Context) ([]string, error) {
	return ds.deploymentStore.GetIDs(ctx)
}

func checkIndicatorWriteSAC(ctx context.Context) error {
	if ok, err := indicatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	return nil
}

// This is a bit ugly -- users can comment on processes, which, as a side-effect, affects tags on deployments.
// However, the tags being stored in the deployment is an internal indexing consistency mechanism for indexing only, and we do NOT
// want to not do this just because the user has process indicator SAC but not deployment SAC.
// Therefore, we use an elevated context for all deployment operations.
func getElevatedCtxForProcessTagOpts() context.Context {
	return sac.WithAllAccess(context.Background())
}

func (ds *datastoreImpl) AddTagsToProcessKey(ctx context.Context, key *analystnotes.ProcessNoteKey, tags []string) error {
	if err := checkIndicatorWriteSAC(ctx); err != nil {
		return err
	}

	elevatedCtx := getElevatedCtxForProcessTagOpts()
	deployment, _, err := ds.GetDeployment(elevatedCtx, key.DeploymentID)
	// It's possible to comment on tags for deployments that no longer exist.
	if err == nil && deployment != nil {
		existingTags := deployment.GetProcessTags()
		unionTags := sliceutils.StringUnion(existingTags, tags)
		sort.Strings(unionTags)
		if !sliceutils.StringEqual(existingTags, unionTags) {
			deployment.ProcessTags = unionTags
			if err := ds.upsertDeployment(elevatedCtx, deployment, false); err != nil {
				return err
			}
		}
	}

	return ds.processTagsStore.UpsertProcessTags(key, tags)
}

func (ds *datastoreImpl) RemoveTagsFromProcessKey(ctx context.Context, key *analystnotes.ProcessNoteKey, tags []string) error {
	if err := checkIndicatorWriteSAC(ctx); err != nil {
		return err
	}

	if err := ds.processTagsStore.RemoveProcessTags(key, tags); err != nil {
		return errors.Wrap(err, "removing from store")
	}
	tagsToRemove := set.NewStringSet(tags...)
	err := ds.processTagsStore.WalkTagsForDeployment(key.DeploymentID, func(tag string) bool {
		// This tag still exists, can't remove it from the deployment.
		tagsToRemove.Remove(tag)
		return tagsToRemove.Cardinality() > 0
	})
	if err != nil {
		return errors.Wrap(err, "walking store")
	}

	if tagsToRemove.Cardinality() == 0 {
		return nil
	}

	elevatedCtx := getElevatedCtxForProcessTagOpts()
	deployment, _, err := ds.GetDeployment(elevatedCtx, key.DeploymentID)
	// It's possible to comment on tags for deployments that no longer exist.
	if err == nil && deployment != nil {
		existingTags := deployment.GetProcessTags()
		diffTags := sliceutils.StringDifference(existingTags, tagsToRemove.AsSlice())
		sort.Strings(diffTags)
		if !sliceutils.StringEqual(existingTags, diffTags) {
			deployment.ProcessTags = diffTags
			if err := ds.upsertDeployment(elevatedCtx, deployment, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func (ds *datastoreImpl) GetTagsForProcessKey(ctx context.Context, key *analystnotes.ProcessNoteKey) ([]string, error) {
	if ok, err := indicatorSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}
	return ds.processTagsStore.GetTagsForProcessKey(key)
}
