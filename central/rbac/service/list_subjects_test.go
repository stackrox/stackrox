package service

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
