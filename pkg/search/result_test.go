package search

import (
	"fmt"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createResults(num int) []Result {
	results := make([]Result, num)
	for i := 0; i < num; i++ {
		results[i].ID = fmt.Sprintf("%d", i)
	}
	return results
}

func TestRemoveMissingResults_None(t *testing.T) {

	results := createResults(8)
	origIDs := ResultsToIDs(results)
	filtered := RemoveMissingResults(results, []int{})
	assert.Equal(t, origIDs, ResultsToIDs(filtered))
}

func TestRemoveMissingResults_Some(t *testing.T) {

	results := createResults(8)
	filtered := RemoveMissingResults(results, []int{2, 3, 7})
	assert.Equal(t, []string{"0", "1", "4", "5", "6"}, ResultsToIDs(filtered))
}

func TestRemoveMissingResults_All(t *testing.T) {

	results := createResults(8)
	filtered := RemoveMissingResults(results, []int{0, 1, 2, 3, 4, 5, 6, 7})
	assert.Empty(t, filtered)
}

// TestResultsToSearchResultProtos tests the conversion from search Results to SearchResult protos
func TestResultsToSearchResultProtos(t *testing.T) {
	// Create test results with field values
	results := []Result{
		{
			ID:       "id1",
			Name:     "test-deployment",
			Location: "cluster1/namespace1/test-deployment",
			Score:    0.95,
			Matches: map[string][]string{
				"Deployment Name": {"test-deployment"},
			},
			FieldValues: map[string]interface{}{
				"Deployment Name": "test-deployment",
				"Cluster":         "cluster1",
			},
		},
		{
			ID:       "id2",
			Name:     "prod-deployment",
			Location: "cluster2/namespace2/prod-deployment",
			Score:    0.87,
			Matches: map[string][]string{
				"Deployment Name": {"prod-deployment"},
			},
			FieldValues: map[string]interface{}{
				"Deployment Name": "prod-deployment",
				"Cluster":         "cluster2",
			},
		},
	}

	// Create a test converter
	converter := &DefaultSearchResultConverter{
		Category: v1.SearchCategory_DEPLOYMENTS,
	}

	// Convert to proto
	protos := ResultsToSearchResultProtos(results, converter)

	// Verify the conversion
	require.Len(t, protos, 2)

	assert.Equal(t, "id1", protos[0].Id)
	assert.Equal(t, "test-deployment", protos[0].Name)
	assert.Equal(t, "cluster1/namespace1/test-deployment", protos[0].Location)
	assert.Equal(t, 0.95, protos[0].Score)
	assert.Equal(t, v1.SearchCategory_DEPLOYMENTS, protos[0].Category)
	assert.NotNil(t, protos[0].FieldToMatches)
	assert.Len(t, protos[0].FieldToMatches, 1)

	assert.Equal(t, "id2", protos[1].Id)
	assert.Equal(t, "prod-deployment", protos[1].Name)
	assert.Equal(t, "cluster2/namespace2/prod-deployment", protos[1].Location)
	assert.Equal(t, 0.87, protos[1].Score)
	assert.Equal(t, v1.SearchCategory_DEPLOYMENTS, protos[1].Category)
}

// TestDefaultSearchResultConverter tests the DefaultSearchResultConverter implementation
func TestDefaultSearchResultConverter(t *testing.T) {
	converter := &DefaultSearchResultConverter{
		Category: v1.SearchCategory_IMAGES,
	}

	result := &Result{
		Name:     "nginx:latest",
		Location: "registry.io/nginx:latest",
	}

	assert.Equal(t, "nginx:latest", converter.BuildName(result))
	assert.Equal(t, "registry.io/nginx:latest", converter.BuildLocation(result))
	assert.Equal(t, v1.SearchCategory_IMAGES, converter.GetCategory())
}

// TestResultsWithFieldValues tests that field values are properly stored and retrieved
func TestResultsWithFieldValues(t *testing.T) {
	result := Result{
		ID: "test-id",
		FieldValues: map[string]interface{}{
			"Name":      "test-name",
			"Registry":  "docker.io",
			"Count":     42,
			"IsScanned": true,
		},
	}

	// Verify field values are accessible
	assert.Equal(t, "test-name", result.FieldValues["Name"])
	assert.Equal(t, "docker.io", result.FieldValues["Registry"])
	assert.Equal(t, 42, result.FieldValues["Count"])
	assert.Equal(t, true, result.FieldValues["IsScanned"])
}
