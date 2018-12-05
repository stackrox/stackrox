package store

import (
	"os"
	"testing"

	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stretchr/testify/suite"
)

func TestAPITokenStore(t *testing.T) {
	suite.Run(t, new(APITokenStoreTestSuite))
}

type APITokenStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store Store
}

func (suite *APITokenStoreTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp(suite.T().Name() + ".db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.store = New(db)
}

func (suite *APITokenStoreTestSuite) TearDownSuite() {
	suite.db.Close()
	os.Remove(suite.db.Path())
}

func (suite *APITokenStoreTestSuite) verifyTokenDoesntExist(id string) {
	token, err := suite.store.GetTokenOrNil(id)
	suite.Require().NoError(err)
	suite.Nil(token)
}

func (suite *APITokenStoreTestSuite) mustGetToken(id string) *storage.TokenMetadata {
	token, err := suite.store.GetTokenOrNil(id)
	suite.Require().NoError(err)
	suite.NotNil(token)
	return token
}

// verifyTokenIDs verifies token ids using by the get one and the get all APIs
func (suite *APITokenStoreTestSuite) verifyTokenIDs(req *v1.GetAPITokensRequest, ids ...string) {
	for _, id := range ids {
		token, err := suite.store.GetTokenOrNil(id)
		suite.NotNil(token, "couldn't find token %s", id)
		suite.NoError(err, "Error retrieving token %s", id)
	}

	tokens, err := suite.store.GetTokens(req)
	suite.Require().NoError(err)
	suite.Len(tokens, len(ids))
	// Inefficient, but doesn't matter.
	for _, id := range ids {
		found := false
		for _, token := range tokens {
			if token.GetId() == id {
				found = true
				break
			}
		}
		suite.True(found, "Couldn't find id %s in tokens %#v", id, tokens)
	}
}

func (suite *APITokenStoreTestSuite) TestRevokedTokensStore() {
	// Initially empty
	suite.verifyTokenIDs(&v1.GetAPITokensRequest{})

	const fakeID = "FAKEID"
	const otherFakeID = "OTHERFAKEID"

	exists, err := suite.store.RevokeToken(fakeID)
	suite.Require().NoError(err)
	suite.False(exists)

	suite.verifyTokenDoesntExist(fakeID)
	suite.verifyTokenDoesntExist(otherFakeID)

	err = suite.store.AddToken(&storage.TokenMetadata{Id: fakeID})
	suite.Require().NoError(err)
	suite.verifyTokenIDs(&v1.GetAPITokensRequest{}, fakeID)

	err = suite.store.AddToken(&storage.TokenMetadata{Id: otherFakeID})
	suite.Require().NoError(err)
	suite.verifyTokenIDs(&v1.GetAPITokensRequest{}, fakeID, otherFakeID)

	exists, err = suite.store.RevokeToken(fakeID)
	suite.Require().NoError(err)
	suite.True(exists)

	token := suite.mustGetToken(fakeID)
	suite.Equal(fakeID, token.GetId())
	suite.True(token.GetRevoked())

	token = suite.mustGetToken(otherFakeID)
	suite.Equal(otherFakeID, token.GetId())
	suite.False(token.GetRevoked())

	suite.verifyTokenIDs(&v1.GetAPITokensRequest{RevokedOneof: &v1.GetAPITokensRequest_Revoked{Revoked: true}}, fakeID)
	suite.verifyTokenIDs(&v1.GetAPITokensRequest{RevokedOneof: &v1.GetAPITokensRequest_Revoked{Revoked: false}}, otherFakeID)
}
