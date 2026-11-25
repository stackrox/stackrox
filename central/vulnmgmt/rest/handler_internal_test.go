package rest

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
)

func TestExceptionStatusesForQuery(t *testing.T) {
	cases := map[string]struct {
		state    string
		expected []string
	}{
		"observed": {state: "OBSERVED", expected: []string{"PENDING"}},
		"deferred": {state: "DEFERRED", expected: []string{"APPROVED_PENDING_UPDATE"}},
		"none":     {state: "", expected: []string{"APPROVED_PENDING_UPDATE"}},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			qb := search.NewQueryBuilder()
			if tc.state != "" {
				qb = qb.AddExactMatches(search.VulnerabilityState, tc.state)
			}
			assert.Equal(t, tc.expected, exceptionStatusesForQuery(qb.ProtoQuery()))
		})
	}
}

func TestQueryWithoutTraversal(t *testing.T) {
	q := search.NewQueryBuilder().
		AddExactMatches(search.CVE, "CVE-2024-0001").
		WithPagination(search.NewPagination().Limit(20)).
		ProtoQuery()
	q.Selects = []*v1.QuerySelect{{}}
	q.GroupBy = &v1.QueryGroupBy{Fields: []string{search.CVE.String()}}

	got := queryWithoutTraversal(q)

	assert.Nil(t, got.GetSelects())
	assert.Nil(t, got.GetGroupBy())
	assert.Nil(t, got.GetPagination())
	assert.NotNil(t, got.GetQuery(), "filters must be preserved")
}

func TestBuildPageQueryRestrictsSorts(t *testing.T) {
	q := search.NewQueryBuilder().
		WithPagination(
			search.NewPagination().
				Limit(20).
				AddSortOption(search.NewSortOption(search.CriticalSeverityCount).Reversed(true)),
		).
		ProtoQuery()

	got := buildPageQuery(q, search.ImageID, search.ImageName, search.ImageName, search.ImageOS)

	assert.Equal(t, search.ImageName.String(), got.GetPagination().GetSortOptions()[0].GetField())
	assert.Contains(t, got.GetGroupBy().GetFields(), search.ImageID.String())
	assert.Contains(t, got.GetGroupBy().GetFields(), search.ImageName.String())
	assert.Len(t, got.GetSelects(), 2)
}

func TestBuildPageQueryKeepsAllowedSorts(t *testing.T) {
	q := search.NewQueryBuilder().
		WithPagination(
			search.NewPagination().
				Limit(20).
				AddSortOption(search.NewSortOption(search.ImageOS).Reversed(true)),
		).
		ProtoQuery()

	got := buildPageQuery(q, search.ImageID, search.ImageName, search.ImageName, search.ImageOS)

	assert.Equal(t, search.ImageOS.String(), got.GetPagination().GetSortOptions()[0].GetField())
	assert.Contains(t, got.GetGroupBy().GetFields(), search.ImageOS.String())
}
