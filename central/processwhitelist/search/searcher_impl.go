package search

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/processwhitelist/index"
	"github.com/stackrox/rox/central/processwhitelist/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/debug"
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
	if err != nil {
		return nil, err
	}
	whitelists := make([]*storage.ProcessWhitelist, 0, len(results))
	for _, result := range results {
		whitelist, err := s.storage.GetWhitelist(result.ID)
		if err != nil {
			return nil, errors.Wrapf(err, "retrieving process whitelist with id '%s'", result.ID)
		}
		// The result may not exist if the object was deleted after the search
		if whitelist == nil {
			continue
		}
		whitelists = append(whitelists, whitelist)
	}
	return whitelists, nil
}
