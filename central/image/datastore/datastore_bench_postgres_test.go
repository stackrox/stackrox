//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/image/datastore/keyfence"
	pgStore "github.com/stackrox/rox/central/image/datastore/store/postgres"
	pgStoreV2 "github.com/stackrox/rox/central/image/datastore/store/v2/postgres"
	"github.com/stackrox/rox/central/ranking"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func BenchmarkImageGetMany(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())

	testDB := pgtest.ForT(b)

	gormDB := testDB.GetGormDB(b)
	db := testDB.DB

	mockRisk := mockRisks.NewMockDataStore(gomock.NewController(b))
	var datastore DataStore
	if features.FlattenCVEData.Enabled() {
		datastore = NewWithPostgres(pgStoreV2.New(db, false, keyfence.ImageKeyFenceSingleton()), mockRisk, ranking.NewRanker(), ranking.NewRanker())
	} else {
		datastore = NewWithPostgres(pgStore.CreateTableAndNewStore(ctx, db, gormDB, false), mockRisk, ranking.NewRanker(), ranking.NewRanker())
	}

	ids := make([]string, 0, 100)
	images := make([]*storage.Image, 0, 100)
	for i := 0; i < 100; i++ {
		img := fixtures.GetImageWithUniqueComponents(5)
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
	ctx := sac.WithAllAccess(context.Background())

	testDB := pgtest.ForT(b)

	gormDB := testDB.GetGormDB(b)
	db := testDB.DB

	mockRisk := mockRisks.NewMockDataStore(gomock.NewController(b))
	var datastore DataStore
	if features.FlattenCVEData.Enabled() {
		datastore = NewWithPostgres(pgStoreV2.New(db, false, keyfence.ImageKeyFenceSingleton()), mockRisk, ranking.NewRanker(), ranking.NewRanker())
	} else {
		datastore = NewWithPostgres(pgStore.CreateTableAndNewStore(ctx, db, gormDB, false), mockRisk, ranking.NewRanker(), ranking.NewRanker())
	}

	images := make([]*storage.Image, 0, 100)
	for i := 0; i < 100; i++ {
		img := fixtures.GetImageWithUniqueComponents(50)
		id := fmt.Sprintf("%d", i)
		img.Id = id
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
