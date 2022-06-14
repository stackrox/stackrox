package bolt

import (
	"context"
	"strings"
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/bolthelper"
	"github.com/stackrox/stackrox/pkg/testutils"
	"github.com/stackrox/stackrox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestImageIntegrationStore(t *testing.T) {
	suite.Run(t, new(ImageIntegrationStoreTestSuite))
}

type ImageIntegrationStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store *storeImpl
}

func (suite *ImageIntegrationStoreTestSuite) SetupTest() {
	db, err := bolthelper.NewTemp(testutils.DBFileName(suite))
	if err != nil {
		suite.FailNow("failure: "+suite.T().Name(), err.Error())
	}

	suite.db = db
	suite.store = New(db)
}

func (suite *ImageIntegrationStoreTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
}

func (suite *ImageIntegrationStoreTestSuite) TestIntegrations() {
	integration := []*storage.ImageIntegration{
		{
			Id:   uuid.NewV4().String(),
			Name: "registry1",
			IntegrationConfig: &storage.ImageIntegration_Docker{
				Docker: &storage.DockerConfig{
					Endpoint: "https://endpoint1",
				},
			},
		},
		{
			Id:   uuid.NewV4().String(),
			Name: "registry2",
			IntegrationConfig: &storage.ImageIntegration_Docker{
				Docker: &storage.DockerConfig{
					Endpoint: "https://endpoint2",
				},
			},
		},
	}

	// Test Add
	ctx := context.Background()
	for _, r := range integration {
		err := suite.store.Upsert(ctx, r)
		suite.NoError(err)
	}

	for _, r := range integration {
		got, exists, err := suite.store.Get(ctx, r.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, r)
	}

	// Test Update
	for _, r := range integration {
		r.Name += "-ext"
	}
	for _, r := range integration {
		suite.NoError(suite.store.Upsert(ctx, r))
	}
	for _, r := range integration {
		r.Name = strings.TrimSuffix(r.Name, "-ext")
	}
	for _, r := range integration {
		suite.NoError(suite.store.Upsert(ctx, r))
	}

	for _, r := range integration {
		got, exists, err := suite.store.Get(ctx, r.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, r)
	}

	// Test Remove
	for _, r := range integration {
		suite.NoError(suite.store.Delete(ctx, r.GetId()))
	}

	for _, r := range integration {
		_, exists, err := suite.store.Get(ctx, r.GetId())
		suite.NoError(err)
		suite.False(exists)
	}
}
