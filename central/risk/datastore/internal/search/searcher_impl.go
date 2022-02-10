package search

import (
	"bytes"
	"context"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/jackc/pgx/v4"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/risk/datastore/internal/index"
	"github.com/stackrox/rox/central/risk/datastore/internal/store"
	"github.com/stackrox/rox/central/risk/mappings"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/search/postgres"
)

var (
	defaultSortOption = &v1.QuerySortOption{
		Field:    search.RiskScore.String(),
		Reversed: true,
	}

	riskSACSearchHelper = sac.ForResource(resources.Risk).MustCreateSearchHelper(mappings.OptionsMap)
)

// searcherImpl provides an intermediary implementation layer for RiskStorage.
type searcherImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

// SearchRawRisk retrieves Risks from the indexer and storage
func (s *searcherImpl) SearchRawRisks(ctx context.Context, q *v1.Query) ([]*storage.Risk, error) {
	if features.PostgresPOC.Enabled() {
		defer metrics.SetIndexOperationDurationTime(time.Now(), ops.SearchAndGet, "Risk")
		rows, err := postgres.RunSearchRequestValue(v1.SearchCategory_RISKS, q, globaldb.GetPostgresDB(), mappings.OptionsMap)
		if err != nil {
			if err == pgx.ErrNoRows {
				return nil, nil
			}
			return nil, err
		}
		defer rows.Close()
		var elems []*storage.Risk
		for rows.Next() {
			var id string
			var data []byte
			if err := rows.Scan(&id, &data); err != nil {
				return nil, err
			}
			msg := new(storage.Risk)
			buf := bytes.NewReader(data)
			t := time.Now()
			if err := jsonpb.Unmarshal(buf, msg); err != nil {
				return nil, err
			}
			metrics.SetJSONPBOperationDurationTime(t, "Unmarshal", "Risk")
			elems = append(elems, msg)
		}
		return elems, nil
	}
	results, err := s.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	ids := search.ResultsToIDs(results)
	risks, _, err := s.storage.GetMany(ids)
	if err != nil {
		return nil, err
	}
	return risks, nil
}

func (s *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return s.searcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (s *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return s.searcher.Count(ctx, q)
}

// Format the search functionality of the indexer to be filtered (for sac) and paginated.
func formatSearcher(unsafeSearcher blevesearch.UnsafeSearcher) search.Searcher {
	filteredSearcher := riskSACSearchHelper.FilteredSearcher(unsafeSearcher) // Make the UnsafeSearcher safe.
	paginatedSearcher := paginated.Paginated(filteredSearcher)
	defaultSortedSearcher := paginated.WithDefaultSortOption(paginatedSearcher, defaultSortOption)
	return defaultSortedSearcher
}
