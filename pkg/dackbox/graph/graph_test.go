package graph

import (
	"testing"

	"github.com/stackrox/rox/pkg/dackbox/sortedkeys"
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
}
