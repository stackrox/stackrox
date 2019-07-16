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
	"github.com/stackrox/rox/pkg/sac"
)

var (
	complianceSAC = sac.ForResource(resources.Compliance)
)

type datastoreImpl struct {
	storage store.Store
	filter  SacFilter
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

	res, err := ds.storage.GetLatestRunResults(clusterID, standardID, flags)
	if err != nil {
		return types.ResultsWithStatus{}, err
	}

	// Filter out results the user is not allowed to see.
	res.LastSuccessfulResults, err = ds.filter.FilterRunResults(ctx, res.LastSuccessfulResults)
	if err != nil {
		return types.ResultsWithStatus{}, err
	}
	return res, err
}

func (ds *datastoreImpl) GetLatestRunResultsBatch(ctx context.Context, clusterIDs, standardIDs []string, flags types.GetFlags) (map[compliance.ClusterStandardPair]types.ResultsWithStatus, error) {
	results, err := ds.storage.GetLatestRunResultsBatch(clusterIDs, standardIDs, flags)
	if err != nil {
		return nil, err
	}
	filteredResults, err := ds.filter.FilterBatchResults(ctx, results)
	if err != nil {
		return nil, err
	}
	return filteredResults, err
}

func (ds *datastoreImpl) GetLatestRunResultsFiltered(ctx context.Context, clusterIDFilter, standardIDFilter func(string) bool, flags types.GetFlags) (map[compliance.ClusterStandardPair]types.ResultsWithStatus, error) {
	results, err := ds.storage.GetLatestRunResultsFiltered(clusterIDFilter, standardIDFilter, flags)
	if err != nil {
		return nil, err
	}
	filteredResults, err := ds.filter.FilterBatchResults(ctx, results)
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
	return ds.storage.StoreRunResults(results)
}

func (ds *datastoreImpl) StoreFailure(ctx context.Context, metadata *storage.ComplianceRunMetadata) error {
	if ok, err := complianceSAC.WriteAllowed(ctx, sac.ClusterScopeKey(metadata.GetClusterId())); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}
	return ds.storage.StoreFailure(metadata)
}
