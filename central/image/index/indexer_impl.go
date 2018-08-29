package index

import (
	"time"

	"github.com/blevesearch/bleve"
	"github.com/deckarep/golang-set"
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
func (b *indexerImpl) SearchImages(q *v1.Query) (results []search.Result, err error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "Search", "Image")

	// First, search deployments for all the image shas that match the deployment portion of the query.
	imageSHAsFromDeploymentQuery, filteredByDeployments, err := b.getImageSHAsFromDeploymentQuery(q)
	if err != nil {
		return nil, err
	}

	subQueryWithoutDeploymentFields, exists := search.FilterQuery(q, func(bq *v1.BaseQuery) bool {
		return !baseQueryMatchesOnDeploymentField(bq)
	})

	// If there are no fields other than deployment fields, then we want to get all images by passing an
	// empty query. However, if we have results from the deployment query anyway, then the final result
	// will be the deployment results (since we take an intersection), so we just return that.
	if !exists {
		if filteredByDeployments {
			return shasToResults(imageSHAsFromDeploymentQuery), nil
		}

		subQueryWithoutDeploymentFields = search.EmptyQuery()
	}

	// Create and run query for fields, and input string query, if it exists.
	results, err = blevesearch.RunSearchRequest(v1.SearchCategory_IMAGES, subQueryWithoutDeploymentFields, b.index, mappings.OptionsMap)
	if err != nil {
		return nil, err
	}

	// Filter results by which fields exist in the results retrieved from the deployments.
	if filteredByDeployments {
		filteredResults := results[:0]
		for _, result := range results {
			if imageSHAsFromDeploymentQuery.Contains(result.ID) {
				filteredResults = append(filteredResults, result)
			}
		}
		results = filteredResults
	}
	return
}

func baseQueryMatchesOnDeploymentField(bq *v1.BaseQuery) bool {
	matchFieldQuery, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
	if ok && mappings.OptionsMap[matchFieldQuery.MatchFieldQuery.GetField()].GetCategory() == v1.SearchCategory_DEPLOYMENTS {
		return true
	}
	return false
}

func (b *indexerImpl) getImageSHAsFromDeploymentQuery(q *v1.Query) (mapset.Set, bool, error) {
	deploymentSubQuery, found := search.FilterQuery(q, baseQueryMatchesOnDeploymentField)
	if !found {
		return nil, false, nil
	}

	query, err := blevesearch.BuildQuery(b.index, v1.SearchCategory_DEPLOYMENTS, deploymentSubQuery, mappings.OptionsMap)
	if err != nil {
		return nil, false, err
	}

	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Fields = []string{"deployment.containers.image.name.sha"}
	searchRequest.Size = maxDeploymentsReturned

	searchResult, err := b.index.Search(searchRequest)
	if err != nil {
		return nil, false, err
	}

	return deploymentResultsToShaSet(searchResult), true, nil
}

func shasToResults(shas mapset.Set) []search.Result {
	if shas == nil {
		return nil
	}

	searchResults := make([]search.Result, 0, shas.Cardinality())
	for sha := range shas.Iter() {
		searchResults = append(searchResults, search.Result{ID: sha.(string)})
	}
	return searchResults
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
