package utils

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBatchProcess(t *testing.T) {
	type Run struct {
		name     string
		input    []int
		expected [][]int
	}
	runs := []Run{
		{
			name:     "partial",
			input:    []int{1, 2, 3, 4, 5, 6, 7, 8},
			expected: [][]int{{1, 2, 3}, {4, 5, 6}, {7, 8}},
		},
		{
			name:     "boundary",
			input:    []int{1, 2, 3, 4, 5, 6, 7, 8, 9},
			expected: [][]int{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}},
		},
		{
			name:     "empty",
			input:    []int{},
			expected: [][]int{},
		},
	}

	for _, run := range runs {
		t.Run(run.name, func(t *testing.T) {
			actual := make([][]int, 0)
			err := BatchProcess(run.input, 3, func(set []int) error {
				actual = append(actual, set)
				return nil
			})
			assert.Equal(t, nil, err)
			assert.EqualValues(t, run.expected, actual)
		})
	}

	err := BatchProcess([]int{1}, 3, func(set []int) error {
		return errors.New("fail")
	})

	assert.NotEqual(t, nil, err)
}
