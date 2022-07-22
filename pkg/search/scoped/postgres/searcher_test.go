package postgres

import (
	"testing"

	"github.com/golang/mock/gomock"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/dackbox/keys/transformation"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/mocks"
	"github.com/stackrox/rox/pkg/search/postgres/mapping"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stretchr/testify/suite"
)

func TestWithScoping(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(scopedSearcherTestSuite))
}

type scopedSearcherTestSuite struct {
	suite.Suite

	mockSearcher *mocks.MockSearcher

	mockCtrl *gomock.Controller
}

func (s *scopedSearcherTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockSearcher = mocks.NewMockSearcher(s.mockCtrl)
}

func (s *scopedSearcherTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *scopedSearcherTestSuite) TestScoping() {
	mapping.RegisterCategoryToTable(v1.SearchCategory_CLUSTERS, schema.ClustersSchema)
	mapping.RegisterCategoryToTable(v1.SearchCategory_NAMESPACES, schema.NamespacesSchema)

	query := search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "dep").ProtoQuery()
	scopes := []scoped.Scope{
		{
			ID:    "c1",
			Level: v1.SearchCategory_CLUSTERS,
		},
	}
	expected := search.ConjunctionQuery(
		query,
		search.NewQueryBuilder().AddExactMatches(search.ClusterID, "c1").ProtoQuery(),
	)
	actual, err := scopeQuery(query, scopes)
	s.NoError(err)
	s.Equal(expected, actual)

	scopes = []scoped.Scope{
		{
			ID:    "c1",
			Level: v1.SearchCategory_CLUSTERS,
		},
		{
			ID:    "n1",
			Level: v1.SearchCategory_NAMESPACES,
		},
	}
	expected = search.ConjunctionQuery(
		query,
		search.NewQueryBuilder().AddExactMatches(search.ClusterID, "c1").ProtoQuery(),
		search.NewQueryBuilder().AddExactMatches(search.NamespaceID, "n1").ProtoQuery(),
	)
	actual, err = scopeQuery(query, scopes)
	s.NoError(err)
	s.Equal(expected, actual)
}

type testProvider map[v1.SearchCategory]transformation.OneToMany

func (tp testProvider) Get(sc v1.SearchCategory) transformation.OneToMany {
	return tp[sc]
}
