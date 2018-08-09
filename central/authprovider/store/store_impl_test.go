package store

import (
	"os"
	"sort"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
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

func (suite *AuthProviderStoreTestSuite) TeardownSuite() {
	suite.db.Close()
	os.Remove(suite.db.Path())
}

func (suite *AuthProviderStoreTestSuite) TestAuthProviders() {
	authProviders := []*v1.AuthProvider{
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
		id, err := suite.store.AddAuthProvider(r)
		suite.NoError(err)
		suite.NotEmpty(id)
	}

	for _, r := range authProviders {
		got, exists, err := suite.store.GetAuthProvider(r.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, r)
	}

	// Test GetAuthProviders
	retrievedProviders, err := suite.store.GetAuthProviders(&v1.GetAuthProvidersRequest{})
	suite.NoError(err)
	suite.Len(retrievedProviders, len(authProviders))
	// Sort them alphabetically by name (which is how we should keep our authProviders slice sorted).
	sort.Slice(retrievedProviders, func(i, j int) bool {
		return retrievedProviders[i].Name < retrievedProviders[j].Name
	})
	for i, retrievedProvider := range retrievedProviders {
		suite.Equal(authProviders[i], retrievedProvider)
	}

	// Test GetAuthProviders with a non-empty request
	retrievedProvidersByType, err := suite.store.GetAuthProviders(&v1.GetAuthProvidersRequest{Type: "Auth Provider 1"})
	suite.NoError(err)
	suite.Len(retrievedProvidersByType, 1)
	suite.Equal(authProviders[0], retrievedProvidersByType[0])

	retrievedProvidersByName, err := suite.store.GetAuthProviders(&v1.GetAuthProvidersRequest{Name: "authProvider1"})
	suite.NoError(err)
	suite.Len(retrievedProvidersByName, 1)
	suite.Equal(authProviders[0], retrievedProvidersByName[0])

	// Test Update
	for _, r := range authProviders {
		r.Name += " in production"
	}

	for _, r := range authProviders {
		suite.NoError(suite.store.UpdateAuthProvider(r))
	}

	for _, r := range authProviders {
		got, exists, err := suite.store.GetAuthProvider(r.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, r)
	}

	// Test Remove
	for _, r := range authProviders {
		suite.NoError(suite.store.RemoveAuthProvider(r.GetId()))
	}

	for _, r := range authProviders {
		_, exists, err := suite.store.GetAuthProvider(r.GetId())
		suite.NoError(err)
		suite.False(exists)
	}
}
