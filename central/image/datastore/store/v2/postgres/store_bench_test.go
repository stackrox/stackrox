//go:build sql_integration

package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/require"
)

// BenchmarkWalkComparison benchmarks both Walk functions for comparison
func BenchmarkWalkComparison(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(b)

	store := New(testDB.DB, false, concurrency.NewKeyFence())

	// Setup: Insert test images
	numImages := 100
	images := make([]*storage.Image, 0, numImages)
	for i := 0; i < numImages; i++ {
		img := fixtures.GetImageWithUniqueComponents(5)
		img.Id = fmt.Sprintf("%d", i)
		images = append(images, img)
	}

	for _, image := range images {
		require.NoError(b, store.Upsert(ctx, image))
	}

	b.Run("WalkByQuery", func(b *testing.B) {
		for b.Loop() {
			count := 0
			err := store.WalkByQuery(ctx, search.EmptyQuery(), func(image *storage.Image) error {
				count++
				return nil
			})
			require.NoError(b, err)
		}
	})

	b.Run("WalkMetadataByQuery", func(b *testing.B) {
		for b.Loop() {
			count := 0
			err := store.WalkMetadataByQuery(ctx, search.EmptyQuery(), func(image *storage.Image) error {
				count++
				return nil
			})
			require.NoError(b, err)
		}
	})
}
