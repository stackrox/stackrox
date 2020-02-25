package search

import (
	"context"

	"github.com/stackrox/rox/central/processwhitelist/index"
	"github.com/stackrox/rox/central/processwhitelist/index/mappings"
	"github.com/stackrox/rox/central/processwhitelist/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/debug"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/search/paginated"
)

const (
	whitelistBatchLimit = 10000
)

var (
	processWhitelistSACSearchHelper = sac.ForResource(resources.ProcessWhitelist).MustCreateSearchHelper(mappings.OptionsMap)
)

type searcherImpl struct {
	storage           store.Store
	indexer           index.Indexer
	formattedSearcher search.Searcher
}

func (s *searcherImpl) buildIndex() error {
	defer debug.FreeOSMemory()
	whitelists := make([]*storage.ProcessWhitelist, 0, whitelistBatchLimit)
	if err := s.storage.WalkAll(func(whitelist *storage.ProcessWhitelist) error {
		whitelists = append(whitelists, whitelist)
		if len(whitelists) == whitelistBatchLimit {
			if err := s.indexer.AddWhitelists(whitelists); err != nil {
				return err
			}

			whitelists = whitelists[:0]
		}

		return nil
	}); err != nil {
		return err
	}

	if len(whitelists) > 0 {
		return s.indexer.AddWhitelists(whitelists)
	}

	return nil
}

func (s *searcherImpl) SearchRawProcessWhitelists(ctx context.Context, q *v1.Query) ([]*storage.ProcessWhitelist, error) {
	results, err := processWhitelistSACSearchHelper.Apply(s.indexer.Search)(ctx, q)
	if err != nil || len(results) == 0 {
		return nil, err
	}
	ids := search.ResultsToIDs(results)
	whitelists, _, err := s.storage.GetWhitelists(ids)
	if err != nil {
		return nil, err
	}
	return whitelists, nil
}

func (s *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return s.formattedSearcher.Search(ctx, q)
}

// Helper functions which format our searching.
///////////////////////////////////////////////

func formatSearcher(unsafeSearcher blevesearch.UnsafeSearcher) search.Searcher {
	filteredSearcher := processWhitelistSACSearchHelper.FilteredSearcher(unsafeSearcher) // Make the UnsafeSearcher safe.
	paginatedSearcher := paginated.Paginated(filteredSearcher)
	return paginatedSearcher
}
