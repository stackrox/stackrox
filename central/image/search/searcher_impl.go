package search

import (
	"bitbucket.org/stack-rox/apollo/central/image/index"
	"bitbucket.org/stack-rox/apollo/central/image/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/images"
	"bitbucket.org/stack-rox/apollo/pkg/search"
)

// searcherImpl provides an intermediary implementation layer for AlertStorage.
type searcherImpl struct {
	storage store.Store
	indexer index.Indexer
}

func (ds *searcherImpl) buildIndex() error {
	images, err := ds.storage.GetImages()
	if err != nil {
		return err
	}
	return ds.indexer.AddImages(images)
}

// SearchImages retrieves SearchResults from the indexer and storage
func (ds *searcherImpl) SearchImages(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	images, results, err := ds.searchImages(request)
	if err != nil {
		return nil, err
	}
	protoResults := make([]*v1.SearchResult, 0, len(images))
	for i, image := range images {
		protoResults = append(protoResults, convertImage(image, results[i]))
	}
	return protoResults, nil
}

// SearchRawImages retrieves SearchResults from the indexer and storage
func (ds *searcherImpl) SearchRawImages(request *v1.ParsedSearchRequest) ([]*v1.Image, error) {
	images, _, err := ds.searchImages(request)
	return images, err
}

func (ds *searcherImpl) searchImages(request *v1.ParsedSearchRequest) ([]*v1.Image, []search.Result, error) {
	results, err := ds.indexer.SearchImages(request)
	if err != nil {
		return nil, nil, err
	}
	var images []*v1.Image
	var newResults []search.Result
	for _, result := range results {
		image, exists, err := ds.storage.GetImage(result.ID)
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

// ConvertImage returns proto search result from a image object and the internal search result
func convertImage(image *v1.Image, result search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_IMAGES,
		Id:             images.NewDigest(image.GetName().GetSha()).Digest(),
		Name:           image.GetName().GetFullName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
