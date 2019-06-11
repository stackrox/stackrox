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
)

var (
	processWhitelistSACSearchHelper = sac.ForResource(resources.Alert).MustCreateSearchHelper(mappings.OptionsMap, sac.ClusterIDAndNamespaceFields)
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
