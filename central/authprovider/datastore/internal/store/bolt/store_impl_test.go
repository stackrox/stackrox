package bolt

import (
	"context"
	"sort"
	"testing"

	"github.com/stackrox/rox/central/authprovider/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestAuthProviderStore(t *testing.T) {
	suite.Run(t, new(AuthProviderStoreTestSuite))
}

type AuthProviderStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store store.Store
}

func (suite *AuthProviderStoreTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp(suite.T().Name() + ".db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.store = New(db)
}

func (suite *AuthProviderStoreTestSuite) TearDownSuite() {
	testutils.TearDownDB(suite.db)
}

func (suite *AuthProviderStoreTestSuite) TestAuthProviders() {
	authProviders := []*storage.AuthProvider{
		{
			Name: "authProvider1",
			Type: "Auth Provider 1",
		},
		{
			Name: "authProvider2",
			Type: "Auth Provider 2",
		},
	}

	ctx := sac.WithAllAccess(context.Background())

	// Test Add
	for _, r := range authProviders {
		r.Id = uuid.NewV4().String()
		err := suite.store.Upsert(ctx, r)
		suite.NoError(err)
	}

	sort.Slice(authProviders, func(i, j int) bool {
		return authProviders[i].Id < authProviders[j].Id
	})

	// Test GetAllAuthProviders
	allProviders, err := suite.store.GetAll(ctx)
	suite.Require().NoError(err)
	sort.Slice(allProviders, func(i, j int) bool {
		return allProviders[i].Id < authProviders[j].Id
	})

	suite.Equal(authProviders, allProviders)

	// Test Update
	for _, r := range authProviders {
		r.Name += " in production"
	}

	for _, r := range authProviders {
		suite.NoError(suite.store.Upsert(ctx, r))
	}

	allProviders, err = suite.store.GetAll(ctx)
	suite.Require().NoError(err)
	sort.Slice(allProviders, func(i, j int) bool {
		return allProviders[i].Id < authProviders[j].Id
	})

	suite.Equal(authProviders, allProviders)

	// Test Remove
	for _, r := range authProviders {
		suite.NoError(suite.store.Delete(ctx, r.GetId()))
	}

	allProviders, err = suite.store.GetAll(ctx)
	suite.NoError(err)
	suite.Empty(allProviders)
}
