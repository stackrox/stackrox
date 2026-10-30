package sortfields

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRiskPriorityClassificationSortExpandsToSortOrderAndScore(t *testing.T) {
	mapper, ok := SortFieldsMap[search.RiskPriorityClassification]
	require.True(t, ok)

	asc := mapper(&v1.QuerySortOption{Field: search.RiskPriorityClassification.String()})
	require.Len(t, asc, 2)
	assert.Equal(t, search.RiskPrioritySortOrder.String(), asc[0].GetField())
	assert.False(t, asc[0].GetReversed())
	assert.Equal(t, search.DeploymentRiskScore.String(), asc[1].GetField())
	assert.True(t, asc[1].GetReversed())

	desc := mapper(&v1.QuerySortOption{Field: search.RiskPriorityClassification.String(), Reversed: true})
	require.Len(t, desc, 2)
	assert.Equal(t, search.RiskPrioritySortOrder.String(), desc[0].GetField())
	assert.True(t, desc[0].GetReversed())
	assert.Equal(t, search.DeploymentRiskScore.String(), desc[1].GetField())
	assert.False(t, desc[1].GetReversed())
}
