//go:build sql_integration
// +build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/image/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/require"
)

func BenchmarkImageGetMany(b *testing.B) {
	envIsolator := envisolator.NewEnvIsolator(b)
	envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")
	defer envIsolator.RestoreAll()

	if !features.PostgresDatastore.Enabled() {
		b.Skip("Skip postgres store tests")
		b.SkipNow()
	}

	ctx := sac.WithAllAccess(context.Background())

	source := pgtest.GetConnectionString(b)
	config, err := pgxpool.ParseConfig(source)
	require.NoError(b, err)

	pool, err := pgxpool.ConnectConfig(ctx, config)
	require.NoError(b, err)
	gormDB := pgtest.OpenGormDB(b, source)
	defer pgtest.CloseGormDB(b, gormDB)

	db := pool
	defer db.Close()

	postgres.Destroy(ctx, db)
	mockRisk := mockRisks.NewMockDataStore(gomock.NewController(b))
	datastore := NewWithPostgres(postgres.CreateTableAndNewStore(ctx, db, gormDB, false), postgres.NewIndexer(db), mockRisk, ranking.NewRanker(), ranking.NewRanker())

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
