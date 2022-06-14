package service

import (
	"testing"

	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
