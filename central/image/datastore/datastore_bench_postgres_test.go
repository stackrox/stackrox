//go:build sql_integration

package datastore

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"testing"

	imageCVEInfoDS "github.com/stackrox/rox/central/cve/image/info/datastore"
	"github.com/stackrox/rox/central/image/datastore/keyfence"
	pgStoreV2 "github.com/stackrox/rox/central/image/datastore/store/v2/postgres"
	"github.com/stackrox/rox/central/ranking"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func BenchmarkImageGetMany(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())

	testDB := pgtest.ForT(b)

	db := testDB.DB

	mockRisk := mockRisks.NewMockDataStore(gomock.NewController(b))
	imageCVEInfo := imageCVEInfoDS.GetTestPostgresDataStore(b, db)
	datastore := NewWithPostgres(pgStoreV2.New(db, false, keyfence.ImageKeyFenceSingleton()), mockRisk, ranking.NewRanker(), ranking.NewRanker(), imageCVEInfo)

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

	db := testDB.DB

	mockRisk := mockRisks.NewMockDataStore(gomock.NewController(b))
	imageCVEInfo := imageCVEInfoDS.GetTestPostgresDataStore(b, db)
	datastore := NewWithPostgres(pgStoreV2.New(db, false, keyfence.ImageKeyFenceSingleton()), mockRisk, ranking.NewRanker(), ranking.NewRanker(), imageCVEInfo)

	images := make([]*storage.Image, 0, 100)
	for i := 0; i < 100; i++ {
		img := fixtures.GetImageWithUniqueComponents(50)
		data := make([]byte, 10)
		if _, err := rand.Read(data); err == nil {
			id := fmt.Sprintf("%x", sha256.Sum256(data))
			require.NoError(b, err)
			img.Id = id
		}
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
