package search

import (
	"context"

	"github.com/stackrox/rox/central/image/datastore/store"
	"github.com/stackrox/rox/central/image/index"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/search/scoped/postgres"
	"github.com/stackrox/rox/pkg/search/sortfields"
)

var (
	defaultSortOption = &v1.QuerySortOption{
		Field: search.LastUpdatedTime.String(),
	}
)

// NewV2 returns a new instance of Searcher for the given storage and indexer.
func NewV2(storage store.Store, indexer index.Indexer) Searcher {
	return &searcherImplV2{
		storage:  storage,
		indexer:  indexer,
		searcher: formatSearcherV2(indexer),
	}
}

func formatSearcherV2(searcher search.Searcher) search.Searcher {
	scopedSearcher := postgres.WithScoping(searcher)
	transformedSortFieldSearcher := sortfields.TransformSortFields(scopedSearcher, schema.ImagesSchema.OptionsMap)
	return paginated.WithDefaultSortOption(transformedSortFieldSearcher, defaultSortOption)
}

// searcherImplV2 provides an intermediary implementation layer for image storage.
type searcherImplV2 struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

// SearchImages retrieves SearchResults from the indexer and storage
func (s *searcherImplV2) SearchImages(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	images, results, err := s.searchImages(ctx, q)
	if err != nil {
		return nil, err
	}
	protoResults := make([]*v1.SearchResult, 0, len(images))
	for i, image := range images {
		protoResults = append(protoResults, convertImage(image, results[i]))
	}
	return protoResults, nil
}

func (s *searcherImplV2) SearchListImages(ctx context.Context, q *v1.Query) ([]*storage.ListImage, error) {
	images, _, err := s.searchImages(ctx, q)
	listImages := make([]*storage.ListImage, 0, len(images))
	for _, image := range images {
		listImages = append(listImages, types.ConvertImageToListImage(image))
	}
	return listImages, err
}

// SearchRawImages retrieves SearchResults from the indexer and storage
func (s *searcherImplV2) SearchRawImages(ctx context.Context, q *v1.Query) ([]*storage.Image, error) {
	results, err := s.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	images, _, err := s.storage.GetMany(ctx, search.ResultsToIDs(results))
	if err != nil {
		return nil, err
	}
	return images, nil
}

func (s *searcherImplV2) searchImages(ctx context.Context, q *v1.Query) ([]*storage.Image, []search.Result, error) {
	results, err := s.Search(ctx, q)
	if err != nil {
		return nil, nil, err
	}
	var images []*storage.Image
	var newResults []search.Result
	for _, result := range results {
		image, exists, err := s.storage.GetImageMetadata(ctx, result.ID)
		if err != nil {
			return nil, nil, err
		}
		// The result may not exist if the object was deleted after the search
		if !exists {
			continue
		}
		images = append(images, image)
		newResults = append(newResults, result)
	}
	return images, newResults, nil
}

func (s *searcherImplV2) Search(ctx context.Context, q *v1.Query) (res []search.Result, err error) {
	return s.searcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (s *searcherImplV2) Count(ctx context.Context, q *v1.Query) (count int, err error) {
	return s.searcher.Count(ctx, q)
}

// convertImage returns proto search result from a image object and the internal search result
func convertImage(image *storage.Image, result search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_IMAGES,
		Id:             types.NewDigest(image.GetId()).Digest(),
		Name:           image.GetName().GetFullName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
