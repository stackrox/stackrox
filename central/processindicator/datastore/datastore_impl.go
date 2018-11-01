package datastore

import (
	"fmt"

	"github.com/prometheus/common/log"
	"github.com/stackrox/rox/central/processindicator/index"
	"github.com/stackrox/rox/central/processindicator/search"
	"github.com/stackrox/rox/central/processindicator/store"
	"github.com/stackrox/rox/generated/api/v1"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

func (ds *datastoreImpl) SearchProcessIndicators(q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.searcher.SearchProcessIndicators(q)
}

func (ds *datastoreImpl) SearchRawProcessIndicators(q *v1.Query) ([]*v1.ProcessIndicator, error) {
	return ds.searcher.SearchRawProcessIndicators(q)
}

func (ds *datastoreImpl) GetProcessIndicator(id string) (*v1.ProcessIndicator, bool, error) {
	return ds.storage.GetProcessIndicator(id)
}

func (ds *datastoreImpl) GetProcessIndicators() ([]*v1.ProcessIndicator, error) {
	return ds.storage.GetProcessIndicators()
}

func (ds *datastoreImpl) AddProcessIndicators(indicators ...*v1.ProcessIndicator) error {
	removedIndicators, err := ds.storage.AddProcessIndicators(indicators...)
	if err != nil {
		return err
	}
	if len(removedIndicators) > 0 {
		if err := ds.indexer.DeleteProcessIndicators(removedIndicators...); err != nil {
			return err
		}
	}
	return ds.indexer.AddProcessIndicators(indicators)
}

func (ds *datastoreImpl) AddProcessIndicator(i *v1.ProcessIndicator) error {
	removedIndicator, err := ds.storage.AddProcessIndicator(i)
	if err != nil {
		return fmt.Errorf("adding indicator to bolt: %s", err)
	}
	if removedIndicator != "" {
		if err := ds.indexer.DeleteProcessIndicator(removedIndicator); err != nil {
			return fmt.Errorf("Error removing process indicator")
		}
	}
	if err := ds.indexer.AddProcessIndicator(i); err != nil {
		return fmt.Errorf("adding indicator to index: %s", err)
	}
	return nil
}

func (ds *datastoreImpl) RemoveProcessIndicator(id string) error {
	if err := ds.storage.RemoveProcessIndicator(id); err != nil {
		return err
	}
	return ds.indexer.DeleteProcessIndicator(id)
}

func (ds *datastoreImpl) RemoveProcessIndicatorsByDeployment(id string) error {
	query := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.DeploymentID, id).ProtoQuery()
	results, err := ds.SearchProcessIndicators(query)
	if err != nil {
		return err
	}
	idsToDelete := make([]string, 0, len(results))
	for _, r := range results {
		idsToDelete = append(idsToDelete, r.GetId())
	}

	for _, id := range idsToDelete {
		if err := ds.storage.RemoveProcessIndicator(id); err != nil {
			log.Warnf("Failed to remove process indicator %q: %v", id, err)
		}
	}
	return ds.indexer.DeleteProcessIndicators(idsToDelete...)
}
