package datastore

import (
	"context"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/compliance"
	"github.com/stackrox/rox/central/compliance/datastore/internal/store"
	"github.com/stackrox/rox/central/compliance/datastore/types"
	"github.com/stackrox/rox/central/compliance/standards"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	complianceSAC = sac.ForResource(resources.Compliance)

	log = logging.LoggerForModule()

	// ErrNotAvailable is returned by the aggregation storage/retrieval methods when SAC is enabled.  Those methods cannot be used under SAC.
	ErrNotAvailable = errors.New("precomputed compliance is not available when SAC is enabled")
)

type datastoreImpl struct {
	storage store.Store
	filter  SacFilter

	storedAggregationMutex    sync.RWMutex
	aggregationSequenceNumber uint64
}

func (ds *datastoreImpl) QueryControlResults(ctx context.Context, query *v1.Query) ([]*storage.ComplianceControlResult, error) {
	// TODO(ROX-2575): this might need implementing.
	return nil, errors.New("not yet implemented")
}

func (ds *datastoreImpl) GetSpecificRunResults(ctx context.Context, clusterID, standardID, runID string, flags types.GetFlags) (types.ResultsWithStatus, error) {
	if !standards.IsSupported(standardID) {
		return types.ResultsWithStatus{}, standards.UnSupportedStandardsErr(standardID)
	}

	if ok, err := complianceSAC.ReadAllowed(ctx, sac.ClusterScopeKey(clusterID)); err != nil {
		return types.ResultsWithStatus{}, err
	} else if !ok {
		return types.ResultsWithStatus{}, errorhelpers.ErrNotFound
	}

	res, err := ds.storage.GetSpecificRunResults(clusterID, standardID, runID, flags)
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

func (ds *datastoreImpl) GetLatestRunResults(ctx context.Context, clusterID, standardID string, flags types.GetFlags) (types.ResultsWithStatus, error) {
	if !standards.IsSupported(standardID) {
		return types.ResultsWithStatus{}, standards.UnSupportedStandardsErr(standardID)
	}

	if ok, err := complianceSAC.ReadAllowed(ctx, sac.ClusterScopeKey(clusterID)); err != nil {
		return types.ResultsWithStatus{}, err
	} else if !ok {
		return types.ResultsWithStatus{}, errorhelpers.ErrNotFound
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
	standardIDs, unsupported := standards.FilterSupported(standardIDs)
	if len(unsupported) > 0 {
		return nil, standards.UnSupportedStandardsErr(unsupported...)
	}

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

func (ds *datastoreImpl) GetLatestRunResultsForClustersAndStandards(ctx context.Context, clusterIDs, standardIDs []string, flags types.GetFlags) (map[compliance.ClusterStandardPair]types.ResultsWithStatus, error) {
	results, err := ds.storage.GetLatestRunResultsByClusterAndStandard(clusterIDs, standardIDs, flags)
	if err != nil {
		return nil, err
	}
	filteredResults, err := ds.filter.FilterBatchResults(ctx, results)
	if err != nil {
		return nil, err
	}
	return filteredResults, err
}

func (ds *datastoreImpl) IsComplianceRunSuccessfulOnCluster(ctx context.Context, clusterID string, standardIDs []string) (bool, error) {
	standardIDs, unsupported := standards.FilterSupported(standardIDs)
	if len(unsupported) > 0 {
		return false, standards.UnSupportedStandardsErr(unsupported...)
	}

	if ok, err := complianceSAC.ReadAllowed(ctx, sac.ClusterScopeKey(clusterID)); err != nil {
		return false, err
	} else if !ok {
		return false, errors.Wrapf(errorhelpers.ErrNotFound, "ClusterID %s", clusterID)
	}
	results, err := ds.storage.GetLatestRunMetadataBatch(clusterID, standardIDs)
	if err != nil || len(results) == 0 {
		return false, err
	}
	for _, v := range results {
		if v.LastSuccessfulRunMetadata == nil {
			return false, nil
		}
	}
	return true, nil
}

func (ds *datastoreImpl) StoreRunResults(ctx context.Context, results *storage.ComplianceRunResults) error {
	if ok, err := complianceSAC.WriteAllowed(ctx, sac.ClusterScopeKey(results.GetDomain().GetCluster().GetId())); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	ds.storedAggregationMutex.Lock()
	defer ds.storedAggregationMutex.Unlock()

	defer func() {
		// Atomic because it will be atomically read outside the mutex
		atomic.AddUint64(&ds.aggregationSequenceNumber, 1)
	}()

	if err := ds.storage.ClearAggregationResults(); err != nil {
		log.Errorf("unable to clear old stored aggregations: %v", err)
	}

	return ds.storage.StoreRunResults(results)
}

func (ds *datastoreImpl) StoreFailure(ctx context.Context, metadata *storage.ComplianceRunMetadata) error {
	if ok, err := complianceSAC.WriteAllowed(ctx, sac.ClusterScopeKey(metadata.GetClusterId())); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	return ds.storage.StoreFailure(metadata)
}

func (ds *datastoreImpl) StoreComplianceDomain(ctx context.Context, domain *storage.ComplianceDomain) error {
	if ok, err := complianceSAC.WriteAllowed(ctx, sac.ClusterScopeKey(domain.GetCluster().GetId())); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	return ds.storage.StoreComplianceDomain(domain)
}

func (ds *datastoreImpl) PerformStoredAggregation(ctx context.Context, args *StoredAggregationArgs) ([]*storage.ComplianceAggregation_Result, []*storage.ComplianceAggregation_Source, map[*storage.ComplianceAggregation_Result]*storage.ComplianceDomain, error) {
	// TODO(ROX-9134): consider storing compliance results for Unrestricted scope
	if true {
		return args.AggregationFunc()
	}

	// Check for a pre-computed aggregation for this query
	results, sources, domainMap, err := ds.storage.GetAggregationResult(args.QueryString, args.GroupBy, args.Unit)
	if err != nil {
		// Log the error and continue.  We can skip this optimization and do the aggregation
		log.Errorf("error getting pre-computed compliance aggregation: %v", err)
	}
	if results != nil && sources != nil && domainMap != nil {
		return results, sources, domainMap, err
	}

	// Get the aggregation sequence number before performing the aggregation.  We don't need to be in a lock here.
	aggregationSequenceNumber := atomic.LoadUint64(&ds.aggregationSequenceNumber)
	// This performs the actual aggregation.  It must occur after getting the sequence number.
	results, sources, domainMap, err = args.AggregationFunc()
	if err != nil {
		return nil, nil, nil, err
	}

	// Store asynchronously so the API stays responsive
	go func() {
		// We do need to lock here so that the compliance data can't be changed between checking the sequence number and
		// storing the aggregation result
		ds.storedAggregationMutex.RLock()
		defer ds.storedAggregationMutex.RUnlock()
		curAggSeqNum := atomic.LoadUint64(&ds.aggregationSequenceNumber)
		// Storing aggregation results is only permitted if the compliance data hasn't changed
		if aggregationSequenceNumber != curAggSeqNum {
			return
		}

		err = ds.storage.StoreAggregationResult(args.QueryString, args.GroupBy, args.Unit, results, sources, domainMap)
		if err != nil {
			// Log the error and continue.  We can skip this optimization without issue
			log.Errorf("error storing compliance aggregation: %v", err)
		}
	}()

	return results, sources, domainMap, nil
}

func (ds *datastoreImpl) ClearAggregationResults(ctx context.Context) error {
	if ok, err := complianceSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	return ds.storage.ClearAggregationResults()
}
