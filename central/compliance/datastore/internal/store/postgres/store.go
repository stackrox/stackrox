package postgres

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/compliance"
	"github.com/stackrox/rox/central/compliance/datastore/internal/store"
	configStore "github.com/stackrox/rox/central/compliance/datastore/internal/store/postgres/compliance_config"
	domainStore "github.com/stackrox/rox/central/compliance/datastore/internal/store/postgres/domain"
	metadataStore "github.com/stackrox/rox/central/compliance/datastore/internal/store/postgres/metadata"
	resultsStore "github.com/stackrox/rox/central/compliance/datastore/internal/store/postgres/results"
	stringsStore "github.com/stackrox/rox/central/compliance/datastore/internal/store/postgres/strings"
	"github.com/stackrox/rox/central/compliance/datastore/types"
	dsTypes "github.com/stackrox/rox/central/compliance/datastore/types"
	"github.com/stackrox/rox/central/globaldb"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()

	domainCacheExpiry = 30 * time.Second
	cacheLock         = concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize)
	domainCache       = expiringcache.NewExpiringCache(domainCacheExpiry, expiringcache.UpdateExpirationOnGets)
)

type metadataIndex interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
}

// NewStore returns a compliance store based on Postgres
func NewStore(db postgres.DB) store.Store {
	return &storeImpl{
		domain:        domainStore.New(db),
		metadata:      metadataStore.New(db),
		metadataIndex: metadataStore.NewIndexer(db),
		results:       resultsStore.New(db),
		strings:       stringsStore.New(db),
		config:        configStore.New(db),
	}
}

type storeImpl struct {
	domain        domainStore.Store
	metadata      metadataStore.Store
	metadataIndex metadataIndex
	results       resultsStore.Store
	strings       stringsStore.Store
	config        configStore.Store
}

func (s *storeImpl) getDomain(ctx context.Context, domainID string) (*storage.ComplianceDomain, error) {
	cacheLock.Lock(domainID)
	defer cacheLock.Unlock(domainID)

	cachedDomain := domainCache.Get(domainID)
	if cachedDomain != nil {
		return cachedDomain.(*storage.ComplianceDomain), nil
	}

	domain, exists, err := s.domain.Get(ctx, domainID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errox.NotFound.Newf("domain with id %q was not found", domainID)
	}
	domainCache.Add(domainID, domain)
	return domain, nil
}

func (s *storeImpl) UpdateConfig(ctx context.Context, config *storage.ComplianceConfig) error {
	return s.config.Upsert(ctx, config)
}

func (s *storeImpl) GetConfig(ctx context.Context, id string) (*storage.ComplianceConfig, bool, error) {
	return s.config.Get(ctx, id)
}

func (s *storeImpl) getResultsFromMetadata(ctx context.Context, metadata *storage.ComplianceRunMetadata, flags types.GetFlags) (*storage.ComplianceRunResults, error) {
	domain, err := s.getDomain(ctx, metadata.GetDomainId())
	if err != nil {
		return nil, err
	}
	results, exists, err := s.results.Get(ctx, metadata.GetRunId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errox.NotFound.Newf("run results with id %q was not found", metadata.GetRunId())
	}

	results.RunMetadata = metadata
	results.Domain = domain

	if flags&(dsTypes.WithMessageStrings|dsTypes.RequireMessageStrings) != 0 {
		externalizedStrings, exists, err := s.strings.Get(ctx, metadata.GetRunId())
		if err != nil || !exists {
			if flags&dsTypes.RequireMessageStrings != 0 {
				return nil, errors.Wrap(err, "loading message strings")
			}
			log.Errorf("Could not load message strings for compliance run results: %v", err)
		}
		if !store.ReconstituteStrings(results, externalizedStrings) {
			return nil, errors.New("some message strings could not be loaded")
		}
	}
	return results, nil
}

func (s *storeImpl) GetSpecificRunResults(ctx context.Context, _, _, runID string, flags types.GetFlags) (types.ResultsWithStatus, error) {
	metadata, exists, err := s.metadata.Get(ctx, runID)
	if err != nil {
		return types.ResultsWithStatus{}, err
	}
	if !exists {
		return types.ResultsWithStatus{}, errox.NotFound.Newf("run metadata with id %q was not found", runID)
	}
	if !metadata.GetSuccess() {
		return types.ResultsWithStatus{
			FailedRuns: []*storage.ComplianceRunMetadata{
				metadata,
			},
		}, nil
	}
	results, err := s.getResultsFromMetadata(ctx, metadata, flags)
	if err != nil {
		return types.ResultsWithStatus{}, err
	}
	return types.ResultsWithStatus{
		LastSuccessfulResults: results,
	}, nil
}

func (s *storeImpl) GetLatestRunResults(ctx context.Context, clusterID, standardID string, flags types.GetFlags) (types.ResultsWithStatus, error) {
	metadataBatch, err := s.getLatestRunMetadata(ctx, clusterID, standardID)
	if err != nil {
		return types.ResultsWithStatus{}, err
	}
	if metadataBatch.LastSuccessfulRunMetadata == nil {
		return types.ResultsWithStatus{
			FailedRuns: metadataBatch.FailedRunsMetadata,
		}, nil
	}
	results, err := s.getResultsFromMetadata(ctx, metadataBatch.LastSuccessfulRunMetadata, flags)
	if err != nil {
		return types.ResultsWithStatus{}, err
	}
	return types.ResultsWithStatus{
		LastSuccessfulResults: results,
		FailedRuns:            metadataBatch.FailedRunsMetadata,
	}, nil
}

func (s *storeImpl) GetLatestRunResultsBatch(ctx context.Context, clusterIDs, standardIDs []string, flags types.GetFlags) (map[compliance.ClusterStandardPair]types.ResultsWithStatus, error) {
	pairsToResults := make(map[compliance.ClusterStandardPair]types.ResultsWithStatus)
	for _, clusterID := range clusterIDs {
		for _, standardID := range standardIDs {
			results, err := s.GetLatestRunResults(ctx, clusterID, standardID, flags)
			if err != nil {
				return nil, err
			}
			pairsToResults[compliance.NewPair(clusterID, standardID)] = results
		}
	}
	return pairsToResults, nil
}

func (s *storeImpl) getLatestRunMetadata(ctx context.Context, clusterID, standardID string) (types.ComplianceRunsMetadata, error) {
	query := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusterID).
		AddExactMatches(search.StandardID, standardID).
		WithPagination(
			search.NewPagination().
				Limit(10).
				AddSortOption(search.NewSortOption(search.ComplianceRunFinishedTimestamp).Reversed(true)),
		).
		ProtoQuery()
	metadataSearchResults, err := s.metadataIndex.Search(ctx, query)
	if err != nil {
		return types.ComplianceRunsMetadata{}, err
	}
	metadatas, _, err := s.metadata.GetMany(ctx, search.ResultsToIDs(metadataSearchResults))
	if err != nil {
		return types.ComplianceRunsMetadata{}, err
	}

	resultsValue := types.ComplianceRunsMetadata{}
	for _, metadata := range metadatas {
		if metadata.GetSuccess() {
			resultsValue.LastSuccessfulRunMetadata = metadata
			break
		}
		resultsValue.FailedRunsMetadata = append(resultsValue.FailedRunsMetadata, metadata)
	}
	return resultsValue, nil
}

func (s *storeImpl) GetLatestRunMetadataBatch(ctx context.Context, clusterID string, standardIDs []string) (map[compliance.ClusterStandardPair]types.ComplianceRunsMetadata, error) {
	results := make(map[compliance.ClusterStandardPair]types.ComplianceRunsMetadata)
	for _, standardID := range standardIDs {
		metadata, err := s.getLatestRunMetadata(ctx, clusterID, standardID)
		if err != nil {
			return nil, err
		}
		results[compliance.NewPair(clusterID, standardID)] = metadata
	}
	return results, nil
}

func (s *storeImpl) StoreRunResults(ctx context.Context, results *storage.ComplianceRunResults) error {
	// Domain is stored separately
	results.Domain = nil
	if err := s.metadata.Upsert(ctx, results.GetRunMetadata()); err != nil {
		return err
	}
	externalizedStrings := store.ExternalizeStrings(results)
	if err := s.strings.Upsert(ctx, externalizedStrings); err != nil {
		return err
	}
	return s.results.Upsert(ctx, results)
}

func (s *storeImpl) StoreFailure(ctx context.Context, metadata *storage.ComplianceRunMetadata) error {
	return s.metadata.Upsert(ctx, metadata)
}

func (s *storeImpl) StoreComplianceDomain(ctx context.Context, domain *storage.ComplianceDomain) error {
	if err := s.domain.Upsert(ctx, domain); err != nil {
		return err
	}
	cacheLock.Lock(domain.GetId())
	defer cacheLock.Unlock(domain.GetId())

	domainCache.Add(domain.GetId(), domain)
	return nil
}
