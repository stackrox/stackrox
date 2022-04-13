package datastore

import (
	"context"

	"github.com/stackrox/rox/central/compliance"
	"github.com/stackrox/rox/central/compliance/datastore/types"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
)

var (
	clusterSAC     = sac.ForResource(resources.Cluster)
	deploymentsSAC = sac.ForResource(resources.Deployment)
	nodeSAC        = sac.ForResource(resources.Node)
)

// SacFilter provides the filtering abilities needed by the compliance datastore.
//
//go:generate mockgen-wrapper
type SacFilter interface {
	FilterRunResults(ctx context.Context, results *storage.ComplianceRunResults) (*storage.ComplianceRunResults, error)
	FilterBatchResults(ctx context.Context, results map[compliance.ClusterStandardPair]types.ResultsWithStatus) (map[compliance.ClusterStandardPair]types.ResultsWithStatus, error)
}

// NewSacFilter returns a new instance of a SacFilter using the input deployment datastore.
func NewSacFilter() SacFilter {
	return &sacFilterImpl{}
}

type sacFilterImpl struct{}

// FilterRunResults filters the deployments and nodes contained in a single ComplianceRunResults to only those that
// the input context has access to.
func (ds *sacFilterImpl) FilterRunResults(ctx context.Context, runResults *storage.ComplianceRunResults) (*storage.ComplianceRunResults, error) {
	if runResults == nil {
		return nil, nil
	}
	filteredDomain, filtered, err := ds.filterDomain(ctx, runResults.Domain)
	if err != nil {
		return nil, err
	}
	if !filtered {
		return runResults, nil
	}

	filteredResults := &storage.ComplianceRunResults{
		Domain:      filteredDomain,
		RunMetadata: runResults.GetRunMetadata(),
	}
	if filteredDomain.GetCluster() != nil {
		filteredResults.ClusterResults = runResults.GetClusterResults()
	}
	if len(filteredDomain.GetNodes()) > 0 {
		filteredResults.NodeResults = runResults.GetNodeResults()
	}
	if len(filteredDomain.GetDeployments()) > 0 {
		if len(filteredResults.GetDeploymentResults()) == len(runResults.GetDomain().GetDeployments()) {
			filteredResults.DeploymentResults = runResults.GetDeploymentResults()
		} else {
			filteredResults.DeploymentResults = make(map[string]*storage.ComplianceRunResults_EntityResults)
			for deploymentID := range filteredDomain.GetDeployments() {
				filteredResults.DeploymentResults[deploymentID] = runResults.GetDeploymentResults()[deploymentID]
			}
		}
	}
	return filteredResults, nil
}

// FilterBatchResults returns a new results map, removing results for the cluster, deployments, and nodes that the input
// context does not have access to.
func (ds *sacFilterImpl) FilterBatchResults(ctx context.Context, batchResults map[compliance.ClusterStandardPair]types.ResultsWithStatus) (map[compliance.ClusterStandardPair]types.ResultsWithStatus, error) {
	clusterIDs := set.NewStringSet()
	for pair := range batchResults {
		clusterIDs.Add(pair.ClusterID)
	}
	allowedClusters, err := ds.filterClusters(ctx, clusterIDs)
	if err != nil {
		return nil, err
	}

	// Create a new map with only the allowed results.
	allowedMap := make(map[compliance.ClusterStandardPair]types.ResultsWithStatus, len(batchResults))
	for pair, batchResult := range batchResults {
		if !allowedClusters.Contains(pair.ClusterID) {
			continue
		}

		// Get and filter the results for the pair.
		batchResult.LastSuccessfulResults, err = ds.FilterRunResults(ctx, batchResult.LastSuccessfulResults)
		if err != nil {
			return nil, err
		}

		// Add the results to filtered returned map.
		allowedMap[pair] = batchResult
	}
	return allowedMap, nil
}

// Helper functions that filter objects.

func (ds *sacFilterImpl) filterClusters(ctx context.Context, clusters set.StringSet) (set.StringSet, error) {
	resourceScopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_ACCESS).Resource(resources.Compliance)

	// Filter the compliance results by cluster.
	allowed := set.NewStringSet()
	for cluster := range clusters {
		if resourceScopeChecker.IsAllowed(sac.ClusterScopeKey(cluster)) {
			allowed.Add(cluster)
		}
	}
	return allowed, nil
}

func (ds *sacFilterImpl) filterDomain(ctx context.Context, domain *storage.ComplianceDomain) (*storage.ComplianceDomain, bool, error) {
	var filtered bool
	newDomain := &storage.ComplianceDomain{}

	ok, err := clusterSAC.ReadAllowed(ctx, sac.ClusterScopeKey(domain.Cluster.Id))
	if err != nil {
		return nil, false, err
	} else if ok {
		newDomain.Cluster = domain.Cluster
	} else {
		filtered = true
	}

	ok, err = nodeSAC.ReadAllowed(ctx, sac.ClusterScopeKey(domain.Cluster.Id))
	if err != nil {
		return nil, false, err
	} else if ok {
		newDomain.Nodes = domain.Nodes
	} else {
		filtered = true
	}

	deploymentsInClusterChecker := deploymentsSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS, sac.ClusterScopeKey(domain.Cluster.Id))
	if deploymentsInClusterChecker.IsAllowed() {
		newDomain.Deployments = domain.Deployments
	} else {
		filteredMap, err := sac.FilterMapReflect(ctx, deploymentsInClusterChecker, domain.Deployments, func(deployment *storage.ComplianceDomain_Deployment) sac.ScopePredicate {
			return sac.ScopeSuffix{sac.NamespaceScopeKey(deployment.GetNamespace())}
		})
		if err != nil {
			return nil, false, err
		}

		newDomain.Deployments = filteredMap.(map[string]*storage.ComplianceDomain_Deployment)
		if len(newDomain.Deployments) < len(domain.Deployments) {
			filtered = true
		}
	}

	return newDomain, filtered, nil
}
