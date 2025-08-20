package sliceutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestContains
func TestContains(t *testing.T) {
	input := []int{1, 2, 3, 4, 5}

	assert.Equal(t, true, Contains(input, 1))
	assert.Equal(t, false, Contains(input, 6))

}
