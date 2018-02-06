package boltdb

import (
	"io/ioutil"
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/suite"
)

func TestBoltAuthProviders(t *testing.T) {
	suite.Run(t, new(BoltAuthProviderTestSuite))
}

type BoltAuthProviderTestSuite struct {
	suite.Suite
	*BoltDB
}

func (suite *BoltAuthProviderTestSuite) SetupSuite() {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		suite.FailNow("Failed to get temporary directory", err.Error())
	}
	db, err := New(tmpDir)
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.BoltDB = db
}

func (suite *BoltAuthProviderTestSuite) TeardownSuite() {
	suite.Close()
	os.Remove(suite.Path())
}

func (suite *BoltAuthProviderTestSuite) TestAuthProviders() {
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
		id, err := suite.AddAuthProvider(r)
		suite.NoError(err)
		suite.NotEmpty(id)
	}

	for _, r := range authProviders {
		got, exists, err := suite.GetAuthProvider(r.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, r)
	}

	// Test Update
	for _, r := range authProviders {
		r.Name += " in production"
	}

	for _, r := range authProviders {
		suite.NoError(suite.UpdateAuthProvider(r))
	}

	for _, r := range authProviders {
		got, exists, err := suite.GetAuthProvider(r.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, r)
	}

	// Test Remove
	for _, r := range authProviders {
		suite.NoError(suite.RemoveAuthProvider(r.GetId()))
	}

	for _, r := range authProviders {
		_, exists, err := suite.GetAuthProvider(r.GetId())
		suite.NoError(err)
		suite.False(exists)
	}
}
