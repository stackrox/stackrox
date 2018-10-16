package datastore

import (
	"fmt"

	"github.com/stackrox/rox/central/processindicator/index"
	"github.com/stackrox/rox/central/processindicator/search"
	"github.com/stackrox/rox/central/processindicator/store"
	"github.com/stackrox/rox/generated/api/v1"
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

func (ds *datastoreImpl) AddProcessIndicator(i *v1.ProcessIndicator) error {
	if err := ds.storage.AddProcessIndicator(i); err != nil {
		return fmt.Errorf("adding indicator to bolt: %s", err)
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
