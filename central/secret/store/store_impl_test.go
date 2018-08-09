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
	secret1 := &v1.Secret{
		Id: "secret1",
	}
	secret2 := &v1.Secret{
		Id: "secret2",
	}
	secrets := []*v1.Secret{secret1, secret2}
	for _, secret := range secrets {
		err := suite.store.UpsertSecret(secret)
		suite.NoError(err)
	}

	// Get all secrets
	retrievedSecrets, err := suite.store.GetAllSecrets()
	suite.Nil(err)
	suite.ElementsMatch(secrets, retrievedSecrets)

	// Get batch secrets
	retrievedSecrets, err = suite.store.GetSecretsBatch([]string{"secret1", "secret2"})
	suite.Nil(err)
	suite.ElementsMatch(secrets, retrievedSecrets)

	// Get secret
	retrievedSecret, exists, err := suite.store.GetSecret("secret1")
	suite.Nil(err)
	suite.True(exists)
	suite.Equal(secret1, retrievedSecret)
}

func (suite *SecretStoreTestSuite) TestRelationships() {
	relatinoship1 := &v1.SecretRelationship{
		Id: "secret1",
	}
	relatinoship2 := &v1.SecretRelationship{
		Id: "secret2",
	}
	relationships := []*v1.SecretRelationship{relatinoship1, relatinoship2}
	for _, p := range relationships {
		err := suite.store.UpsertRelationship(p)
		suite.NoError(err)
	}

	// Get batch relationships
	retrievedRelationships, err := suite.store.GetRelationshipBatch([]string{"secret1", "secret2"})
	suite.Nil(err)
	suite.ElementsMatch(relationships, retrievedRelationships)

	// Get secret
	retrievedRelationship, exists, err := suite.store.GetRelationship("secret1")
	suite.Nil(err)
	suite.True(exists)
	suite.Equal(relatinoship1, retrievedRelationship)
}
