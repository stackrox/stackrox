//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestSecretDataStore(t *testing.T) {
	suite.Run(t, new(SecretDataStoreTestSuite))
}

type SecretDataStoreTestSuite struct {
	suite.Suite

	datastore DataStore

	pool postgres.DB

	ctx context.Context
}

func (suite *SecretDataStoreTestSuite) SetupSuite() {
	var err error

	pgtestbase := pgtest.ForT(suite.T())
	suite.Require().NotNil(pgtestbase)
	suite.pool = pgtestbase.DB
	suite.datastore, err = GetTestPostgresDataStore(suite.T(), suite.pool)
	suite.Require().NoError(err)

	suite.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Secret)))
}

func (suite *SecretDataStoreTestSuite) TearDownSuite() {
	suite.pool.Close()
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

	nonExistentID := uuid.NewV4().String()
	_, found, err = suite.datastore.GetSecret(suite.ctx, nonExistentID)
	suite.Require().NoError(err)
	suite.False(found)

	validQ := search.NewQueryBuilder().AddStrings(search.Cluster, secret.GetClusterName()).ProtoQuery()
	suite.assertSearchResults(validQ, secret)

	invalidQ := search.NewQueryBuilder().AddStrings(search.Cluster, nonExistentID).ProtoQuery()
	suite.assertSearchResults(invalidQ, nil)

	err = suite.datastore.RemoveSecret(suite.ctx, secret.GetId())
	suite.Require().NoError(err)

	_, found, err = suite.datastore.GetSecret(suite.ctx, secret.GetId())
	suite.Require().NoError(err)
	suite.False(found)

	suite.assertSearchResults(validQ, nil)
}
