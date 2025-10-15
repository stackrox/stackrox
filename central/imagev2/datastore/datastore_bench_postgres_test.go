//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/image/datastore/keyfence"
	pgStore "github.com/stackrox/rox/central/imagev2/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func BenchmarkImageGetMany(b *testing.B) {
	if !features.FlattenImageData.Enabled() {
		b.Skip("Image flattened data model is not enabled")
	}
	ctx := sac.WithAllAccess(context.Background())

	testDB := pgtest.ForT(b)

	db := testDB.DB

	mockRisk := mockRisks.NewMockDataStore(gomock.NewController(b))
	datastore := NewWithPostgres(pgStore.New(db, false, keyfence.ImageKeyFenceSingleton()), mockRisk, ranking.NewRanker(), ranking.NewRanker())

	ids := make([]string, 0, 100)
	images := make([]*storage.ImageV2, 0, 100)
	for i := 0; i < 100; i++ {
		img := fixtures.GetImageV2WithUniqueComponents(5)
		id := fmt.Sprintf("%d", i)
		ids = append(ids, id)
		img.Id = id
		images = append(images, img)
	}

	for _, image := range images {
		require.NoError(b, datastore.UpsertImage(ctx, image))
	}

	b.Run("GetImagesBatch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := datastore.GetImagesBatch(ctx, ids)
			require.NoError(b, err)
		}
	})

	b.Run("GetManyImageMetadata", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := datastore.GetManyImageMetadata(ctx, ids)
			require.NoError(b, err)
		}
	})
}

func BenchmarkImageUpsert(b *testing.B) {
	if !features.FlattenImageData.Enabled() {
		b.Skip("Image flattened data model is not enabled")
	}
	ctx := sac.WithAllAccess(context.Background())

	testDB := pgtest.ForT(b)

	db := testDB.DB

	mockRisk := mockRisks.NewMockDataStore(gomock.NewController(b))
	datastore := NewWithPostgres(pgStore.New(db, false, keyfence.ImageKeyFenceSingleton()), mockRisk, ranking.NewRanker(), ranking.NewRanker())

	images := make([]*storage.ImageV2, 0, 100)
	for i := 0; i < 100; i++ {
		img := fixtures.GetImageV2WithUniqueComponents(5)
		img.Id = uuid.NewV4().String()
		images = append(images, img)
	}

	b.Run("UpsertImage", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, img := range images {
				require.NoError(b, datastore.UpsertImage(ctx, img))
			}
		}
	})
}
