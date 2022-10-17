package sliceutils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSelect(t *testing.T) {
	input := []string{"foo", "bar", "baz", "qux"}
	cases := []struct {
		indices  []int
		expected []string
		panics   bool
	}{
		{
			indices:  []int{1, 3},
			expected: []string{"bar", "qux"},
		},
		{
			indices:  []int{2, 0},
			expected: []string{"baz", "foo"},
		},
		{
			indices:  []int{},
			expected: nil,
		},
		{
			indices:  []int{0, 0, 1, 1, 2, 2, 3, 3},
			expected: []string{"foo", "foo", "bar", "bar", "baz", "baz", "qux", "qux"},
		},
		{
			indices: []int{0, -1},
			panics:  true,
		},
		{
			indices: []int{0, 4},
			panics:  true,
		},
	}

	for _, testCase := range cases {
		c := testCase
		t.Run(fmt.Sprintf("%v", c.indices), func(t *testing.T) {
			if c.panics {
				assert.Panics(t, func() {
					Select(input, c.indices...)
				})
			} else {
				result := Select(input, c.indices...)
				assert.Equal(t, c.expected, result)
			}
		})
	}
}
