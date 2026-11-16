package postgres

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeploymentSchemaHasRiskComparisonSortFields(t *testing.T) {
	_, ok := pkgSchema.DeploymentsSchema.OptionsMap.Get(search.DeploymentRiskScore.String())
	assert.True(t, ok)
	_, ok = pkgSchema.DeploymentsSchema.OptionsMap.Get(search.ProposedDeploymentRiskScore.String())
	assert.True(t, ok)
	_, ok = pkgSchema.DeploymentsSchema.OptionsMap.Get(search.RiskPrioritySortOrder.String())
	assert.True(t, ok)
	_, ok = pkgSchema.DeploymentsSchema.OptionsMap.Get(search.RiskActivitySortOrder.String())
	assert.True(t, ok)
	_, ok = pkgSchema.DeploymentsSchema.OptionsMap.Get(search.RiskExposureSortOrder.String())
	assert.True(t, ok)
	_, ok = pkgSchema.DeploymentsSchema.OptionsMap.Get(search.RiskPostureSortOrder.String())
	assert.True(t, ok)
}

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
	assert.Equal(t, search.ProposedDeploymentRiskScore.String(), prepared.GetPagination().GetSortOptions()[1].GetField())
	assert.True(t, prepared.GetPagination().GetSortOptions()[1].GetReversed())

	// The preparation must not mutate the caller-owned query.
	require.Len(t, query.GetPagination().GetSortOptions(), 1)
	assert.Equal(t, search.RiskPriorityClassification.String(), query.GetPagination().GetSortOptions()[0].GetField())
}

func TestPrepareListDeploymentQueryExpandsRiskDimensionSorts(t *testing.T) {
	testCases := map[search.FieldLabel]search.FieldLabel{
		search.RiskActivityClassification: search.RiskActivitySortOrder,
		search.RiskExposureClassification: search.RiskExposureSortOrder,
		search.RiskPostureClassification:  search.RiskPostureSortOrder,
	}
	for classificationField, sortOrderField := range testCases {
		t.Run(classificationField.String(), func(t *testing.T) {
			query := &v1.Query{Pagination: &v1.QueryPagination{SortOptions: []*v1.QuerySortOption{{Field: classificationField.String()}}}}
			prepared := prepareListDeploymentQuery(query)
			require.NotNil(t, prepared)
			require.Len(t, prepared.GetPagination().GetSortOptions(), 1)
			assert.Equal(t, sortOrderField.String(), prepared.GetPagination().GetSortOptions()[0].GetField())
			assert.False(t, prepared.GetPagination().GetSortOptions()[0].GetReversed())
		})
	}
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
	assert.Equal(t, search.ProposedDeploymentRiskScore.String(), prepared.GetPagination().GetSortOptions()[1].GetField())
	assert.False(t, prepared.GetPagination().GetSortOptions()[1].GetReversed())
}
