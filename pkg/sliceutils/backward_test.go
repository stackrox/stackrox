package sliceutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBackward(t *testing.T) {
	testArray := []string{"Alice", "Bob", "Vera"}
	expectedIndices := []int{2, 1, 0}
	expectedValues := []string{"Vera", "Bob", "Alice"}

	indices := make([]int, 0, len(testArray))
	values := make([]string, 0, len(testArray))
	for ix, val := range Backward(testArray) {
		indices = append(indices, ix)
		values = append(values, val)
	}

	assert.Equal(t, expectedIndices, indices)
	assert.Equal(t, expectedValues, values)
}
