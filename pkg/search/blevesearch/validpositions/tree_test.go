package validpositions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTreeAddingValues(t *testing.T) {
	values := [][]uint64{
		{1, 2, 3},
		{4, 5, 6},
		{7, 8, 9},
		{1, 4, 5},
	}
	tree := NewTreeFromValues(values...)
	for _, value := range values {
		assert.True(t, tree.Contains(value))
		assert.True(t, tree.Contains(value[:1]))
		assert.True(t, tree.Contains(value[:2]))
	}
	for _, nonExistentValue := range [][]uint64{
		{1, 2, 2},
		{2, 3},
		{4, 5, 9},
		{1, 2, 3, 4},
		{1, 4, 5, 6},
	} {
		assert.False(t, tree.Contains(nonExistentValue), "Shouldn't contain %+v", nonExistentValue)
	}
}

func TestTreeMerges(t *testing.T) {
	cases := []struct {
		name             string
		values           [][][]uint64
		shouldContain    [][]uint64
		shouldNotContain [][]uint64
		empty            bool
	}{
		{
			name: "One value, matches in both places",
			values: [][][]uint64{
				{
					{5, 6, 7},
				},
				{
					{5, 6, 7},
				},
			},
			shouldContain: [][]uint64{
				{5, 6, 7},
			},
		},
		{
			name: "One value, matches in all four places",
			values: [][][]uint64{
				{
					{5, 6, 7},
				},
				{
					{5, 6, 7},
				},
				{
					{5, 6, 7},
				},
				{
					{5, 6, 7},
				},
			},
			shouldContain: [][]uint64{
				{5, 6, 7},
			},
		},
		{
			name: "Some matching, some not matching",
			values: [][][]uint64{
				{
					{1, 2, 3},
					{5, 6, 7},
				},
				{
					{4, 5, 6},
					{5, 6, 7},
				},
				{
					{9, 10, 11},
					{5, 6, 7},
				},
				{
					{4, 5, 6},
					{5, 6, 7},
				},
			},
			shouldContain: [][]uint64{
				{5, 6, 7},
			},
			shouldNotContain: [][]uint64{
				{1, 2, 3},
				{4, 5, 6},
				{9, 10, 11},
			},
		},
		{
			name: "Different array lengths",
			values: [][][]uint64{
				{
					{1, 2},
					{3, 4},
				},
				{
					{1, 2, 3},
					{1, 2, 4},
					{5, 6, 7},
				},
			},
			shouldContain: [][]uint64{
				{1, 2, 3},
				{1, 2, 4},
				{1, 2},
			},
			shouldNotContain: [][]uint64{
				{3, 4},
				{5, 6, 7},
			},
		},
		{
			name: "Different array lengths, extreme case",
			values: [][][]uint64{
				{
					{1, 2, 3},
					{1, 2, 4},
					{5, 6, 7},
				},
				{
					{1, 2},
					{3, 4},
				},
				{
					{1},
					{2},
				},
				{
					{},
				},
			},
			shouldContain: [][]uint64{
				{1, 2, 3},
				{1, 2, 4},
				{1, 2},
			},
			shouldNotContain: [][]uint64{
				{2},
				{3},
				{3, 4},
				{5, 6, 7},
			},
		},
		{
			name: "No values, should be empty",
			values: [][][]uint64{
				{},
			},
			shouldNotContain: [][]uint64{
				{1, 2, 3},
				{1, 2, 4},
				{1, 2},
				{1},
				{5, 6, 7},
				{5},
				{5, 6},
			},
			empty: true,
		},
		{
			name: "No values for one, should be empty",
			values: [][][]uint64{
				{
					{1, 2, 3},
					{1, 2, 4},
					{5, 6, 7},
				},
				{},
			},
			shouldNotContain: [][]uint64{
				{1, 2, 3},
				{1, 2, 4},
				{1, 2},
				{1},
				{5, 6, 7},
				{5},
				{5, 6},
			},
			empty: true,
		},
		{
			name: "Different array lengths, no intersection",
			values: [][][]uint64{
				{
					{1, 2, 3},
					{1, 2, 4},
					{5, 6, 7},
				},
				{
					{3, 4},
				},
			},
			shouldNotContain: [][]uint64{
				{1, 2, 3},
				{1, 2, 4},
				{1, 2},
				{3, 4},
				{5, 6, 7},
			},
		},
		{
			name: "Multiple matches",
			values: [][][]uint64{
				{
					{1, 2, 3},
					{5, 6, 7},
				},
				{
					{4, 5, 6},
					{5, 6, 7},
					{1, 2, 3},
				},
				{
					{9, 10, 11},
					{5, 6, 7},
					{1, 2, 3},
				},
				{
					{4, 5, 6},
					{1, 2, 3},
					{5, 6, 7},
				},
			},
			shouldContain: [][]uint64{
				{1, 2, 3},
				{5, 6, 7},
			},
			shouldNotContain: [][]uint64{
				{4, 5, 6},
				{9, 10, 11},
			},
		},
		{
			name: "No intersection",
			values: [][][]uint64{
				{
					{5, 7},
					{6, 9},
				},
				{
					{5, 8},
					{6, 7},
				},
			},
			shouldNotContain: [][]uint64{
				{5, 7},
				{6, 9},
				{5, 8},
				{6, 7},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// We construct the tree by merging the values both in forward and reverse order, as an implicit test
			// that all the functions are symmetric, as they should be.
			for _, reverse := range []bool{false, true} {
				tree := constructTreeFromValues(c.values, reverse)
				for _, values := range c.shouldContain {
					for i := 0; i < len(values); i++ {
						assert.True(t, tree.Contains(values[:i+1]), "Should contain %+v", values)
					}
				}
				for _, values := range c.shouldNotContain {
					assert.False(t, tree.Contains(values), "Should not contain %+v", values)
				}
				assert.Equal(t, c.empty, tree.Empty(), "Tree's emptiness was not what we expected")
			}

		})
	}
}

func constructTreeFromValues(values [][][]uint64, reverse bool) *Tree {
	var tree *Tree
	for i := range values {
		var actualIndex int
		if reverse {
			actualIndex = len(values) - i - 1
		} else {
			actualIndex = i
		}
		if i == 0 {
			tree = NewTreeFromValues(values[actualIndex]...)
		} else {
			tree.Merge(NewTreeFromValues(values[actualIndex]...))
		}
	}
	return tree
}
