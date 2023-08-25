package set

import (
	"sort"
	"testing"

	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stretchr/testify/assert"
)

func assertSetContainsExactly(t *testing.T, set StringSet, elements ...string) {
	a := assert.New(t)
	for _, elem := range elements {
		a.True(set.Contains(elem))
	}
	a.Equal(len(elements), set.Cardinality())
	a.Equal(len(elements) == 0, set.IsEmpty())

	falseCases := []string{"BLAH", "blah", "BLACK", "SheeP"}
	for _, elem := range falseCases {
		if sliceutils.Find(falseCases, elem) == -1 {
			a.False(set.Contains(elem))
		}
	}
	a.ElementsMatch(set.AsSlice(), elements)

	sort.Strings(elements)
	a.Equal(elements, set.AsSortedSlice(func(i, j string) bool {
		return i < j
	}))

	sort.Slice(elements, func(i, j int) bool {
		return elements[i] > elements[j]
	})
	a.Equal(elements, set.AsSortedSlice(func(i, j string) bool {
		return i > j
	}))
}

func TestAddMatching(t *testing.T) {
	a := assert.New(t)

	var set StringSet
	a.False(set.AddMatching(func(s string) bool {
		return s == "matching"
	}, "1", "2"))
	a.Nil(set)

	a.True(set.AddMatching(func(s string) bool {
		return s != ""
	}, "1", "2"))
	a.NotNil(set)
	a.ElementsMatch([]string{"1", "2"}, set.AsSlice())

	a.False(set.AddMatching(func(s string) bool {
		return s != ""
	}, "1", "2"))
	a.NotNil(set)
	a.ElementsMatch([]string{"1", "2"}, set.AsSlice())
}

func TestStringSet(t *testing.T) {
	elements := []string{"a", "bcd"}
	set := NewSet(elements[0])
	assertSetContainsExactly(t, set, elements[0])
	assert.Equal(t, elements[0], set.GetArbitraryElem())
	set.AddAll(elements[1:]...)
	assertSetContainsExactly(t, set, elements...)

	assert.True(t, set.Add("foo"))
	assert.False(t, set.Add("bcd"))
	assert.True(t, set.AddAll("bar", "baz"))

	assertSetContainsExactly(t, set, "a", "bcd", "foo", "bar", "baz")

	assert.True(t, set.Remove("a"))
	assert.False(t, set.Remove("b"))
	assert.True(t, set.RemoveAll("bcd", "qux"))

	assertSetContainsExactly(t, set, "foo", "bar", "baz")

	emptyFS := NewSet[string]()
	assertSetContainsExactly(t, emptyFS)
	assert.Equal(t, "", emptyFS.GetArbitraryElem())
}

func TestUnion(t *testing.T) {
	a := NewSet("a", "b", "c")
	b := NewSet("b", "c", "d")
	aPlusB := a.Union(b)

	assertSetContainsExactly(t, aPlusB, "a", "b", "c", "d")

	emptyPlusA := StringSet{}.Union(a)
	assertSetContainsExactly(t, emptyPlusA, a.AsSlice()...)
	aPlusEmpty := a.Union(StringSet{})
	assertSetContainsExactly(t, aPlusEmpty, a.AsSlice()...)

	// Test for no aliasing
	assert.True(t, emptyPlusA.Add("f"))
	assert.True(t, aPlusEmpty.Add("f"))

	assertSetContainsExactly(t, a, "a", "b", "c")
}

func TestIntersection(t *testing.T) {
	a := NewSet("a", "b", "c")
	b := NewSet("b", "c", "d")
	aAndB := a.Intersect(b)

	assertSetContainsExactly(t, aAndB, "b", "c")
	assert.True(t, a.Intersects(b))

	emptyAndA := StringSet{}.Intersect(a)
	assertSetContainsExactly(t, emptyAndA)
	assert.False(t, StringSet{}.Intersects(a))
	aAndEmpty := a.Intersect(StringSet{})
	assertSetContainsExactly(t, aAndEmpty)
	assert.False(t, a.Intersects(StringSet{}))

	// Test for no aliasing
	assert.True(t, emptyAndA.Add("f"))
	assert.True(t, aAndEmpty.Add("f"))

	assertSetContainsExactly(t, a, "a", "b", "c")
}

func TestDifference(t *testing.T) {
	a := NewSet("a", "b", "c")
	b := NewSet("b", "c", "d")
	aMinusB := a.Difference(b)

	assertSetContainsExactly(t, aMinusB, "a")

	emptyMinusA := StringSet{}.Difference(a)
	assertSetContainsExactly(t, emptyMinusA)
	aMinusEmpty := a.Difference(StringSet{})
	assertSetContainsExactly(t, aMinusEmpty, a.AsSlice()...)

	// Test for no aliasing
	assert.True(t, emptyMinusA.Add("f"))
	assert.True(t, aMinusEmpty.Add("f"))

	assertSetContainsExactly(t, a, "a", "b", "c")
}
