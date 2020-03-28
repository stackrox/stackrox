package store

import (
	"sort"
	"testing"

	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestAuthProviderStore(t *testing.T) {
	suite.Run(t, new(AuthProviderStoreTestSuite))
}

type AuthProviderStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store Store
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

	// Test Add
	for _, r := range authProviders {
		r.Id = uuid.NewV4().String()
		err := suite.store.AddAuthProvider(r)
		suite.NoError(err)
	}

	sort.Slice(authProviders, func(i, j int) bool {
		return authProviders[i].Id < authProviders[j].Id
	})

	// Test GetAllAuthProviders
	allProviders, err := suite.store.GetAllAuthProviders()
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
		suite.NoError(suite.store.UpdateAuthProvider(r))
	}

	allProviders, err = suite.store.GetAllAuthProviders()
	suite.Require().NoError(err)
	sort.Slice(allProviders, func(i, j int) bool {
		return allProviders[i].Id < authProviders[j].Id
	})

	suite.Equal(authProviders, allProviders)

	// Test Remove
	for _, r := range authProviders {
		suite.NoError(suite.store.RemoveAuthProvider(r.GetId()))
	}

	allProviders, err = suite.store.GetAllAuthProviders()
	suite.NoError(err)
	suite.Empty(allProviders)
}
