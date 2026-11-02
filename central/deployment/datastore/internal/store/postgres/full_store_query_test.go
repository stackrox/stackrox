package postgres

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareListDeploymentQueryExpandsRiskClassSort(t *testing.T) {
	query := &v1.Query{
		Pagination: &v1.QueryPagination{
			SortOptions: []*v1.QuerySortOption{
				{Field: search.RiskPriorityClassification.String()},
			},
		},
	}

	prepared := prepareListDeploymentQuery(query)
	require.NotNil(t, prepared)
	require.Len(t, prepared.GetPagination().GetSortOptions(), 2)

	assert.Equal(t, search.RiskPrioritySortOrder.String(), prepared.GetPagination().GetSortOptions()[0].GetField())
	assert.False(t, prepared.GetPagination().GetSortOptions()[0].GetReversed())
	assert.Equal(t, search.DeploymentRiskScore.String(), prepared.GetPagination().GetSortOptions()[1].GetField())
	assert.True(t, prepared.GetPagination().GetSortOptions()[1].GetReversed())

	// The preparation must not mutate the caller-owned query.
	require.Len(t, query.GetPagination().GetSortOptions(), 1)
	assert.Equal(t, search.RiskPriorityClassification.String(), query.GetPagination().GetSortOptions()[0].GetField())
}

func TestPrepareListDeploymentQueryExpandsReverseRiskClassSort(t *testing.T) {
	query := &v1.Query{
		Pagination: &v1.QueryPagination{
			SortOptions: []*v1.QuerySortOption{
				{Field: search.RiskPriorityClassification.String(), Reversed: true},
			},
		},
	}

	prepared := prepareListDeploymentQuery(query)
	require.NotNil(t, prepared)
	require.Len(t, prepared.GetPagination().GetSortOptions(), 2)

	assert.Equal(t, search.RiskPrioritySortOrder.String(), prepared.GetPagination().GetSortOptions()[0].GetField())
	assert.True(t, prepared.GetPagination().GetSortOptions()[0].GetReversed())
	assert.Equal(t, search.DeploymentRiskScore.String(), prepared.GetPagination().GetSortOptions()[1].GetField())
	assert.False(t, prepared.GetPagination().GetSortOptions()[1].GetReversed())
}
