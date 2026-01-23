package datastore

import (
	"context"
	"strings"

	pgStore "github.com/stackrox/rox/central/imagecomponent/v2/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	riskDataStore "github.com/stackrox/rox/central/risk/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

type datastoreImpl struct {
	storage pgStore.Store

	risks                riskDataStore.DataStore
	imageComponentRanker *ranking.Ranker
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return ds.storage.Search(ctx, q)
}

// Count returns the number of search results from the query
func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.storage.Count(ctx, q)
}

func (ds *datastoreImpl) SearchImageComponents(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	// Clone the query and ensure it includes Component in select fields to populate result names
	if q == nil {
		q = pkgSearch.EmptyQuery()
	}

	qClone := q.CloneVT()

	qClone.Selects = append(qClone.GetSelects(), pkgSearch.NewQuerySelect(pkgSearch.Component).Proto())

	results, err := ds.Search(ctx, qClone)
	if err != nil {
		return nil, err
	}

	searchTag := strings.ToLower(pkgSearch.Component.String())
	for i := range results {
		if results[i].FieldValues != nil {
			if nameVal, ok := results[i].FieldValues[searchTag]; ok {
				results[i].Name = nameVal
			}
		}
	}

	return pkgSearch.ResultsToSearchResultProtos(results, &ImageComponentSearchResultConverter{}), nil
}

func (ds *datastoreImpl) SearchRawImageComponents(ctx context.Context, q *v1.Query) ([]*storage.ImageComponentV2, error) {
	var components []*storage.ImageComponentV2
	err := ds.storage.GetByQueryFn(ctx, q, func(component *storage.ImageComponentV2) error {
		components = append(components, component)
		return nil
	})
	if err != nil {
		return nil, err
	}

	ds.updateImageComponentPriority(components...)
	return components, nil
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.ImageComponentV2, bool, error) {
	component, found, err := ds.storage.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}

	ds.updateImageComponentPriority(component)
	return component, true, nil
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	found, err := ds.storage.Exists(ctx, id)
	if err != nil || !found {
		return false, err
	}
	return true, nil
}

func (ds *datastoreImpl) GetBatch(ctx context.Context, ids []string) ([]*storage.ImageComponentV2, error) {
	components, _, err := ds.storage.GetMany(ctx, ids)
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

	err := ds.storage.Walk(readCtx, func(component *storage.ImageComponentV2) error {
		ds.imageComponentRanker.Add(component.GetId(), component.GetRiskScore())
		return nil
	})
	if err != nil {
		log.Errorf("unable to initialize image component ranking: %v", err)
		return
	}

	log.Info("Initialized image component ranking")
}

func (ds *datastoreImpl) updateImageComponentPriority(ics ...*storage.ImageComponentV2) {
	for _, ic := range ics {
		ic.Priority = ds.imageComponentRanker.GetRankForID(ic.GetId())
	}
}

// ImageComponentSearchResultConverter implements search.SearchResultConverter for image component search results.
type ImageComponentSearchResultConverter struct{}

func (c *ImageComponentSearchResultConverter) BuildName(result *pkgSearch.Result) string {
	return result.Name
}

func (c *ImageComponentSearchResultConverter) BuildLocation(result *pkgSearch.Result) string {
	// Image components do not have a location
	return ""
}

func (c *ImageComponentSearchResultConverter) GetCategory() v1.SearchCategory {
	return v1.SearchCategory_IMAGE_COMPONENTS_V2
}

func (c *ImageComponentSearchResultConverter) GetScore(result *pkgSearch.Result) float64 {
	return result.Score
}
