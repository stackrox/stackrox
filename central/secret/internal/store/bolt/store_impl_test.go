package bolt

import (
	"testing"

	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/central/secret/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestSecretStore(t *testing.T) {
	suite.Run(t, new(SecretStoreTestSuite))
}

type SecretStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store store.Store
}

func (suite *SecretStoreTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp(suite.T().Name() + ".db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.store = New(db)
}

func (suite *SecretStoreTestSuite) TearDownSuite() {
	testutils.TearDownDB(suite.db)
}

func (suite *SecretStoreTestSuite) TestSecrets() {
	var secrets = []*storage.Secret{
		{
			Id: "secret1",
		},
		{
			Id: "secret2",
		},
	}

	for _, secret := range secrets {
		err := suite.store.Upsert(secret)
		suite.NoError(err)
	}

	// Get all secrets
	var retrievedSecrets []*storage.Secret
	err := suite.store.Walk(func(secret *storage.Secret) error {
		retrievedSecrets = append(retrievedSecrets, secret)
		return nil
	})
	suite.Nil(err)
	suite.ElementsMatch(secrets, retrievedSecrets)

	for _, s := range secrets {
		secret, exists, err := suite.store.Get(s.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(s, secret)
	}

	// Get batch secrets
	var missing []int
	retrievedSecrets, missing, err = suite.store.GetMany([]string{"secret1", "secret2", "non-existant"})
	suite.Nil(err)
	suite.Len(retrievedSecrets, 2)
	suite.Len(missing, 1)
	suite.Equal(2, missing[0])
}
