package search

import (
	"github.com/stackrox/rox/central/processwhitelist/index"
	"github.com/stackrox/rox/central/processwhitelist/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/debug"
	"github.com/stackrox/rox/pkg/search"
)

type searcherImpl struct {
	storage store.Store
	indexer index.Indexer
}

func (s *searcherImpl) buildIndex() error {
	defer debug.FreeOSMemory()
	whitelists, err := s.storage.ListWhitelists()
	if err != nil {
		return err
	}
	return s.indexer.AddWhitelists(whitelists)
}

func (s *searcherImpl) SearchRawProcessWhitelists(q *v1.Query) ([]*storage.ProcessWhitelist, error) {
	results, err := s.indexer.Search(q)
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
