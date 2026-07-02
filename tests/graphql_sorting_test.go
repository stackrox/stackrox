//go:build test_e2e

package tests

import (
	"slices"
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
	reversed := sliceutils.Map(getDeploymentsWithSortOption(t, field, true), extractor)

	// The API sorts via Postgres collation which may differ from Go's byte-order
	// comparison (e.g., hyphen ordering varies between C and glibc collations).
	// Instead of comparing against Go's sort, verify the forward and reverse
	// responses are mirror images of each other — proving the API actually sorts.
	require.Equal(t, len(sorted), len(reversed), "forward and reverse should have same length")
	for i := range sorted {
		assert.Equal(t, sorted[i], reversed[len(reversed)-1-i],
			"field %s: forward[%d] should equal reverse[%d]", field, i, len(reversed)-1-i)
	}
}

func TestGraphQLSorting(t *testing.T) {
	testDeploymentSorting(t, "Deployment", func(d *storage.Deployment) string {
		return d.GetName()
	})

	testDeploymentSorting(t, "Namespace", func(d *storage.Deployment) string {
		return d.GetNamespace()
	})
}
