package bolt

import (
	"testing"

	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/central/apitoken/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestAPITokenStore(t *testing.T) {
	suite.Run(t, new(APITokenStoreTestSuite))
}

type APITokenStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store store.Store
}

func (suite *APITokenStoreTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp(suite.T().Name() + ".db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.store = MustNew(db)
}

func (suite *APITokenStoreTestSuite) TearDownSuite() {
	testutils.TearDownDB(suite.db)
}

func (suite *APITokenStoreTestSuite) TestStore() {
	token, exists, err := suite.store.Get("token1")
	suite.NoError(err)
	suite.False(exists)
	suite.Nil(token)

	token1 := &storage.TokenMetadata{
		Id:   "token1",
		Name: "name",
	}
	suite.NoError(suite.store.Upsert(token1))

	token2 := &storage.TokenMetadata{
		Id:   "token2",
		Name: "name",
	}
	suite.NoError(suite.store.Upsert(token2))

	token, exists, err = suite.store.Get("token1")
	suite.NoError(err)
	suite.True(exists)
	suite.Equal(token, token1)

	token.Revoked = true
	suite.NoError(suite.store.Upsert(token))

	token1.Revoked = true
	token, exists, err = suite.store.Get("token1")
	suite.NoError(err)
	suite.True(exists)
	suite.Equal(token, token1)
}
