package postgres

import (
	"context"
	"time"

	metrics "github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	search "github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

// NewIndexer returns a new image indexer.
func NewIndexer(db postgres.DB) *indexerImpl {
	return &indexerImpl{
		db: db,
	}
}

type indexerImpl struct {
	db postgres.DB
}

func (b *indexerImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Count, "Image")

	return pgSearch.RunCountRequest(ctx, v1.SearchCategory_IMAGES, q, b.db)
}

func (b *indexerImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Search, "Image")

	return pgSearch.RunSearchRequest(ctx, v1.SearchCategory_IMAGES, q, b.db)
}

//// Stubs for satisfying interfaces

func (b *indexerImpl) AddImage(_ *storage.Image) error {
	return nil
}

func (b *indexerImpl) AddImages(_ []*storage.Image) error {
	return nil
}

func (b *indexerImpl) DeleteImage(_ string) error {
	return nil
}

func (b *indexerImpl) DeleteImages(_ []string) error {
	return nil
}

func (b *indexerImpl) MarkInitialIndexingComplete() error {
	return nil
}

func (b *indexerImpl) NeedsInitialIndexing() (bool, error) {
	return false, nil
}
