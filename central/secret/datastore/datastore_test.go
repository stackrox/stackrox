package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve/v2"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/secret/internal/index"
	"github.com/stackrox/rox/central/secret/internal/store"
	rocksdbStore "github.com/stackrox/rox/central/secret/internal/store/rocksdb"
	secretSearch "github.com/stackrox/rox/central/secret/search"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

func TestSecretDataStore(t *testing.T) {
	suite.Run(t, new(SecretDataStoreTestSuite))
}

type SecretDataStoreTestSuite struct {
	suite.Suite

	bleveIndex bleve.Index

	db *rocksdb.RocksDB

	indexer   index.Indexer
	searcher  secretSearch.Searcher
	storage   store.Store
	datastore DataStore

	ctx context.Context
}

func (suite *SecretDataStoreTestSuite) SetupSuite() {
	var err error
	suite.bleveIndex, err = globalindex.TempInitializeIndices("")
	suite.Require().NoError(err)

	db, err := rocksdb.NewTemp(suite.T().Name() + ".db")
	suite.Require().NoError(err)

	suite.db = db

	suite.storage = rocksdbStore.New(db)
	suite.indexer = index.New(suite.bleveIndex)
	suite.searcher = secretSearch.New(suite.storage, suite.indexer)
	suite.datastore, err = New(suite.storage, suite.indexer, suite.searcher)
	suite.Require().NoError(err)

	suite.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Secret)))
}

func (suite *SecretDataStoreTestSuite) TearDownSuite() {
	suite.NoError(suite.bleveIndex.Close())
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *SecretDataStoreTestSuite) assertSearchResults(q *v1.Query, s *storage.Secret) {
	results, err := suite.datastore.SearchSecrets(suite.ctx, q)
	suite.Require().NoError(err)
	if s != nil {
		suite.Len(results, 1)
		suite.Equal(s.GetId(), results[0].GetId())
	} else {
		suite.Len(results, 0)
	}

	secrets, err := suite.datastore.SearchListSecrets(suite.ctx, q)
	suite.Require().NoError(err)
	if s != nil {
		suite.Len(secrets, 1)
		suite.Equal(s.GetId(), results[0].GetId())
	} else {
		suite.Len(secrets, 0)
	}

	rawSecrets, err := suite.datastore.SearchRawSecrets(suite.ctx, q)
	suite.Require().NoError(err)
	if s != nil {
		suite.Len(rawSecrets, 1)
		suite.Equal(s.GetId(), results[0].GetId())
	} else {
		suite.Len(rawSecrets, 0)
	}
}

func (suite *SecretDataStoreTestSuite) TestSecretsDataStore() {
	secret := fixtures.GetSecret()
	err := suite.datastore.UpsertSecret(suite.ctx, secret)
	suite.Require().NoError(err)

	foundSecret, found, err := suite.datastore.GetSecret(suite.ctx, secret.GetId())
	suite.Require().NoError(err)
	suite.True(found)
	suite.Equal(secret, foundSecret)

	_, found, err = suite.datastore.GetSecret(suite.ctx, "NONEXISTENT")
	suite.Require().NoError(err)
	suite.False(found)

	validQ := search.NewQueryBuilder().AddStrings(search.Cluster, secret.GetClusterName()).ProtoQuery()
	suite.assertSearchResults(validQ, secret)

	invalidQ := search.NewQueryBuilder().AddStrings(search.Cluster, "NONEXISTENT").ProtoQuery()
	suite.assertSearchResults(invalidQ, nil)

	err = suite.datastore.RemoveSecret(suite.ctx, secret.GetId())
	suite.Require().NoError(err)

	_, found, err = suite.datastore.GetSecret(suite.ctx, secret.GetId())
	suite.Require().NoError(err)
	suite.False(found)

	suite.assertSearchResults(validQ, nil)
}
