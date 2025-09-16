package datastore

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/deployment/cache"
	deploymentStore "github.com/stackrox/rox/central/deployment/datastore/internal/store"
	"github.com/stackrox/rox/central/globaldb"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/metrics"
	nfDS "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	platformmatcher "github.com/stackrox/rox/central/platform/matcher"
	pwDS "github.com/stackrox/rox/central/processbaseline/datastore"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

var (
	deploymentsSAC = sac.ForResource(resources.Deployment)
)

type datastoreImpl struct {
	deploymentStore deploymentStore.Store

	images                 imageDS.DataStore
	networkFlows           nfDS.ClusterDataStore
	baselines              pwDS.DataStore
	risks                  riskDS.DataStore
	deletedDeploymentCache cache.DeletedDeployments
	processFilter          filter.Filter

	keyedMutex *concurrency.KeyedMutex

	clusterRanker    *ranking.Ranker
	nsRanker         *ranking.Ranker
	deploymentRanker *ranking.Ranker
	platformMatcher  platformmatcher.PlatformMatcher
}

func newDatastoreImpl(
	storage deploymentStore.Store,
	images imageDS.DataStore,
	baselines pwDS.DataStore,
	networkFlows nfDS.ClusterDataStore,
	risks riskDS.DataStore,
	deletedDeploymentCache cache.DeletedDeployments,
	processFilter filter.Filter,
	clusterRanker *ranking.Ranker,
	nsRanker *ranking.Ranker,
	deploymentRanker *ranking.Ranker,
	platformMatcher platformmatcher.PlatformMatcher) *datastoreImpl {
	return &datastoreImpl{
		deploymentStore:        storage,
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
		platformMatcher:  platformMatcher,
	}
}

func (ds *datastoreImpl) initializeRanker() {
	readCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS), sac.ResourceScopeKeys(resources.Deployment)))

	clusterScores := make(map[string]float32)
	nsScores := make(map[string]float32)
	// The store search function does not use select fields, only views do. Hence empty query is used in the walk below
	err := ds.deploymentStore.WalkByQuery(readCtx, pkgSearch.EmptyQuery(), func(deployment *storage.Deployment) error {
		riskScore := deployment.GetRiskScore()
		ds.deploymentRanker.Add(deployment.GetId(), riskScore)

		// TODO: ROX-6235: account for nodes in cluster risk
		// aggregate deployment risk scores to get cluster risk score
		clusterScores[deployment.GetClusterId()] += riskScore

		// aggregate deployment risk scores to obtain namespace risk score
		nsScores[deployment.GetNamespaceId()] += riskScore

		return nil
	})
	if err != nil {
		log.Errorf("unable to initialize deployment ranking: %v", err)
		return
	}

	if ds.nsRanker != nil {
		// update namespace risk scores
		for id, score := range nsScores {
			ds.nsRanker.Add(id, score)
		}
	} else {
		log.Warn("Not updating namespace risk scores, no ranker found")
	}

	if ds.clusterRanker != nil {
		// update cluster risk scores
		for id, score := range clusterScores {
			ds.clusterRanker.Add(id, score)
		}
	} else {
		log.Warn("Not updating cluster risk scores, no ranker found")
	}
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return ds.deploymentStore.Search(ctx, q)
}

// Count returns the number of search results from the query
func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.deploymentStore.Count(ctx, q)
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

	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	ids := pkgSearch.ResultsToIDs(results)
	listDeployments, _, err := ds.deploymentStore.GetManyListDeployments(ctx, ids...)
	if err != nil {
		return nil, err
	}

	ds.updateListDeploymentPriority(listDeployments...)
	return listDeployments, nil
}

// SearchDeployments
func (ds *datastoreImpl) SearchDeployments(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Deployment", "SearchDeployments")
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	ids := pkgSearch.ResultsToIDs(results)
	deployments, missingIndices, err := ds.deploymentStore.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	results = pkgSearch.RemoveMissingResults(results, missingIndices)

	if len(deployments) != len(results) {
		return nil, errors.Errorf("expected %d deployments but got %d", len(results), len(deployments))
	}

	protoResults := make([]*v1.SearchResult, 0, len(deployments))
	for i, deployment := range deployments {
		protoResults = append(protoResults, convertDeployment(deployment, results[i]))
	}
	return protoResults, nil
}

// SearchRawDeployments
func (ds *datastoreImpl) SearchRawDeployments(ctx context.Context, q *v1.Query) ([]*storage.Deployment, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Deployment", "SearchRawDeployments")

	var deployments []*storage.Deployment
	err := ds.deploymentStore.WalkByQuery(ctx, q, func(deployment *storage.Deployment) error {
		deployments = append(deployments, deployment)
		return nil
	})
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
	if _, err := deploymentsSAC.ReadAllowed(ctx); err != nil {
		return 0, err
	}
	return ds.Count(ctx, pkgSearch.EmptyQuery())
}

func (ds *datastoreImpl) WalkByQuery(ctx context.Context, query *v1.Query, fn func(deployment *storage.Deployment) error) error {
	wrappedFn := func(deployment *storage.Deployment) error {
		ds.updateDeploymentPriority(deployment)
		return fn(deployment)
	}
	return ds.deploymentStore.WalkByQuery(ctx, query, wrappedFn)
}

// UpsertDeployment inserts a deployment into deploymentStore
func (ds *datastoreImpl) UpsertDeployment(ctx context.Context, deployment *storage.Deployment) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Deployment", "UpsertDeployment")

	return ds.upsertDeployment(ctx, deployment)
}

func allImagesAreSpecifiedByDigest(d *storage.Deployment) bool {
	for _, c := range d.GetContainers() {
		if c.GetImage().GetId() == "" {
			return false
		}
	}
	return true
}

func (ds *datastoreImpl) mergeCronJobs(ctx context.Context, deployment *storage.Deployment) error {
	if deployment.GetType() != kubernetes.CronJob {
		return nil
	}
	if allImagesAreSpecifiedByDigest(deployment) {
		return nil
	}
	oldDeployment, exists, err := ds.deploymentStore.Get(ctx, deployment.GetId())
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	// Major changes to spec, just upsert
	if len(oldDeployment.GetContainers()) != len(deployment.GetContainers()) {
		return nil
	}
	for i, container := range deployment.GetContainers() {
		if container.GetImage().GetId() != "" {
			continue
		}
		oldContainer := oldDeployment.Containers[i]
		if oldContainer.GetImage().GetId() == "" {
			continue
		}
		if container.GetImage().GetName().GetFullName() != oldContainer.GetImage().GetName().GetFullName() {
			continue
		}
		container.Image.Id = oldContainer.GetImage().GetId()
	}
	return nil
}

// upsertDeployment inserts a deployment into deploymentStore
func (ds *datastoreImpl) upsertDeployment(ctx context.Context, deployment *storage.Deployment) error {
	if ok, err := deploymentsSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	// Update deployment with latest risk score
	deployment.RiskScore = ds.deploymentRanker.GetScoreForID(deployment.GetId())

	// Deployments that run intermittently and do not have images that are referenced by digest
	// should maintain the digest of the last used image
	if err := ds.mergeCronJobs(ctx, deployment); err != nil {
		return errors.Wrapf(err, "error merging deployment %s", deployment.GetId())
	}

	if features.PlatformComponents.Enabled() {
		match, err := ds.platformMatcher.MatchDeployment(deployment)
		if err != nil {
			return err
		}
		deployment.PlatformComponent = match
	}

	if err := ds.deploymentStore.Upsert(ctx, deployment); err != nil {
		return errors.Wrapf(err, "inserting deployment '%s' to store", deployment.GetId())
	}
	return nil
}

// RemoveDeployment removes an alert from the deploymentStore
func (ds *datastoreImpl) RemoveDeployment(ctx context.Context, clusterID, id string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Deployment", "RemoveDeployment")

	if ok, err := deploymentsSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	// Dedupe the removed deployments. This can happen because Pods have many completion states
	// and we may receive multiple Remove calls
	if ds.deletedDeploymentCache != nil {
		if ds.deletedDeploymentCache.Contains(id) {
			return nil
		}
		ds.deletedDeploymentCache.Add(id)
	}
	// Though the filter is updated upon pod update,
	// We still want to ensure it is properly cleared when the deployment is deleted.
	ds.processFilter.Delete(id)

	errorList := errorhelpers.NewErrorList("deleting related objects of deployments")
	deleteRelatedCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph, resources.DeploymentExtension),
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

	// Delete should be last to ensure that the above is always cleaned up even in the case of crash
	err = ds.keyedMutex.DoStatusWithLock(id, func() error {
		if err := ds.deploymentStore.Delete(ctx, id); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
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

// convertDeployment returns proto search result from a deployment object and the internal search result
func convertDeployment(deployment *storage.Deployment, result pkgSearch.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_DEPLOYMENTS,
		Id:             deployment.GetId(),
		Name:           deployment.GetName(),
		FieldToMatches: pkgSearch.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
		Location:       fmt.Sprintf("/%s/%s", deployment.GetClusterName(), deployment.GetNamespace()),
	}
}
