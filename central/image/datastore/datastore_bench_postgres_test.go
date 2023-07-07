//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	pgStore "github.com/stackrox/rox/central/image/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func BenchmarkImageGetMany(b *testing.B) {

	ctx := sac.WithAllAccess(context.Background())

	source := pgtest.GetConnectionString(b)
	config, err := postgres.ParseConfig(source)
	require.NoError(b, err)

	pool, err := postgres.New(ctx, config)
	require.NoError(b, err)
	gormDB := pgtest.OpenGormDB(b, source)
	defer pgtest.CloseGormDB(b, gormDB)

	db := pool
	defer db.Close()

	pgStore.Destroy(ctx, db)
	mockRisk := mockRisks.NewMockDataStore(gomock.NewController(b))
	datastore := NewWithPostgres(pgStore.CreateTableAndNewStore(ctx, db, gormDB, false), pgStore.NewIndexer(db), mockRisk, ranking.NewRanker(), ranking.NewRanker())

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
			_, err = datastore.GetImagesBatch(ctx, ids)
			require.NoError(b, err)
		}
	})

	b.Run("GetManyImageMetadata", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err = datastore.GetManyImageMetadata(ctx, ids)
			require.NoError(b, err)
		}
	})
}
