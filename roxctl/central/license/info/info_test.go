package info

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatList(t *testing.T) {
	tests := []struct {
		title    string
		items    []string
		expected string
	}{
		{
			title: "empty list",
		},
		{
			title:    "one item",
			items:    []string{"item1"},
			expected: "Only item1",
		},
		{
			title:    "two items",
			items:    []string{"item1", "item2"},
			expected: "Either item1 or item2",
		},
		{
			title:    "multiple items",
			items:    []string{"item1", "item2", "item3"},
			expected: "Any of item1, item2, or item3",
		},
	}

	for index, test := range tests {
		name := fmt.Sprintf("%d %s", index+1, test.title)
		t.Run(name, func(t *testing.T) {
			actual := formatList(test.items)
			assert.Equal(t, test.expected, actual)
		})
	}
}
