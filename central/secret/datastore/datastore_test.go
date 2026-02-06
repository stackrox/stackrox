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
	"github.com/stackrox/rox/pkg/protoassert"
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
	pgtestbase := pgtest.ForT(suite.T())
	suite.Require().NotNil(pgtestbase)
	suite.pool = pgtestbase.DB
	suite.datastore = GetTestPostgresDataStore(suite.T(), suite.pool)

	suite.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Secret)))
}

func (suite *SecretDataStoreTestSuite) TearDownTest() {
	_, err := suite.pool.Exec(suite.ctx, "TRUNCATE TABLE secrets CASCADE")
	suite.Require().NoError(err)
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

func (suite *SecretDataStoreTestSuite) TestSearchListSecrets() {
	secret1 := fixtures.GetSecret()
	secret1.Id = uuid.NewV4().String()
	secret1.Name = "test-secret-1"
	secret1.Namespace = "default"
	secret1.Files = []*storage.SecretDataFile{
		{Type: storage.SecretType_PUBLIC_CERTIFICATE},
		{Type: storage.SecretType_RSA_PRIVATE_KEY},
	}

	secret2 := fixtures.GetSecret()
	secret2.Id = uuid.NewV4().String()
	secret2.Name = "test-secret-2"
	secret2.Namespace = "kube-system"
	secret2.Files = []*storage.SecretDataFile{
		{Type: storage.SecretType_IMAGE_PULL_SECRET},
	}

	suite.NoError(suite.datastore.UpsertSecret(suite.ctx, secret1))
	suite.NoError(suite.datastore.UpsertSecret(suite.ctx, secret2))

	// Test retrieval with empty query
	results, err := suite.datastore.SearchListSecrets(suite.ctx, search.EmptyQuery())
	suite.NoError(err)
	suite.Equal(len(results), 2)

	// Find our test secrets
	var found1, found2 *storage.ListSecret
	for _, r := range results {
		if r.GetId() == secret1.GetId() {
			found1 = r
		} else if r.GetId() == secret2.GetId() {
			found2 = r
		}
	}

	suite.NotNil(found1)
	suite.Equal(secret1.GetName(), found1.GetName())
	suite.Equal(secret1.GetNamespace(), found1.GetNamespace())
	suite.ElementsMatch(
		[]storage.SecretType{storage.SecretType_PUBLIC_CERTIFICATE, storage.SecretType_RSA_PRIVATE_KEY},
		found1.GetTypes(),
	)

	suite.NotNil(found2)
	suite.Equal([]storage.SecretType{storage.SecretType_IMAGE_PULL_SECRET}, found2.GetTypes())
}

func (suite *SecretDataStoreTestSuite) TestSearchListSecrets_NoFiles() {
	// Test secret with no files (should return UNDETERMINED type)
	secret := fixtures.GetSecret()
	secret.Id = uuid.NewV4().String()
	secret.Name = "empty-secret"
	secret.Files = nil // No files

	suite.NoError(suite.datastore.UpsertSecret(suite.ctx, secret))

	query := search.NewQueryBuilder().AddStrings(search.SecretID, secret.GetId()).ProtoQuery()
	results, err := suite.datastore.SearchListSecrets(suite.ctx, query)

	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal([]storage.SecretType{storage.SecretType_UNDETERMINED}, results[0].GetTypes())
}

func (suite *SecretDataStoreTestSuite) TestSearchListSecrets_WithFilter() {
	// Test that search filters work correctly with single-pass query
	secret1 := fixtures.GetSecret()
	secret1.Id = uuid.NewV4().String()
	secret1.Name = "filter-test-1"
	secret1.Namespace = "default"

	secret2 := fixtures.GetSecret()
	secret2.Id = uuid.NewV4().String()
	secret2.Name = "filter-test-2"
	secret2.Namespace = "kube-system"

	suite.NoError(suite.datastore.UpsertSecret(suite.ctx, secret1))
	suite.NoError(suite.datastore.UpsertSecret(suite.ctx, secret2))

	// Filter by namespace
	query := search.NewQueryBuilder().AddExactMatches(search.Namespace, "kube-system").ProtoQuery()
	results, err := suite.datastore.SearchListSecrets(suite.ctx, query)

	suite.NoError(err)
	// Find our test secret in results
	var found *storage.ListSecret
	for _, r := range results {
		if r.GetId() == secret2.GetId() {
			found = r
			break
		}
	}
	suite.NotNil(found)
	suite.Equal("kube-system", found.GetNamespace())
}

func (suite *SecretDataStoreTestSuite) TestSearchListSecrets_DuplicateTypes() {
	// Test that framework correctly deduplicates types via jsonb_agg
	secret := fixtures.GetSecret()
	secret.Id = uuid.NewV4().String()
	secret.Name = "duplicate-types-secret"
	secret.Files = []*storage.SecretDataFile{
		{Type: storage.SecretType_PUBLIC_CERTIFICATE},
		{Type: storage.SecretType_PUBLIC_CERTIFICATE}, // Duplicate
		{Type: storage.SecretType_RSA_PRIVATE_KEY},
	}

	suite.NoError(suite.datastore.UpsertSecret(suite.ctx, secret))

	query := search.NewQueryBuilder().AddStrings(search.SecretID, secret.GetId()).ProtoQuery()
	results, err := suite.datastore.SearchListSecrets(suite.ctx, query)

	suite.NoError(err)
	suite.Len(results, 1)
	// Should have only 2 unique types
	suite.Len(results[0].GetTypes(), 2)
	suite.ElementsMatch(
		[]storage.SecretType{storage.SecretType_PUBLIC_CERTIFICATE, storage.SecretType_RSA_PRIVATE_KEY},
		results[0].GetTypes(),
	)
}

func (suite *SecretDataStoreTestSuite) TestSecretsDataStore() {
	secret := fixtures.GetSecret()
	err := suite.datastore.UpsertSecret(suite.ctx, secret)
	suite.Require().NoError(err)

	foundSecret, found, err := suite.datastore.GetSecret(suite.ctx, secret.GetId())
	suite.Require().NoError(err)
	suite.True(found)
	protoassert.Equal(suite.T(), secret, foundSecret)

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

func (suite *SecretDataStoreTestSuite) TestSearchSecrets() {
	// Create test secrets
	secret1 := fixtures.GetSecret()
	secret1.Id = uuid.NewV4().String()
	secret1.Name = "test-secret-1"
	secret1.ClusterName = "cluster-1"
	secret1.Namespace = "namespace-1"
	err := suite.datastore.UpsertSecret(suite.ctx, secret1)
	suite.Require().NoError(err)

	secret2 := fixtures.GetSecret()
	secret2.Id = uuid.NewV4().String()
	secret2.Name = "test-secret-2"
	secret2.ClusterName = "cluster-2"
	secret2.Namespace = "namespace-2"
	err = suite.datastore.UpsertSecret(suite.ctx, secret2)
	suite.Require().NoError(err)

	secret3 := fixtures.GetSecret()
	secret3.Id = uuid.NewV4().String()
	secret3.Name = "test-secret-3"
	secret3.ClusterName = "cluster-1"
	secret3.Namespace = "namespace-3"
	err = suite.datastore.UpsertSecret(suite.ctx, secret3)
	suite.Require().NoError(err)

	// Define test cases
	testCases := []struct {
		name          string
		query         *v1.Query
		expectedCount int
		expectedIDs   []string
		expectedNames []string
	}{
		{
			name:          "empty query returns all secrets with names populated",
			query:         search.EmptyQuery(),
			expectedCount: 3,
			expectedIDs:   []string{secret1.GetId(), secret2.GetId(), secret3.GetId()},
			expectedNames: []string{"test-secret-1", "test-secret-2", "test-secret-3"},
		},
		{
			name:          "nil query defaults to empty query",
			query:         nil,
			expectedCount: 3,
			expectedIDs:   []string{secret1.GetId(), secret2.GetId(), secret3.GetId()},
			expectedNames: []string{"test-secret-1", "test-secret-2", "test-secret-3"},
		},
		{
			name:          "query by secret name - exact match",
			query:         search.NewQueryBuilder().AddExactMatches(search.SecretName, "test-secret-1").ProtoQuery(),
			expectedCount: 1,
			expectedIDs:   []string{secret1.GetId()},
			expectedNames: []string{"test-secret-1"},
		},
	}

	// Run test cases
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			results, err := suite.datastore.SearchSecrets(suite.ctx, tc.query)
			suite.NoError(err)
			suite.Len(results, tc.expectedCount, "Expected %d results, got %d", tc.expectedCount, len(results))

			actualIDs := make([]string, 0, len(results))
			actualNames := make([]string, 0, len(results))
			for _, result := range results {
				actualIDs = append(actualIDs, result.GetId())
				suite.Equal(v1.SearchCategory_SECRETS, result.GetCategory())
				actualNames = append(actualNames, result.GetName())
			}

			if len(tc.expectedIDs) > 0 {
				suite.ElementsMatch(tc.expectedIDs, actualIDs)
				suite.ElementsMatch(tc.expectedNames, actualNames)
			}
		})
	}
}
