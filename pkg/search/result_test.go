package search

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createResults(num int) []Result {
	results := make([]Result, num)
	for i := 0; i < num; i++ {
		results[i].ID = fmt.Sprintf("%d", i)
	}
	return results
}

func TestRemoveMissingResults_None(t *testing.T) {
	t.Parallel()

	results := createResults(8)
	origIDs := ResultsToIDs(results)
	filtered := RemoveMissingResults(results, []int{})
	assert.Equal(t, origIDs, ResultsToIDs(filtered))
}

func TestRemoveMissingResults_Some(t *testing.T) {
	t.Parallel()

	results := createResults(8)
	filtered := RemoveMissingResults(results, []int{2, 3, 7})
	assert.Equal(t, []string{"0", "1", "4", "5", "6"}, ResultsToIDs(filtered))
}

func TestRemoveMissingResults_All(t *testing.T) {
	t.Parallel()

	results := createResults(8)
	filtered := RemoveMissingResults(results, []int{0, 1, 2, 3, 4, 5, 6, 7})
	assert.Empty(t, filtered)
}
