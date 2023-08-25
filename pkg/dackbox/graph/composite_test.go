package graph

import (
	"testing"

	"github.com/stackrox/rox/pkg/concurrency/sortedkeys"
	"github.com/stretchr/testify/assert"
)

func TestComposite(t *testing.T) {
	baseGraph := NewGraph()
	baseGraph.SetRefs([]byte("fromKey1"), sortedkeys.SortedKeys{[]byte("toKey1"), []byte("toKey2")})
	baseGraph.SetRefs([]byte("fromKey2"), sortedkeys.SortedKeys{[]byte("toKey1"), []byte("toKey4")})
	baseGraph.SetRefs([]byte("fromKey3"), sortedkeys.SortedKeys{[]byte("toKey1"), []byte("toKey6")})

	graph1 := NewModifiedGraph(baseGraph.Copy())
	graph1.SetRefs([]byte("fromKey2"), sortedkeys.SortedKeys{[]byte("toKey2"), []byte("toKey3")})

	graph2 := NewModifiedGraph(graph1.RWGraph.(*Graph).Copy())
	graph2.SetRefs([]byte("fromKey3"), sortedkeys.SortedKeys{[]byte("toKey3"), []byte("toKey4")})

	composite := NewCompositeGraph(baseGraph, graph1, graph2)

	assert.Equal(t, [][]byte{[]byte("toKey1"), []byte("toKey2")}, composite.GetRefsFrom([]byte("fromKey1")))
	assert.Equal(t, [][]byte{[]byte("toKey2"), []byte("toKey3")}, composite.GetRefsFrom([]byte("fromKey2")))
	assert.Equal(t, [][]byte{[]byte("toKey3"), []byte("toKey4")}, composite.GetRefsFrom([]byte("fromKey3")))

	assert.Equal(t, [][]byte{[]byte("fromKey1")}, composite.GetRefsTo([]byte("toKey1")))
	assert.Equal(t, [][]byte{[]byte("fromKey1"), []byte("fromKey2")}, composite.GetRefsTo([]byte("toKey2")))
	assert.Equal(t, [][]byte{[]byte("fromKey2"), []byte("fromKey3")}, composite.GetRefsTo([]byte("toKey3")))
	assert.Equal(t, [][]byte{[]byte("fromKey3")}, composite.GetRefsTo([]byte("toKey4")))
	assert.Equal(t, [][]byte(nil), composite.GetRefsTo([]byte("toKey5")))
	assert.Equal(t, [][]byte(nil), composite.GetRefsTo([]byte("toKey6")))
}
