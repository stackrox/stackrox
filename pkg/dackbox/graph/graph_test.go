package graph

import (
	"testing"

	"github.com/stackrox/rox/pkg/concurrency/sortedkeys"
	"github.com/stretchr/testify/assert"
)

func TestGraph(t *testing.T) {
	graph := NewGraph()

	graph.SetRefs([]byte("fromKey1"), sortedkeys.SortedKeys{[]byte("toKey1"), []byte("toKey2")})
	graph.SetRefs([]byte("fromKey2"), sortedkeys.SortedKeys{[]byte("toKey1"), []byte("toKey2")})
	graph.SetRefs([]byte("fromKey3"), sortedkeys.SortedKeys{[]byte("toKey1"), []byte("toKey2")})

	graph.SetRefs([]byte("fromKey2"), sortedkeys.SortedKeys{[]byte("toKey3"), []byte("toKey4")}) // resets
	graph.SetRefs([]byte("fromKey3"), sortedkeys.SortedKeys{[]byte("toKey1"), []byte("toKey2")}) // resets

	assert.Equal(t, [][]byte{[]byte("toKey1"), []byte("toKey2")}, graph.GetRefsFrom([]byte("fromKey1")))
	assert.Equal(t, [][]byte{[]byte("toKey3"), []byte("toKey4")}, graph.GetRefsFrom([]byte("fromKey2")))
	assert.Equal(t, [][]byte{[]byte("toKey1"), []byte("toKey2")}, graph.GetRefsFrom([]byte("fromKey3")))

	assert.Equal(t, [][]byte{[]byte("fromKey1"), []byte("fromKey3")}, graph.GetRefsTo([]byte("toKey1")))
	assert.Equal(t, [][]byte{[]byte("fromKey1"), []byte("fromKey3")}, graph.GetRefsTo([]byte("toKey2")))
	assert.Equal(t, [][]byte{[]byte("fromKey2")}, graph.GetRefsTo([]byte("toKey3")))
	assert.Equal(t, [][]byte{[]byte("fromKey2")}, graph.GetRefsTo([]byte("toKey4")))

	graph.DeleteRefsFrom([]byte("fromKey2")) // resets
	graph.DeleteRefsTo([]byte("toKey2"))     // resets

	assert.Equal(t, [][]byte{[]byte("toKey1")}, graph.GetRefsFrom([]byte("fromKey1")))
	assert.Equal(t, [][]byte(nil), graph.GetRefsFrom([]byte("fromKey2")))
	assert.Equal(t, [][]byte{[]byte("toKey1")}, graph.GetRefsFrom([]byte("fromKey3")))

	assert.Equal(t, [][]byte{[]byte("fromKey1"), []byte("fromKey3")}, graph.GetRefsTo([]byte("toKey1")))
	assert.Equal(t, [][]byte(nil), graph.GetRefsTo([]byte("toKey2")))
	assert.Equal(t, [][]byte(nil), graph.GetRefsTo([]byte("toKey3")))
	assert.Equal(t, [][]byte(nil), graph.GetRefsTo([]byte("toKey4")))
}

func TestFindFirstWithPrefix(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		keys     []string
		prefix   string
		expected int
	}{
		"no match": {
			keys:     []string{"foo:bar", "foo:baz", "foo:quux", "foo:qux"},
			prefix:   "bar:",
			expected: -1,
		},
		"match at index 0": {
			keys:     []string{"foo:bar", "foo:baz", "foo:quux", "foo:qux"},
			prefix:   "foo:",
			expected: 0,
		},
		"match at a higher index": {
			keys:     []string{"bar:baz", "bar:foo", "foo:bar", "foo:baz", "foo:quux", "foo:qux"},
			prefix:   "foo:",
			expected: 2,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			keys := make([][]byte, 0, len(c.keys))
			for _, key := range c.keys {
				keys = append(keys, []byte(key))
			}
			actual := findFirstWithPrefix([]byte(c.prefix), keys)
			assert.Equal(t, c.expected, actual)
		})
	}
}

func TestFilterByPrefix(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		keys     []string
		prefix   string
		expected []string
	}{
		"no match": {
			keys:     []string{"foo:bar", "foo:baz", "foo:quux", "foo:qux"},
			prefix:   "bar:",
			expected: nil,
		},
		"full match": {
			keys:     []string{"foo:bar", "foo:baz", "foo:quux", "foo:qux"},
			prefix:   "foo:",
			expected: []string{"foo:bar", "foo:baz", "foo:quux", "foo:qux"},
		},
		"subslice match": {
			keys:     []string{"bar:baz", "bar:foo", "foo:bar", "foo:baz", "foo:quux", "foo:qux", "qux:bar", "qux:baz", "qux:foo"},
			prefix:   "foo:",
			expected: []string{"foo:bar", "foo:baz", "foo:quux", "foo:qux"},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			keys := make([][]byte, 0, len(c.keys))
			for _, key := range c.keys {
				keys = append(keys, []byte(key))
			}

			var expectedKeys [][]byte
			for _, key := range c.expected {
				expectedKeys = append(expectedKeys, []byte(key))
			}

			actual := filterByPrefix([]byte(c.prefix), keys)
			assert.Equal(t, expectedKeys, actual)
		})
	}
}
