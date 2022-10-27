package tests

import (
	"sort"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getDeploymentsWithSortOption(t *testing.T, field string, reversed bool) []*storage.Deployment {
	var resp struct {
		Deployments []*storage.Deployment
	}
	makeGraphQLRequest(t, `
  		query deployments($query: String, $pagination: Pagination) {
  			deployments(query: $query, pagination: $pagination) {
				id
				name
				namespace
  			}
		}
	`, map[string]interface{}{
		"pagination": map[string]interface{}{
			"sortOption": map[string]interface{}{"field": field, "reversed": reversed},
		},
	}, &resp, timeout)

	require.True(t, len(resp.Deployments) > 0, "UNEXPECTED: no deployments found in API!")

	return resp.Deployments
}

func testDeploymentSorting(t *testing.T, field string, extractor func(d *storage.Deployment) string) {
	sorted := sliceutils.Map(getDeploymentsWithSortOption(t, field, false), extractor)
	assert.True(t, sort.StringsAreSorted(sorted), "field %s not sorted in response (got %v)", field, sorted)

	sortedReverse := sliceutils.Map(getDeploymentsWithSortOption(t, field, true), extractor)
	assert.True(t, sort.SliceIsSorted(sortedReverse, func(i, j int) bool {
		return sortedReverse[i] > sortedReverse[j]
	}), "field %s not sorted in reverse in response (got %v)", field, sortedReverse)
}

func TestGraphQLSorting(t *testing.T) {
	testDeploymentSorting(t, "Deployment", func(d *storage.Deployment) string {
		return d.GetName()
	})

	testDeploymentSorting(t, "Namespace", func(d *storage.Deployment) string {
		return d.GetNamespace()
	})
}
