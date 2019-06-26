package datastore

import (
	"context"
	"errors"

	"github.com/stackrox/rox/central/compliance"
	"github.com/stackrox/rox/central/compliance/datastore/internal/store"
	"github.com/stackrox/rox/central/compliance/datastore/types"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	complianceSAC = sac.ForResource(resources.Compliance)
)

type datastoreImpl struct {
	boltStore store.Store
}

func (ds *datastoreImpl) QueryControlResults(ctx context.Context, query *v1.Query) ([]*storage.ComplianceControlResult, error) {
	// TODO(ROX-2575): this might need implementing.
	return nil, errors.New("not yet implemented")
}

func (ds *datastoreImpl) GetLatestRunResults(ctx context.Context, clusterID, standardID string, flags types.GetFlags) (types.ResultsWithStatus, error) {
	if ok, err := complianceSAC.ReadAllowed(ctx, sac.ClusterScopeKey(clusterID)); err != nil {
		return types.ResultsWithStatus{}, err
	} else if !ok {
		return types.ResultsWithStatus{}, errors.New("not found")
	}
	res, err := ds.boltStore.GetLatestRunResults(clusterID, standardID, flags)
	return fromInternalResultsWithStatus(res), err
}

func (ds *datastoreImpl) GetLatestRunResultsBatch(ctx context.Context, clusterIDs, standardIDs []string, flags types.GetFlags) (map[compliance.ClusterStandardPair]types.ResultsWithStatus, error) {
	results, err := ds.boltStore.GetLatestRunResultsBatch(clusterIDs, standardIDs, flags)
	if err != nil {
		return nil, err
	}
	filteredResults, err := ds.filterResults(ctx, sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_ACCESS).Resource(resources.Compliance), results)
	if err != nil {
		return nil, err
	}
	return filteredResults, err
}

func (ds *datastoreImpl) GetLatestRunResultsFiltered(ctx context.Context, clusterIDFilter, standardIDFilter func(string) bool, flags types.GetFlags) (map[compliance.ClusterStandardPair]types.ResultsWithStatus, error) {
	results, err := ds.boltStore.GetLatestRunResultsFiltered(clusterIDFilter, standardIDFilter, flags)
	if err != nil {
		return nil, err
	}
	filteredResults, err := ds.filterResults(ctx, sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_ACCESS).Resource(resources.Compliance), results)
	if err != nil {
		return nil, err
	}
	return filteredResults, err
}

func (ds *datastoreImpl) StoreRunResults(ctx context.Context, results *storage.ComplianceRunResults) error {
	if ok, err := complianceSAC.WriteAllowed(ctx, sac.ClusterScopeKey(results.GetDomain().GetCluster().GetId())); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}
	return ds.boltStore.StoreRunResults(results)
}

func (ds *datastoreImpl) StoreFailure(ctx context.Context, metadata *storage.ComplianceRunMetadata) error {
	if ok, err := complianceSAC.WriteAllowed(ctx, sac.ClusterScopeKey(metadata.GetClusterId())); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}
	return ds.boltStore.StoreFailure(metadata)
}

func (ds *datastoreImpl) filterResults(
	ctx context.Context,
	resourceScopeChecker sac.ScopeChecker,
	results map[compliance.ClusterStandardPair]types.ResultsWithStatus) (map[compliance.ClusterStandardPair]types.ResultsWithStatus, error) {

	allowed, maybe := ds.filterResultsFirst(resourceScopeChecker, results)
	if len(maybe) > 0 {
		if err := resourceScopeChecker.PerformChecks(ctx); err != nil {
			return nil, err
		}
		extraAllowed, maybe := ds.filterResultsSecond(resourceScopeChecker, maybe)
		if len(maybe) > 0 {
			errorhelpers.PanicOnDevelopmentf("still %d maybe results after PerformChecks", len(maybe))
		}
		allowed = append(allowed, extraAllowed...)
	}

	allowedMap := make(map[compliance.ClusterStandardPair]types.ResultsWithStatus, len(allowed))
	for _, pair := range allowed {
		allowedMap[pair] = fromInternalResultsWithStatus(results[pair])
	}
	return allowedMap, nil
}

func (ds *datastoreImpl) filterResultsFirst(
	resourceScopeChecker sac.ScopeChecker,
	results map[compliance.ClusterStandardPair]types.ResultsWithStatus) (allowed []compliance.ClusterStandardPair, maybe []compliance.ClusterStandardPair) {

	for pair := range results {
		if res := resourceScopeChecker.TryAllowed(sac.ClusterScopeKey(pair.ClusterID)); res == sac.Allow {
			allowed = append(allowed, pair)
		} else if res == sac.Unknown {
			maybe = append(maybe, pair)
		}
	}
	return
}

func (ds *datastoreImpl) filterResultsSecond(
	resourceScopeChecker sac.ScopeChecker,
	pairs []compliance.ClusterStandardPair) (allowed []compliance.ClusterStandardPair, maybe []compliance.ClusterStandardPair) {

	for _, pair := range pairs {
		if res := resourceScopeChecker.TryAllowed(sac.ClusterScopeKey(pair.ClusterID)); res == sac.Allow {
			allowed = append(allowed, pair)
		} else if res == sac.Unknown {
			maybe = append(maybe, pair)
		}
	}
	return
}

// Static helper functions.
///////////////////////////
func fromInternalResultsWithStatus(internal types.ResultsWithStatus) types.ResultsWithStatus {
	return types.ResultsWithStatus{
		LastSuccessfulResults: internal.LastSuccessfulResults,
		FailedRuns:            internal.FailedRuns,
	}
}
