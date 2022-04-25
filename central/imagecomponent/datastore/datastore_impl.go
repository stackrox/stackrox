package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/imagecomponent/index"
	sacFilters "github.com/stackrox/rox/central/imagecomponent/sac"
	"github.com/stackrox/rox/central/imagecomponent/search"
	"github.com/stackrox/rox/central/imagecomponent/store"
	"github.com/stackrox/rox/central/ranking"
	riskDataStore "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/filtered"
	"github.com/stackrox/rox/pkg/search/postgres"
)

var (
	log = logging.LoggerForModule()
	// TODO: Need to setup sac for Image Components correctly instead of relying on global access.
	imagesSAC = sac.ForResource(resources.Image)
	nodesSac  = sac.ForResource(resources.Node)
)

type imageComponentPks struct {
	name    string
	version string
	os      string
}

type datastoreImpl struct {
	storage       store.Store
	indexer       index.Indexer
	searcher      search.Searcher
	graphProvider graph.Provider

	risks                riskDataStore.DataStore
	imageComponentRanker *ranking.Ranker
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return ds.searcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.searcher.Count(ctx, q)
}

func (ds *datastoreImpl) SearchImageComponents(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.searcher.SearchImageComponents(ctx, q)
}

func (ds *datastoreImpl) SearchRawImageComponents(ctx context.Context, q *v1.Query) ([]*storage.ImageComponent, error) {
	components, err := ds.searcher.SearchRawImageComponents(ctx, q)
	if err != nil {
		return nil, err
	}
	ds.updateImageComponentPriority(components...)
	return components, nil
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.ImageComponent, bool, error) {
	filteredIDs, err := ds.filterReadable(ctx, []string{id})
	if err != nil || len(filteredIDs) != 1 {
		return nil, false, err
	}

	var pks imageComponentPks
	if features.PostgresDatastore.Enabled() {
		pks, err = getPKs(id)
		if err != nil {
			return nil, false, err
		}
	}
	// For dackbox, we do not need all the primary keys.

	component, found, err := ds.storage.Get(ctx, id, pks.name, pks.version, pks.os)
	if err != nil || !found {
		return nil, false, err
	}

	ds.updateImageComponentPriority(component)
	return component, true, nil
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	filteredIDs, err := ds.filterReadable(ctx, []string{id})
	if err != nil || len(filteredIDs) != 1 {
		return false, err
	}

	var pks imageComponentPks
	if features.PostgresDatastore.Enabled() {
		pks, err = getPKs(id)
		if err != nil {
			return false, err
		}
	}
	// For dackbox, we do not need all the primary keys.

	found, err := ds.storage.Exists(ctx, id, pks.name, pks.version, pks.os)
	if err != nil || !found {
		return false, err
	}
	return true, nil
}

func (ds *datastoreImpl) GetBatch(ctx context.Context, ids []string) ([]*storage.ImageComponent, error) {
	filteredIDs, err := ds.filterReadable(ctx, ids)
	if err != nil {
		return nil, err
	}

	components, _, err := ds.storage.GetMany(ctx, filteredIDs)
	if err != nil {
		return nil, err
	}

	ds.updateImageComponentPriority(components...)
	return components, nil
}

func (ds *datastoreImpl) initializeRankers() {
	readCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS), sac.ResourceScopeKeys(resources.Image, resources.Node)))

	results, err := ds.Search(readCtx, pkgSearch.EmptyQuery())
	if err != nil {
		log.Error(err)
		return
	}

	for _, id := range pkgSearch.ResultsToIDs(results) {
		var pks imageComponentPks
		if features.PostgresDatastore.Enabled() {
			pks, err = getPKs(id)
			if err != nil {
				log.Error(err)
				continue
			}
		}
		// For dackbox, we do not need all the primary keys.

		component, found, err := ds.storage.Get(readCtx, id, pks.name, pks.version, pks.os)
		if err != nil {
			log.Error(err)
			continue
		} else if !found {
			continue
		}
		ds.imageComponentRanker.Add(id, component.GetRiskScore())
	}
}

func (ds *datastoreImpl) updateImageComponentPriority(ics ...*storage.ImageComponent) {
	for _, ic := range ics {
		ic.Priority = ds.imageComponentRanker.GetRankForID(ic.GetId())
	}
}

func (ds *datastoreImpl) filterReadable(ctx context.Context, ids []string) ([]string, error) {
	var filteredIDs []string
	var err error
	graph.Context(ctx, ds.graphProvider, func(graphContext context.Context) {
		filteredIDs, err = filtered.ApplySACFilter(graphContext, ids, sacFilters.GetSACFilter())
	})
	return filteredIDs, err
}

func getPKs(id string) (imageComponentPks, error) {
	parts := postgres.IDToParts(id)
	if len(parts) != 3 {
		return imageComponentPks{}, errors.Errorf("unexpected number of primary keys (%v) found for image component. Expected 3 parts", parts)
	}

	return imageComponentPks{
		name:    parts[0],
		version: parts[1],
		os:      parts[2],
	}, nil
}
