package store

import (
	"os"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stretchr/testify/suite"
)

func TestSecretStore(t *testing.T) {
	suite.Run(t, new(SecretStoreTestSuite))
}

type SecretStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store Store
}

func (suite *SecretStoreTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp(suite.T().Name() + ".db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.store = New(db)
}

func (suite *SecretStoreTestSuite) TeardownSuite() {
	suite.db.Close()
	os.Remove(suite.db.Path())
}

func (suite *SecretStoreTestSuite) TestSecrets() {
	var secrets = []*v1.Secret{
		{
			Id: "secret1",
		},
		{
			Id: "secret2",
		},
	}

	for _, secret := range secrets {
		err := suite.store.UpsertSecret(secret)
		suite.NoError(err)
	}

	// Get all secrets
	retrievedSecrets, err := suite.store.GetAllSecrets()
	suite.Nil(err)
	suite.ElementsMatch(secrets, retrievedSecrets)

	for _, s := range secrets {
		secret, exists, err := suite.store.GetSecret(s.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(s, secret)
	}

	// Get batch secrets
	retrievedListSecrets, err := suite.store.ListSecrets([]string{"secret1", "secret2"})
	suite.Nil(err)
	suite.Len(retrievedListSecrets, 2)
}
