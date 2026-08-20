package set

import (
	"slices"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertFrozenSetContainsExactly(t *testing.T, fs FrozenStringSet, elements ...string) {
	a := assert.New(t)
	for _, elem := range elements {
		a.True(fs.Contains(elem))
	}
	a.Equal(len(elements), fs.Cardinality())
	a.Equal(len(elements) == 0, fs.IsEmpty())

	falseCases := []string{"BLAH", "blah", "BLACK", "SheeP"}
	for _, elem := range falseCases {
		if slices.Index(falseCases, elem) == -1 {
			a.False(fs.Contains(elem))
		}
	}
	a.ElementsMatch(fs.AsSlice(), elements)

	slices.Sort(elements)
	a.Equal(elements, fs.AsSortedSlice(func(i, j string) bool {
		return i < j
	}))

	sort.Slice(elements, func(i, j int) bool {
		return elements[i] > elements[j]
	})
	a.Equal(elements, fs.AsSortedSlice(func(i, j string) bool {
		return i > j
	}))
}

func TestFrozenStringSet(t *testing.T) {
	elements := []string{"a", "bcd"}
	fs := NewFrozenSet(elements...)
	assertFrozenSetContainsExactly(t, fs, elements...)

	emptyFS := NewFrozenSet[string]()
	assertFrozenSetContainsExactly(t, emptyFS)
}

func TestFrozenSetAll(t *testing.T) {
	tests := map[string]struct {
		elements []string
	}{
		"empty set":         {elements: nil},
		"single element":    {elements: []string{"x"}},
		"multiple elements": {elements: []string{"a", "b", "c"}},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			fs := NewFrozenSet(tc.elements...)
			var collected []string
			for elem := range fs.All() {
				collected = append(collected, elem)
			}
			assert.ElementsMatch(t, fs.AsSlice(), collected)
		})
	}
}

func TestFrozenSetAllEarlyBreak(t *testing.T) {
	fs := NewFrozenSet("a", "b", "c", "d", "e")
	count := 0
	for range fs.All() {
		count++
		if count == 2 {
			break
		}
	}
	assert.Equal(t, 2, count)
}

func TestFrozenStringSetAfterFreeze(t *testing.T) {
	set := NewSet[string]()
	set.Add("a")
	set.Add("apple")
	fs := set.Freeze()

	assertFrozenSetContainsExactly(t, fs, "a", "apple")

	emptySet := NewSet[string]()
	emptyFS := emptySet.Freeze()
	assertFrozenSetContainsExactly(t, emptyFS)
}
