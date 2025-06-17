package tracker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_orderedValues(t *testing.T) {
	ov := orderedValues{
		{int: 2, string: "b"},
		{int: 1, string: "a"},
		{int: 3, string: "c"},
	}
	assert.Equal(t, "a,b,c", ov.join(','))

	// Test with duplicate ints.
	ov2 := orderedValues{
		{int: 2, string: "b"},
		{int: 1, string: "a"},
		{int: 2, string: "bb"},
	}
	joined := ov2.join('-')
	assert.True(t, joined == "a-b-bb" || joined == "a-bb-b", joined)

	// Test with empty slice.
	assert.Empty(t, orderedValues{}.join(','))

	// Test with single element
	ov4 := orderedValues{{int: 5, string: "s"}}
	assert.Equal(t, "s", ov4.join('|'))
}
