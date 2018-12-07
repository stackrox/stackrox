package index

import (
	"time"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/image/index/mappings"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/batcher"
	"github.com/stackrox/rox/pkg/images/types"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

const batchSize = 5000

type indexerImpl struct {
	index bleve.Index
}

type imageWrapper struct {
	*storage.Image `json:"image"`
	Type           string `json:"type"`
}

// AddImage adds the image to the index
func (i *indexerImpl) AddImage(image *storage.Image) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Add, "Image")
	digest := types.NewDigest(image.GetId()).Digest()
	return i.index.Index(digest, &imageWrapper{Type: v1.SearchCategory_IMAGES.String(), Image: image})
}

func (i *indexerImpl) processBatch(images []*storage.Image) error {
	batch := i.index.NewBatch()
	for _, image := range images {
		digest := types.NewDigest(image.GetId()).Digest()
		batch.Index(digest, &imageWrapper{Type: v1.SearchCategory_IMAGES.String(), Image: image})
	}
	return i.index.Batch(batch)
}

// AddImages adds the images to the index
func (i *indexerImpl) AddImages(images []*storage.Image) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.AddMany, "Image")
	batchManager := batcher.New(len(images), batchSize)
	for {
		start, end, ok := batchManager.Next()
		if !ok {
			break
		}
		if err := i.processBatch(images[start:end]); err != nil {
			return err
		}
	}
	return nil
}

// DeleteImage deletes the image from the index
func (i *indexerImpl) DeleteImage(sha string) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Remove, "Image")
	digest := types.NewDigest(sha).Digest()
	return i.index.Delete(digest)
}

// Search takes a SearchRequest and finds any matches
func (i *indexerImpl) Search(q *v1.Query) (results []search.Result, err error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Search, "Image")
	return blevesearch.RunSearchRequest(v1.SearchCategory_IMAGES, q, i.index, mappings.OptionsMap)
}
