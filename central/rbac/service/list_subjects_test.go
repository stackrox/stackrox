package service

import (
	"context"
	"testing"

	roleBindingMocks "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func getSubjects() []*storage.Subject {
	return []*storage.Subject{
		{
			Name: "def",
			Kind: storage.SubjectKind_GROUP,
		},
		{
			Name: "def",
			Kind: storage.SubjectKind_USER,
		},
		{
			Name: "hij",
			Kind: storage.SubjectKind_SERVICE_ACCOUNT,
		},
		{
			Name: "abc",
			Kind: storage.SubjectKind_USER,
		},
		{
			Name: "abc",
			Kind: storage.SubjectKind_GROUP,
		},
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
				{
					Field:    search.SubjectName.String(),
					Reversed: false,
				},
			},
			expected: []*storage.Subject{
				{
					Name: "abc",
					Kind: storage.SubjectKind_USER,
				},
				{
					Name: "abc",
					Kind: storage.SubjectKind_GROUP,
				},
				{
					Name: "def",
					Kind: storage.SubjectKind_GROUP,
				},
				{
					Name: "def",
					Kind: storage.SubjectKind_USER,
				},
				{
					Name: "hij",
					Kind: storage.SubjectKind_SERVICE_ACCOUNT,
				},
			},
		},
		{
			name: "subject sort - reversed",
			sortOptions: []*v1.QuerySortOption{
				{
					Field:    search.SubjectName.String(),
					Reversed: true,
				},
			},
			expected: []*storage.Subject{
				{
					Name: "hij",
					Kind: storage.SubjectKind_SERVICE_ACCOUNT,
				},
				{
					Name: "def",
					Kind: storage.SubjectKind_GROUP,
				},
				{
					Name: "def",
					Kind: storage.SubjectKind_USER,
				},
				{
					Name: "abc",
					Kind: storage.SubjectKind_USER,
				},
				{
					Name: "abc",
					Kind: storage.SubjectKind_GROUP,
				},
			},
		},
		{
			name: "subject sort - kind sort",
			sortOptions: []*v1.QuerySortOption{
				{
					Field: search.SubjectName.String(),
				},
				{
					Field: search.SubjectKind.String(),
				},
			},
			expected: []*storage.Subject{
				{
					Name: "abc",
					Kind: storage.SubjectKind_GROUP,
				},
				{
					Name: "abc",
					Kind: storage.SubjectKind_USER,
				},
				{
					Name: "def",
					Kind: storage.SubjectKind_GROUP,
				},
				{
					Name: "def",
					Kind: storage.SubjectKind_USER,
				},
				{
					Name: "hij",
					Kind: storage.SubjectKind_SERVICE_ACCOUNT,
				},
			},
		},
		{
			name: "subject sort - kind sort",
			sortOptions: []*v1.QuerySortOption{
				{
					Field: search.SubjectName.String(),
				},
				{
					Field:    search.SubjectKind.String(),
					Reversed: true,
				},
			},
			expected: []*storage.Subject{
				{
					Name: "abc",
					Kind: storage.SubjectKind_USER,
				},
				{
					Name: "abc",
					Kind: storage.SubjectKind_GROUP,
				},
				{
					Name: "def",
					Kind: storage.SubjectKind_USER,
				},
				{
					Name: "def",
					Kind: storage.SubjectKind_GROUP,
				},
				{
					Name: "hij",
					Kind: storage.SubjectKind_SERVICE_ACCOUNT,
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			q := &v1.Query{
				Pagination: &v1.QueryPagination{
					SortOptions: c.sortOptions,
				},
			}

			testSubjects := getSubjects()
			err := sortSubjects(q, testSubjects)
			if c.hasError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, c.expected, testSubjects)
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
				{
					Name: "sub1",
					Kind: storage.SubjectKind_GROUP,
				},
				{
					Name: "sub2",
					Kind: storage.SubjectKind_USER,
				},
			},
			query: search.NewQueryBuilder().AddStrings(search.SubjectName, "sub1").ProtoQuery(),
			expectedSubjects: []*storage.Subject{
				{
					Name: "sub1",
					Kind: storage.SubjectKind_GROUP,
				},
			},
		},
		{
			name: "kind search",
			subjects: []*storage.Subject{
				{
					Name: "sub1",
					Kind: storage.SubjectKind_GROUP,
				},
				{
					Name: "sub2",
					Kind: storage.SubjectKind_USER,
				},
			},
			query: search.NewQueryBuilder().AddStrings(search.SubjectKind, storage.SubjectKind_USER.String()).ProtoQuery(),
			expectedSubjects: []*storage.Subject{
				{
					Name: "sub2",
					Kind: storage.SubjectKind_USER,
				},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			filteredSubjects, err := GetFilteredSubjects(c.query, c.subjects)
			require.NoError(t, err)
			assert.Equal(t, c.expectedSubjects, filteredSubjects)
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
	s.testBindings[2].Subjects = s.testBindings[2].Subjects[:1]
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
			s.ElementsMatch(tc.expected, s.standardizeResults(results))

			count, err := s.subjectSearcher.Count(s.ctx, tc.query)
			s.NoError(err)
			s.Equal(len(tc.expected), count)

			v1SearchResults, err := s.subjectSearcher.SearchSubjects(s.ctx, tc.query)
			s.NoError(err)
			s.ElementsMatch(s.resultsToV1SearchResults(tc.expected), v1SearchResults)
		})
	}
}

func (s *SubjectSearcherTestSuite) testCases() []testCase {
	return []testCase{
		{
			desc:             "Search by subject name",
			query:            search.NewQueryBuilder().AddStrings(search.SubjectName, s.testBindings[0].Subjects[1].Name).ProtoQuery(),
			expectedBindings: []*storage.K8SRoleBinding{s.testBindings[0]},
			expected: []search.Result{
				{
					ID: s.testBindings[0].Subjects[1].Name,
					Matches: map[string][]string{
						"subject.name": {s.testBindings[0].Subjects[1].Name},
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
					ID: s.testBindings[0].Subjects[1].Name,
					Matches: map[string][]string{
						"subject.kind": {"user"},
					},
				},
				{
					ID: s.testBindings[0].Subjects[2].Name,
					Matches: map[string][]string{
						"subject.kind": {"group"},
					},
				},
				{
					ID: s.testBindings[1].Subjects[1].Name,
					Matches: map[string][]string{
						"subject.kind": {"user"},
					},
				},
				{
					ID: s.testBindings[1].Subjects[2].Name,
					Matches: map[string][]string{
						"subject.kind": {"group"},
					},
				},
			},
		},
		{
			desc:             "Search by cluster name",
			query:            search.NewQueryBuilder().AddStrings(search.Cluster, s.testBindings[1].ClusterName).ProtoQuery(),
			expectedBindings: []*storage.K8SRoleBinding{s.testBindings[1]},
			expected: []search.Result{
				{
					ID: s.testBindings[1].Subjects[1].Name,
					Matches: map[string][]string{
						"k8srolebinding.cluster_name": {s.testBindings[1].ClusterName},
					},
				},
				{
					ID: s.testBindings[1].Subjects[2].Name,
					Matches: map[string][]string{
						"k8srolebinding.cluster_name": {s.testBindings[1].ClusterName},
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
					ID: s.testBindings[0].Subjects[1].Name,
					Matches: map[string][]string{
						"k8srolebinding.cluster_role": {"true"},
					},
				},
				{
					ID: s.testBindings[0].Subjects[2].Name,
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
		v1SearchResults = append(v1SearchResults, &v1.SearchResult{
			Id:       r.ID,
			Name:     r.ID,
			Category: v1.SearchCategory_SUBJECTS,
		})
	}
	return v1SearchResults
}

func (s *SubjectSearcherTestSuite) standardizeResults(results []search.Result) []search.Result {
	newResults := make([]search.Result, 0, len(results))
	for _, r := range results {
		r.Fields = nil
		newResults = append(newResults, r)
	}
	return newResults
}
