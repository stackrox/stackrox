package resolvers

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/graph-gophers/graphql-go"
	"github.com/jackc/pgx/v4/pgxpool"
	imageComponentCVEEdgeDS "github.com/stackrox/rox/central/componentcveedge/datastore"
	imageComponentCVEEdgePostgres "github.com/stackrox/rox/central/componentcveedge/datastore/store/postgres"
	imageComponentCVEEdgeSearch "github.com/stackrox/rox/central/componentcveedge/search"
	imageCVEDS "github.com/stackrox/rox/central/cve/image/datastore"
	imageCVESearch "github.com/stackrox/rox/central/cve/image/datastore/search"
	imageCVEPostgres "github.com/stackrox/rox/central/cve/image/datastore/store/postgres"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	imagePostgres "github.com/stackrox/rox/central/image/datastore/store/postgres"
	imageComponentDS "github.com/stackrox/rox/central/imagecomponent/datastore"
	imageComponentPostgres "github.com/stackrox/rox/central/imagecomponent/datastore/store/postgres"
	imageComponentSearch "github.com/stackrox/rox/central/imagecomponent/search"
	"github.com/stackrox/rox/central/ranking"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/grpc/authn"
	mockIdentity "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func setupPostgresConn(t testing.TB) (*pgxpool.Pool, *gorm.DB) {
	source := pgtest.GetConnectionString(t)
	config, err := pgxpool.ParseConfig(source)
	assert.NoError(t, err)

	pool, err := pgxpool.ConnectConfig(context.Background(), config)
	assert.NoError(t, err)

	gormDB := pgtest.OpenGormDB(t, source)

	return pool, gormDB
}

func setupResolver(
	t testing.TB,
	imageDataStore imageDS.DataStore,
	imageComponentDataStore imageComponentDS.DataStore,
	cveDataStore imageCVEDS.DataStore,
	imageComponentCVEEdgeDatastore imageComponentCVEEdgeDS.DataStore,
) (*Resolver, *graphql.Schema) {
	// loaders used by graphql layer
	registerImageLoader(t, imageDataStore)
	registerImageComponentLoader(t, imageComponentDataStore)
	registerImageCveLoader(t, cveDataStore)

	resolver := &Resolver{
		ImageDataStore:            imageDataStore,
		ImageComponentDataStore:   imageComponentDataStore,
		ImageCVEDataStore:         cveDataStore,
		ComponentCVEEdgeDataStore: imageComponentCVEEdgeDatastore,
	}

	schema, err := graphql.ParseSchema(Schema(), resolver)
	assert.NoError(t, err)

	return resolver, schema
}

func createImageDatastore(_ testing.TB, ctrl *gomock.Controller, db *pgxpool.Pool, gormDB *gorm.DB) imageDS.DataStore {
	ctx := context.Background()
	imagePostgres.Destroy(ctx, db)

	return imageDS.NewWithPostgres(
		imagePostgres.CreateTableAndNewStore(ctx, db, gormDB, false),
		imagePostgres.NewIndexer(db),
		mockRisks.NewMockDataStore(ctrl),
		ranking.NewRanker(),
		ranking.NewRanker(),
	)
}

func createImageComponentDatastore(_ testing.TB, ctrl *gomock.Controller, db *pgxpool.Pool, gormDB *gorm.DB) imageComponentDS.DataStore {
	ctx := context.Background()
	imageComponentPostgres.Destroy(ctx, db)

	mockRisk := mockRisks.NewMockDataStore(ctrl)
	storage := imageComponentPostgres.CreateTableAndNewStore(ctx, db, gormDB)
	indexer := imageComponentPostgres.NewIndexer(db)
	searcher := imageComponentSearch.NewV2(storage, indexer)

	return imageComponentDS.New(
		nil, storage, indexer, searcher, mockRisk, ranking.NewRanker(),
	)
}

func createImageCVEDatastore(t testing.TB, db *pgxpool.Pool, gormDB *gorm.DB) imageCVEDS.DataStore {
	ctx := context.Background()
	imageCVEPostgres.Destroy(ctx, db)

	storage := imageCVEPostgres.CreateTableAndNewStore(ctx, db, gormDB)
	indexer := imageCVEPostgres.NewIndexer(db)
	searcher := imageCVESearch.New(storage, indexer)
	datastore, err := imageCVEDS.New(storage, indexer, searcher, nil)
	assert.NoError(t, err)

	return datastore
}

func createImageComponentCVEEdgeDatastore(_ testing.TB, db *pgxpool.Pool, gormDB *gorm.DB) imageComponentCVEEdgeDS.DataStore {
	ctx := context.Background()
	imageComponentCVEEdgePostgres.Destroy(ctx, db)

	storage := imageComponentCVEEdgePostgres.CreateTableAndNewStore(ctx, db, gormDB)
	indexer := imageComponentCVEEdgePostgres.NewIndexer(db)
	searcher := imageComponentCVEEdgeSearch.NewV2(storage, indexer)

	return imageComponentCVEEdgeDS.New(nil, storage, indexer, searcher)
}

func registerImageLoader(_ testing.TB, ds imageDS.DataStore) {
	loaders.RegisterTypeFactory(reflect.TypeOf(storage.Image{}), func() interface{} {
		return loaders.NewImageLoader(ds)
	})
}

func registerImageComponentLoader(_ testing.TB, ds imageComponentDS.DataStore) {
	loaders.RegisterTypeFactory(reflect.TypeOf(storage.ImageComponent{}), func() interface{} {
		return loaders.NewComponentLoader(ds)
	})
}

func registerImageCveLoader(_ testing.TB, ds imageCVEDS.DataStore) {
	loaders.RegisterTypeFactory(reflect.TypeOf(storage.ImageCVE{}), func() interface{} {
		return loaders.NewImageCVELoader(ds)
	})
}

func getTestImages(imageCount int) []*storage.Image {
	images := make([]*storage.Image, 0, imageCount)
	for i := 0; i < imageCount; i++ {
		img := fixtures.GetImageWithUniqueComponents(100)
		id := fmt.Sprintf("%d", i)
		img.Id = id
		images = append(images, img)
	}
	return images
}

func contextWithImagePerm(t testing.TB, ctrl *gomock.Controller) context.Context {
	id := mockIdentity.NewMockIdentity(ctrl)
	id.EXPECT().Permissions().Return(map[string]storage.Access{"Image": storage.Access_READ_ACCESS}).AnyTimes()
	return authn.ContextWithIdentity(sac.WithAllAccess(loaders.WithLoaderContext(context.Background())), id, t)
}
