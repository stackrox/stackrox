package search

import (
	"bytes"
	"context"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/jackc/pgx/v4"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/processindicator/index"
	"github.com/stackrox/rox/central/processindicator/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/options/processindicators"
	"github.com/stackrox/rox/pkg/search/postgres"
	mappings "github.com/stackrox/rox/pkg/search/options/processindicators"
)

var (
	indicatorSACSearchHelper = sac.ForResource(resources.Indicator).MustCreateSearchHelper(processindicators.OptionsMap)
)

// searcherImpl provides an intermediary implementation layer for ProcessStorage.
type searcherImpl struct {
	storage store.Store
	indexer index.Indexer
}

// SearchRawIndicators retrieves Policies from the indexer and storage
func (s *searcherImpl) SearchRawProcessIndicators(ctx context.Context, q *v1.Query) ([]*storage.ProcessIndicator, error) {
	if features.PostgresPOC.Enabled() {
		defer metrics.SetIndexOperationDurationTime(time.Now(), ops.SearchAndGet, "ProcessIndicator")
		rows, err := postgres.RunSearchRequestValue(v1.SearchCategory_PROCESS_INDICATORS, q, globaldb.GetPostgresDB(), mappings.OptionsMap)
		if err != nil {
			if err == pgx.ErrNoRows {
				return nil, nil
			}
			return nil, err
		}
		defer rows.Close()
		var elems []*storage.ProcessIndicator
		for rows.Next() {
			var data []byte
			if err := rows.Scan(&data); err != nil {
				return nil, err
			}
			msg := new(storage.ProcessIndicator)
			buf := bytes.NewReader(data)
			t := time.Now()
			if err := jsonpb.Unmarshal(buf, msg); err != nil {
				return nil, err
			}
			metrics.SetJSONPBOperationDurationTime(t, "Unmarshal", "ProcessIndicator")
			elems = append(elems, msg)
		}
		return elems, nil
	}
	results, err := s.Search(ctx, q)
	if err != nil {
		return nil, err
	}
	processes, _, err := s.storage.GetMany(search.ResultsToIDs(results))
	return processes, err
}

func (s *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return indicatorSACSearchHelper.Apply(s.indexer.Search)(ctx, q)
}

// Count returns the number of search results from the query
func (s *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return indicatorSACSearchHelper.ApplyCount(s.indexer.Count)(ctx, q)
}
