package index

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	search "github.com/stackrox/rox/pkg/search"
	blevesearch "github.com/stackrox/rox/pkg/search/blevesearch"
)

// Indexer is the image-cve edge indexer.
//
//go:generate mockgen-wrapper
type Indexer interface {
	AddImageCVEEdge(imagecveedge *storage.ImageCVEEdge) error
	AddImageCVEEdges(imagecveedges []*storage.ImageCVEEdge) error
	Count(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) (int, error)
	DeleteImageCVEEdge(id string) error
	DeleteImageCVEEdges(ids []string) error
	Search(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error)
}
