package index

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	search "github.com/stackrox/rox/pkg/search"
	blevesearch "github.com/stackrox/rox/pkg/search/blevesearch"
)

// Indexer is the image indexer.
//
//go:generate mockgen-wrapper
type Indexer interface {
	AddImage(image *storage.Image) error
	AddImages(images []*storage.Image) error
	DeleteImage(id string) error
	DeleteImages(ids []string) error
	Search(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error)
	Count(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) (int, error)
}
