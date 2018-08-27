package index

import (
	"time"

	"github.com/blevesearch/bleve"
	"github.com/deckarep/golang-set"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/image/index/mappings"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

// maxDeploymentsReturned is for the specific scoped request to deployments which is then joined with images
// so we need to return as many deployments as possible to not exclude relevant images
const maxDeploymentsReturned = 500

// AlertIndex provides storage functionality for alerts.
type indexerImpl struct {
	index bleve.Index
}

type imageWrapper struct {
	*v1.Image `json:"image"`
	Type      string `json:"type"`
}

// AddImage adds the image to the index
func (b *indexerImpl) AddImage(image *v1.Image) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "Add", "Image")
	digest := types.NewDigest(image.GetName().GetSha()).Digest()
	return b.index.Index(digest, &imageWrapper{Type: v1.SearchCategory_IMAGES.String(), Image: image})
}

// AddImages adds the images to the index
func (b *indexerImpl) AddImages(imageList []*v1.Image) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "AddBatch", "Image")

	batch := b.index.NewBatch()
	for _, image := range imageList {
		digest := types.NewDigest(image.GetName().GetSha()).Digest()
		batch.Index(digest, &imageWrapper{Type: v1.SearchCategory_IMAGES.String(), Image: image})
	}
	return b.index.Batch(batch)
}

// DeleteImage deletes the image from the index
func (b *indexerImpl) DeleteImage(sha string) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "Delete", "Image")
	digest := types.NewDigest(sha).Digest()
	return b.index.Delete(digest)
}

// SearchImages takes a SearchRequest and finds any matches
// It is different from other requests because it requires that we actually search the deployments
// for the image that may match the criteria
func (b *indexerImpl) SearchImages(request *v1.ParsedSearchRequest) (results []search.Result, err error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "Search", "Image")

	// If we have scopes set, or the request has nothing set, get the list of image SHAs from the deployments.
	// If no scopes are set, this returns ALL image SHAs present in deployments.
	var imageSHAs mapset.Set

	request, imageSHAs, filteredByDeployments, err := b.getImageSHAsFromDeploymentQuery(request)
	if err != nil {
		return nil, err
	}

	// If there is not query or query files, then we don't need to filter our image set.
	if len(request.GetFields()) == 0 && request.GetStringQuery() == "" {
		results, err = shasToResults(imageSHAs)
		return
	}

	// Create and run query for fields, and input string query, if it exists.
	imageQueries, err := blevesearch.FieldsToQuery(b.index, v1.SearchCategory_IMAGES, request, mappings.OptionsMap)
	if err != nil {
		return nil, err
	}
	if request.GetStringQuery() != "" {
		imageQueries = append(imageQueries, bleve.NewQueryStringQuery(request.GetStringQuery()))
	}
	results, err = blevesearch.RunQuery(bleve.NewConjunctionQuery(imageQueries...), b.index)
	if err != nil {
		return nil, err
	}

	// Filter results by which fields exist in the results retrieved from the deployments.
	if filteredByDeployments {
		filteredResults := results[:0]
		for _, result := range results {
			if imageSHAs.Contains(result.ID) {
				filteredResults = append(filteredResults, result)
			}
		}
		results = filteredResults
	}
	return
}

func (b *indexerImpl) getImageSHAsFromDeploymentQuery(request *v1.ParsedSearchRequest) (*v1.ParsedSearchRequest, mapset.Set, bool, error) {
	newRequest := proto.Clone(request).(*v1.ParsedSearchRequest)

	req := &v1.ParsedSearchRequest{
		Fields: map[string]*v1.ParsedSearchRequest_Values{},
	}

	if values, ok := request.Fields[search.Cluster]; ok {
		req.Fields[search.Cluster] = values
		delete(newRequest.Fields, search.Cluster)
	}
	if values, ok := request.Fields[search.Namespace]; ok {
		req.Fields[search.Namespace] = values
		delete(newRequest.Fields, search.Namespace)
	}
	if values, ok := request.Fields[search.LabelKey]; ok {
		req.Fields[search.LabelKey] = values
		delete(newRequest.Fields, search.LabelKey)
	}
	if values, ok := request.Fields[search.LabelValue]; ok {
		req.Fields[search.LabelValue] = values
		delete(newRequest.Fields, search.LabelValue)
	}

	if len(req.Fields) == 0 {
		return newRequest, nil, false, nil
	}

	query, err := blevesearch.BuildQuery(b.index, v1.SearchCategory_DEPLOYMENTS, req, mappings.OptionsMap)
	if err != nil {
		return newRequest, nil, false, err
	}

	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Fields = []string{"deployment.containers.image.name.sha"}
	searchRequest.Size = maxDeploymentsReturned

	searchResult, err := b.index.Search(searchRequest)
	if err != nil {
		return newRequest, nil, false, err
	}
	return newRequest, deploymentResultsToShaSet(searchResult), true, nil
}

func shasToResults(shas mapset.Set) ([]search.Result, error) {
	if shas == nil {
		return nil, nil
	}

	searchResults := make([]search.Result, 0, shas.Cardinality())
	for sha := range shas.Iter() {
		searchResults = append(searchResults, search.Result{ID: sha.(string)})
	}
	return searchResults, nil
}

func deploymentResultsToShaSet(searchResult *bleve.SearchResult) mapset.Set {
	shaSetFromDeployment := mapset.NewSet()
	for _, hit := range searchResult.Hits {
		shaObj := hit.Fields["deployment.containers.image.name.sha"]
		if shaObj == nil {
			continue
		}
		switch typedObj := shaObj.(type) {
		case []interface{}:
			for _, s := range typedObj {
				shaSetFromDeployment.Add(s.(string))
			}
		case string:
			shaSetFromDeployment.Add(typedObj)
		default:
			log.Errorf("Unexpected type %T for image sha", shaObj)
		}
	}
	return shaSetFromDeployment
}
