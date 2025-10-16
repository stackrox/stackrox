package service

import (
	"context"
	"testing"

	roleBindingMocks "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func getSubjects() []*storage.Subject {
	return []*storage.Subject{
		storage.Subject_builder{
			Name: "def",
			Kind: storage.SubjectKind_GROUP,
		}.Build(),
		storage.Subject_builder{
			Name: "def",
			Kind: storage.SubjectKind_USER,
		}.Build(),
		storage.Subject_builder{
			Name: "hij",
			Kind: storage.SubjectKind_SERVICE_ACCOUNT,
		}.Build(),
		storage.Subject_builder{
			Name: "abc",
			Kind: storage.SubjectKind_USER,
		}.Build(),
		storage.Subject_builder{
			Name: "abc",
			Kind: storage.SubjectKind_GROUP,
		}.Build(),
	}
}

func TestSortSubjects(t *testing.T) {
	cases := []struct {
		name        string
		sortOptions []*v1.QuerySortOption
		expected    []*storage.Subject
		hasError    bool
	}{
		{
			name: "subject sort",
			sortOptions: []*v1.QuerySortOption{
				v1.QuerySortOption_builder{
					Field:    search.SubjectName.String(),
					Reversed: false,
				}.Build(),
			},
			expected: []*storage.Subject{
				storage.Subject_builder{
					Name: "abc",
					Kind: storage.SubjectKind_USER,
				}.Build(),
				storage.Subject_builder{
					Name: "abc",
					Kind: storage.SubjectKind_GROUP,
				}.Build(),
				storage.Subject_builder{
					Name: "def",
					Kind: storage.SubjectKind_GROUP,
				}.Build(),
				storage.Subject_builder{
					Name: "def",
					Kind: storage.SubjectKind_USER,
				}.Build(),
				storage.Subject_builder{
					Name: "hij",
					Kind: storage.SubjectKind_SERVICE_ACCOUNT,
				}.Build(),
			},
		},
		{
			name: "subject sort - reversed",
			sortOptions: []*v1.QuerySortOption{
				v1.QuerySortOption_builder{
					Field:    search.SubjectName.String(),
					Reversed: true,
				}.Build(),
			},
			expected: []*storage.Subject{
				storage.Subject_builder{
					Name: "hij",
					Kind: storage.SubjectKind_SERVICE_ACCOUNT,
				}.Build(),
				storage.Subject_builder{
					Name: "def",
					Kind: storage.SubjectKind_GROUP,
				}.Build(),
				storage.Subject_builder{
					Name: "def",
					Kind: storage.SubjectKind_USER,
				}.Build(),
				storage.Subject_builder{
					Name: "abc",
					Kind: storage.SubjectKind_USER,
				}.Build(),
				storage.Subject_builder{
					Name: "abc",
					Kind: storage.SubjectKind_GROUP,
				}.Build(),
			},
		},
		{
			name: "subject sort - kind sort",
			sortOptions: []*v1.QuerySortOption{
				v1.QuerySortOption_builder{
					Field: search.SubjectName.String(),
				}.Build(),
				v1.QuerySortOption_builder{
					Field: search.SubjectKind.String(),
				}.Build(),
			},
			expected: []*storage.Subject{
				storage.Subject_builder{
					Name: "abc",
					Kind: storage.SubjectKind_GROUP,
				}.Build(),
				storage.Subject_builder{
					Name: "abc",
					Kind: storage.SubjectKind_USER,
				}.Build(),
				storage.Subject_builder{
					Name: "def",
					Kind: storage.SubjectKind_GROUP,
				}.Build(),
				storage.Subject_builder{
					Name: "def",
					Kind: storage.SubjectKind_USER,
				}.Build(),
				storage.Subject_builder{
					Name: "hij",
					Kind: storage.SubjectKind_SERVICE_ACCOUNT,
				}.Build(),
			},
		},
		{
			name: "subject sort - kind sort",
			sortOptions: []*v1.QuerySortOption{
				v1.QuerySortOption_builder{
					Field: search.SubjectName.String(),
				}.Build(),
				v1.QuerySortOption_builder{
					Field:    search.SubjectKind.String(),
					Reversed: true,
				}.Build(),
			},
			expected: []*storage.Subject{
				storage.Subject_builder{
					Name: "abc",
					Kind: storage.SubjectKind_USER,
				}.Build(),
				storage.Subject_builder{
					Name: "abc",
					Kind: storage.SubjectKind_GROUP,
				}.Build(),
				storage.Subject_builder{
					Name: "def",
					Kind: storage.SubjectKind_USER,
				}.Build(),
				storage.Subject_builder{
					Name: "def",
					Kind: storage.SubjectKind_GROUP,
				}.Build(),
				storage.Subject_builder{
					Name: "hij",
					Kind: storage.SubjectKind_SERVICE_ACCOUNT,
				}.Build(),
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			qp := &v1.QueryPagination{}
			qp.SetSortOptions(c.sortOptions)
			q := &v1.Query{}
			q.SetPagination(qp)

			testSubjects := getSubjects()
			err := sortSubjects(q, testSubjects)
			if c.hasError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			protoassert.SlicesEqual(t, c.expected, testSubjects)
		})
	}
}

func TestGetFiltered(t *testing.T) {
	cases := []struct {
		name             string
		query            *v1.Query
		subjects         []*storage.Subject
		expectedSubjects []*storage.Subject
	}{
		{
			name: "name search",
			subjects: []*storage.Subject{
				storage.Subject_builder{
					Name: "sub1",
					Kind: storage.SubjectKind_GROUP,
				}.Build(),
				storage.Subject_builder{
					Name: "sub2",
					Kind: storage.SubjectKind_USER,
				}.Build(),
			},
			query: search.NewQueryBuilder().AddStrings(search.SubjectName, "sub1").ProtoQuery(),
			expectedSubjects: []*storage.Subject{
				storage.Subject_builder{
					Name: "sub1",
					Kind: storage.SubjectKind_GROUP,
				}.Build(),
			},
		},
		{
			name: "kind search",
			subjects: []*storage.Subject{
				storage.Subject_builder{
					Name: "sub1",
					Kind: storage.SubjectKind_GROUP,
				}.Build(),
				storage.Subject_builder{
					Name: "sub2",
					Kind: storage.SubjectKind_USER,
				}.Build(),
			},
			query: search.NewQueryBuilder().AddStrings(search.SubjectKind, storage.SubjectKind_USER.String()).ProtoQuery(),
			expectedSubjects: []*storage.Subject{
				storage.Subject_builder{
					Name: "sub2",
					Kind: storage.SubjectKind_USER,
				}.Build(),
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			filteredSubjects, err := GetFilteredSubjects(c.query, c.subjects)
			require.NoError(t, err)
			protoassert.SlicesEqual(t, c.expectedSubjects, filteredSubjects)
		})
	}
}

func TestSubjectSearcher(t *testing.T) {
	suite.Run(t, new(SubjectSearcherTestSuite))
}

type testCase struct {
	desc             string
	query            *v1.Query
	expectedBindings []*storage.K8SRoleBinding
	expected         []search.Result
}

type SubjectSearcherTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller

	ctx               context.Context
	mockBindingsStore *roleBindingMocks.MockDataStore
	subjectSearcher   *SubjectSearcher
	testBindings      []*storage.K8SRoleBinding
}

func (s *SubjectSearcherTestSuite) SetupTest() {
	s.ctx = sac.WithAllAccess(context.Background())

	s.mockCtrl = gomock.NewController(s.T())
	s.mockBindingsStore = roleBindingMocks.NewMockDataStore(s.mockCtrl)
	s.subjectSearcher = NewSubjectSearcher(s.mockBindingsStore)
	s.testBindings = fixtures.GetMultipleK8sRoleBindings(3, 3)
	// For one binding, keep only the SERVICE_ACCOUNT kind subject
	s.testBindings[2].SetSubjects(s.testBindings[2].GetSubjects()[:1])
}

func (s *SubjectSearcherTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *SubjectSearcherTestSuite) TestSearcher() {
	for _, tc := range s.testCases() {
		s.T().Run(tc.desc, func(t *testing.T) {
			s.mockBindingsStore.EXPECT().SearchRawRoleBindings(s.ctx, tc.query).Times(3).Return(tc.expectedBindings, nil)
			results, err := s.subjectSearcher.Search(s.ctx, tc.query)
			s.NoError(err)
			s.ElementsMatch(tc.expected, results)

			count, err := s.subjectSearcher.Count(s.ctx, tc.query)
			s.NoError(err)
			s.Equal(len(tc.expected), count)

			v1SearchResults, err := s.subjectSearcher.SearchSubjects(s.ctx, tc.query)
			s.NoError(err)
			protoassert.ElementsMatch(s.T(), s.resultsToV1SearchResults(tc.expected), v1SearchResults)
		})
	}
}

func (s *SubjectSearcherTestSuite) testCases() []testCase {
	return []testCase{
		{
			desc:             "Search by subject name",
			query:            search.NewQueryBuilder().AddStrings(search.SubjectName, s.testBindings[0].GetSubjects()[1].GetName()).ProtoQuery(),
			expectedBindings: []*storage.K8SRoleBinding{s.testBindings[0]},
			expected: []search.Result{
				{
					ID: s.testBindings[0].GetSubjects()[1].GetName(),
					Matches: map[string][]string{
						"subject.name": {s.testBindings[0].GetSubjects()[1].GetName()},
					},
				},
			},
		},
		{
			desc:             "Search by subject kind",
			query:            search.NewQueryBuilder().AddStrings(search.SubjectKind, "").ProtoQuery(),
			expectedBindings: []*storage.K8SRoleBinding{s.testBindings[0], s.testBindings[1], s.testBindings[2]},
			expected: []search.Result{
				{
					ID: s.testBindings[0].GetSubjects()[1].GetName(),
					Matches: map[string][]string{
						"subject.kind": {"user"},
					},
				},
				{
					ID: s.testBindings[0].GetSubjects()[2].GetName(),
					Matches: map[string][]string{
						"subject.kind": {"group"},
					},
				},
				{
					ID: s.testBindings[1].GetSubjects()[1].GetName(),
					Matches: map[string][]string{
						"subject.kind": {"user"},
					},
				},
				{
					ID: s.testBindings[1].GetSubjects()[2].GetName(),
					Matches: map[string][]string{
						"subject.kind": {"group"},
					},
				},
			},
		},
		{
			desc:             "Search by cluster name",
			query:            search.NewQueryBuilder().AddStrings(search.Cluster, s.testBindings[1].GetClusterName()).ProtoQuery(),
			expectedBindings: []*storage.K8SRoleBinding{s.testBindings[1]},
			expected: []search.Result{
				{
					ID: s.testBindings[1].GetSubjects()[1].GetName(),
					Matches: map[string][]string{
						"k8srolebinding.cluster_name": {s.testBindings[1].GetClusterName()},
					},
				},
				{
					ID: s.testBindings[1].GetSubjects()[2].GetName(),
					Matches: map[string][]string{
						"k8srolebinding.cluster_name": {s.testBindings[1].GetClusterName()},
					},
				},
			},
		},
		{
			desc:             "Search by cluster role",
			query:            search.NewQueryBuilder().AddStrings(search.ClusterRole, "tr").ProtoQuery(),
			expectedBindings: []*storage.K8SRoleBinding{s.testBindings[0], s.testBindings[2]},
			expected: []search.Result{
				{
					ID: s.testBindings[0].GetSubjects()[1].GetName(),
					Matches: map[string][]string{
						"k8srolebinding.cluster_role": {"true"},
					},
				},
				{
					ID: s.testBindings[0].GetSubjects()[2].GetName(),
					Matches: map[string][]string{
						"k8srolebinding.cluster_role": {"true"},
					},
				},
			},
		},
		{
			desc:             "Search by an unsupported field",
			query:            search.NewQueryBuilder().AddStrings(search.DeploymentName, "d1").ProtoQuery(),
			expectedBindings: []*storage.K8SRoleBinding{},
			expected:         []search.Result{},
		},
	}
}

func (s *SubjectSearcherTestSuite) resultsToV1SearchResults(results []search.Result) []*v1.SearchResult {
	v1SearchResults := make([]*v1.SearchResult, 0, len(results))
	for _, r := range results {
		sr := &v1.SearchResult{}
		sr.SetId(r.ID)
		sr.SetName(r.ID)
		sr.SetCategory(v1.SearchCategory_SUBJECTS)
		v1SearchResults = append(v1SearchResults, sr)
	}
	return v1SearchResults
}
