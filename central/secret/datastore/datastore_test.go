package datastore

import (
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/secret/index"
	secretSearch "github.com/stackrox/rox/central/secret/search"
	"github.com/stackrox/rox/central/secret/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

func TestSecretDataStore(t *testing.T) {
	suite.Run(t, new(SecretDataStoreTestSuite))
}

type SecretDataStoreTestSuite struct {
	suite.Suite

	bleveIndex bleve.Index

	indexer   index.Indexer
	searcher  secretSearch.Searcher
	storage   store.Store
	datastore DataStore
}

func (suite *SecretDataStoreTestSuite) SetupSuite() {
	var err error
	suite.bleveIndex, err = globalindex.TempInitializeIndices("")
	suite.Require().NoError(err)

	db, err := bolthelper.NewTemp(suite.T().Name() + ".db")
	suite.Require().NoError(err)

	suite.storage = store.New(db)
	suite.searcher = secretSearch.New(suite.storage, suite.bleveIndex)
	suite.indexer = index.New(suite.bleveIndex)
	suite.datastore = New(suite.storage, suite.indexer, suite.searcher)
}

func (suite *SecretDataStoreTestSuite) TeardownSuite() {
	suite.bleveIndex.Close()
}

func (suite *SecretDataStoreTestSuite) assertSearchResults(q *v1.Query, s *v1.Secret) {
	results, err := suite.datastore.SearchSecrets(q)
	suite.Require().NoError(err)
	if s != nil {
		suite.Len(results, 1)
		suite.Equal(s.GetId(), results[0].GetId())
	} else {
		suite.Len(results, 0)
	}

	secrets, err := suite.datastore.SearchListSecrets(q)
	suite.Require().NoError(err)
	if s != nil {
		suite.Len(secrets, 1)
		suite.Equal(s.GetId(), results[0].GetId())
	} else {
		suite.Len(secrets, 0)
	}

}

func (suite *SecretDataStoreTestSuite) TestSecretsDataStore() {
	secret := fixtures.GetSecret()
	err := suite.datastore.UpsertSecret(secret)
	suite.Require().NoError(err)

	foundSecret, found, err := suite.datastore.GetSecret(secret.GetId())
	suite.Require().NoError(err)
	suite.True(found)
	suite.Equal(secret, foundSecret)

	_, found, err = suite.datastore.GetSecret("NONEXISTENT")
	suite.Require().NoError(err)
	suite.False(found)

	validQ := search.NewQueryBuilder().AddStrings(search.Cluster, secret.GetClusterName()).ProtoQuery()
	suite.assertSearchResults(validQ, secret)

	invalidQ := search.NewQueryBuilder().AddStrings(search.Cluster, "NONEXISTENT").ProtoQuery()
	suite.assertSearchResults(invalidQ, nil)

	err = suite.datastore.RemoveSecret(secret.GetId())
	suite.Require().NoError(err)

	_, found, err = suite.datastore.GetSecret(secret.GetId())
	suite.Require().NoError(err)
	suite.False(found)

	suite.assertSearchResults(validQ, nil)
}
