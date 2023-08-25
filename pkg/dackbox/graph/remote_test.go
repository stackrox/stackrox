package graph

import (
	"testing"

	"github.com/stackrox/rox/pkg/concurrency/sortedkeys"
	"github.com/stretchr/testify/assert"
)

func TestRemoteGraph(t *testing.T) {
	refState := NewGraph()

	refState.SetRefs([]byte("fromKey1"), sortedkeys.SortedKeys{[]byte("toKey1"), []byte("toKey2")})
	refState.SetRefs([]byte("fromKey2"), sortedkeys.SortedKeys{[]byte("toKey1"), []byte("toKey2")})
	refState.SetRefs([]byte("fromKey3"), sortedkeys.SortedKeys{[]byte("toKey1"), []byte("toKey2")})

	reader := &simpleRemote{refState}

	remoteGraph := NewRemoteGraph(NewModifiedGraph(refState), reader.Read)
	remoteGraph.SetRefs([]byte("fromKey2"), sortedkeys.SortedKeys{[]byte("toKey3"), []byte("toKey4")})
	remoteGraph.SetRefs([]byte("fromKey3"), sortedkeys.SortedKeys{[]byte("toKey1"), []byte("toKey2")})

	assert.Equal(t, [][]byte{[]byte("toKey1"), []byte("toKey2")}, remoteGraph.GetRefsFrom([]byte("fromKey1")))
	assert.Equal(t, [][]byte{[]byte("toKey3"), []byte("toKey4")}, remoteGraph.GetRefsFrom([]byte("fromKey2")))
	assert.Equal(t, [][]byte{[]byte("toKey1"), []byte("toKey2")}, remoteGraph.GetRefsFrom([]byte("fromKey3")))

	assert.Equal(t, [][]byte{[]byte("fromKey1"), []byte("fromKey3")}, remoteGraph.GetRefsTo([]byte("toKey1")))
	assert.Equal(t, [][]byte{[]byte("fromKey1"), []byte("fromKey3")}, remoteGraph.GetRefsTo([]byte("toKey2")))
	assert.Equal(t, [][]byte{[]byte("fromKey2")}, remoteGraph.GetRefsTo([]byte("toKey3")))
	assert.Equal(t, [][]byte{[]byte("fromKey2")}, remoteGraph.GetRefsTo([]byte("toKey4")))

	assert.Equal(t, sortedkeys.SortedKeys{[]byte("fromKey2"), []byte("fromKey3")}, remoteGraph.RWGraph.(*ModifiedGraph).modifiedFrom)
	assert.Equal(t, sortedkeys.SortedKeys{[]byte("toKey1"), []byte("toKey2"), []byte("toKey3"), []byte("toKey4")}, remoteGraph.RWGraph.(*ModifiedGraph).modifiedTo)
}

// simple RemoteReader implementation for testing.
type simpleRemote struct {
	graph *Graph
}

func (sr *simpleRemote) Read(reader func(graph RGraph)) {
	reader(sr.graph)
}
