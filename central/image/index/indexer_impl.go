package index

import (
	"time"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/image/index/mappings"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

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
func (b *indexerImpl) SearchImages(q *v1.Query) (results []search.Result, err error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), "Search", "Image")
	return blevesearch.RunSearchRequest(v1.SearchCategory_IMAGES, q, b.index, mappings.OptionsMap)
}
