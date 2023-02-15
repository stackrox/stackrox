package index

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/globalindex"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	fakeID = "ABC"
)

var (
	ctx = sac.WithAllAccess(context.Background())
)

func TestSecretIndex(t *testing.T) {
	suite.Run(t, new(SecretIndexTestSuite))
}

type SecretIndexTestSuite struct {
	suite.Suite

	bleveIndex bleve.Index

	indexer Indexer
}

func (suite *SecretIndexTestSuite) SetupSuite() {
	pgtest.SkipIfPostgresEnabled(suite.T())

	tmpIndex, err := globalindex.TempInitializeIndices("")
	suite.Require().NoError(err)

	suite.bleveIndex = tmpIndex
	suite.indexer = New(tmpIndex)

	secret := fixtures.GetSecret()
	secret.Files = []*storage.SecretDataFile{
		{
			Name: "blah",
			Type: storage.SecretType_CERTIFICATE_REQUEST,
		},
	}
	suite.NoError(suite.indexer.AddSecret(secret))

	secondSecret := fixtures.GetSecret()
	secondSecret.Id = fakeID
	suite.NoError(suite.indexer.AddSecret(secondSecret))
}

func (suite *SecretIndexTestSuite) TestSecretSearch() {
	cases := []struct {
		name        string
		q           *v1.Query
		expectedIDs []string
	}{
		{
			name:        "Empty",
			q:           search.EmptyQuery(),
			expectedIDs: []string{fakeID, fixtures.GetSecret().GetId()},
		},
		{
			name:        "Secret type",
			q:           search.NewQueryBuilder().AddStrings(search.SecretType, storage.SecretType_CERTIFICATE_REQUEST.String()).ProtoQuery(),
			expectedIDs: []string{fixtures.GetSecret().GetId()},
		},
		{
			name:        "Secret type",
			q:           search.NewQueryBuilder().AddStrings(search.SecretType, search.NegateQueryString(storage.SecretType_IMAGE_PULL_SECRET.String())).ProtoQuery(),
			expectedIDs: []string{fixtures.GetSecret().GetId()},
		},
	}

	for _, c := range cases {
		suite.T().Run(c.name, func(t *testing.T) {
			results, err := suite.indexer.Search(ctx, c.q)
			require.NoError(t, err)
			resultIDs := make([]string, 0, len(results))
			for _, r := range results {
				resultIDs = append(resultIDs, r.ID)
			}
			assert.ElementsMatch(t, resultIDs, c.expectedIDs)
		})
	}
}

func (suite *SecretIndexTestSuite) TearDownSuite() {
	suite.NoError(suite.bleveIndex.Close())
}
