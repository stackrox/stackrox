package blevesearch

import (
	"reflect"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/images"
	"bitbucket.org/stack-rox/apollo/pkg/set"
	"github.com/blevesearch/bleve"
	"github.com/deckarep/golang-set"
)

var imageObjectMap = map[string]string{
	"image": "",
}

// AddImage adds the image to the index
func (b *Indexer) AddImage(image *v1.Image) error {
	digest := images.NewDigest(image.GetName().GetSha()).Digest()
	return b.imageIndex.Index(digest, image)
}

// DeleteImage deletes the image from the index
func (b *Indexer) DeleteImage(sha string) error {
	return b.imageIndex.Delete(sha)
}

func (b *Indexer) getImageSHAsFromScope(request *v1.SearchRequest) (mapset.Set, error) {
	if scopesQuery := getScopesQuery(request.GetScopes(), scopeToDeploymentQuery); scopesQuery != nil {
		searchRequest := bleve.NewSearchRequest(scopesQuery)
		searchRequest.Fields = []string{"containers.image.name.sha"}
		searchResult, err := b.deploymentIndex.Search(searchRequest)
		if err != nil {
			return nil, err
		}
		shaSetFromDeployment := mapset.NewSet()
		for _, hit := range searchResult.Hits {
			shaObj := hit.Fields["containers.image.name.sha"]
			t := reflect.TypeOf(shaObj)
			kind := t.Kind()
			switch kind {
			case reflect.Slice:
				strSlice, ok := shaObj.([]interface{})
				if !ok {
					logger.Errorf("Unexpected Slice type %s for image sha", t)
					continue
				}
				for _, s := range strSlice {
					shaSetFromDeployment.Add(s.(string))
				}
			case reflect.String:
				shaSetFromDeployment.Add(shaObj.(string))
			default:
				logger.Errorf("Unexpected type %s for image sha", t)
			}
		}
		return shaSetFromDeployment, nil
	}
	return nil, nil
}

// SearchImages takes a SearchRequest and finds any matches
// It is different from other requests because it requires that we actually search the deployments
// for the image that may match the criteria
func (b *Indexer) SearchImages(request *v1.SearchRequest) ([]string, error) {
	shaSetFromDeployment, err := b.getImageSHAsFromScope(request)
	if err != nil {
		return nil, err
	}
	// If there is only scope defined, then we should return all images
	if len(request.GetFields()) == 0 {
		return set.StringSliceFromSet(shaSetFromDeployment), nil
	}

	imageQuery := fieldsToQuery(request.GetFields(), imageObjectMap)
	results, err := runQuery(imageQuery, b.imageIndex)
	if err != nil {
		return nil, err
	}
	// If we retrieve images from scope, then find the intersection of the results
	if shaSetFromDeployment != nil {
		imageSet := set.NewSetFromStringSlice(results)
		return set.StringSliceFromSet(imageSet.Intersect(shaSetFromDeployment)), nil
	}
	return results, nil
}
