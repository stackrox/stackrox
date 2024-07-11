package sliceutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnique(t *testing.T) {
	assert.Equal(t, []string{"a", "b", "c", "d"}, Unique([]string{"a", "b", "c", "a", "d", "d"}))
	assert.Equal(t, []string{"a", "b", "c", "d"}, Unique([]string{"a", "b", "c", "d"}))
	assert.Equal(t, []string{"a", "b"}, Unique([]string{"a", "a", "b", "a", "b"}))
	assert.Equal(t, []string{}, Unique([]string{}))
}
