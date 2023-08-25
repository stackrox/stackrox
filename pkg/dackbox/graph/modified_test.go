package graph

import (
	"testing"

	"github.com/stackrox/rox/pkg/concurrency/sortedkeys"
	"github.com/stretchr/testify/assert"
)

func TestModifiedGraph(t *testing.T) {
	base := NewGraph()
	modified := NewModifiedGraph(base)

	modified.SetRefs([]byte("fromKey1"), sortedkeys.SortedKeys{[]byte("toKey1"), []byte("toKey2")})
	modified.SetRefs([]byte("fromKey2"), sortedkeys.SortedKeys{[]byte("toKey1"), []byte("toKey2")})
	modified.SetRefs([]byte("fromKey3"), sortedkeys.SortedKeys{[]byte("toKey1"), []byte("toKey2")})

	modified.SetRefs([]byte("fromKey2"), sortedkeys.SortedKeys{[]byte("toKey3"), []byte("toKey4")}) // resets
	modified.SetRefs([]byte("fromKey3"), sortedkeys.SortedKeys{[]byte("toKey1"), []byte("toKey2")}) // resets

	assert.Equal(t, [][]byte{[]byte("toKey1"), []byte("toKey2")}, modified.GetRefsFrom([]byte("fromKey1")))
	assert.Equal(t, [][]byte{[]byte("toKey3"), []byte("toKey4")}, modified.GetRefsFrom([]byte("fromKey2")))
	assert.Equal(t, [][]byte{[]byte("toKey1"), []byte("toKey2")}, modified.GetRefsFrom([]byte("fromKey3")))

	assert.Equal(t, [][]byte{[]byte("fromKey1"), []byte("fromKey3")}, modified.GetRefsTo([]byte("toKey1")))
	assert.Equal(t, [][]byte{[]byte("fromKey1"), []byte("fromKey3")}, modified.GetRefsTo([]byte("toKey2")))
	assert.Equal(t, [][]byte{[]byte("fromKey2")}, modified.GetRefsTo([]byte("toKey3")))
	assert.Equal(t, [][]byte{[]byte("fromKey2")}, modified.GetRefsTo([]byte("toKey4")))

	assert.Equal(t, sortedkeys.SortedKeys{[]byte("fromKey1"), []byte("fromKey2"), []byte("fromKey3")}, modified.modifiedFrom)
	assert.Equal(t, sortedkeys.SortedKeys{[]byte("toKey1"), []byte("toKey2"), []byte("toKey3"), []byte("toKey4")}, modified.modifiedTo)

	modified.DeleteRefsFrom([]byte("fromKey3"))
	assert.Equal(t, [][]byte{[]byte("toKey3"), []byte("toKey4")}, modified.GetRefsFrom([]byte("fromKey2")))
	assert.Equal(t, [][]byte(nil), modified.GetRefsFrom([]byte("fromKey3")))
	assert.Equal(t, [][]byte{[]byte("fromKey1")}, modified.GetRefsTo([]byte("toKey2")))
	assert.Equal(t, [][]byte{[]byte("fromKey1")}, modified.GetRefsTo([]byte("toKey1")))
	assert.Equal(t, [][]byte{[]byte("fromKey2")}, modified.GetRefsTo([]byte("toKey3")))
	assert.Equal(t, [][]byte{[]byte("fromKey2")}, modified.GetRefsTo([]byte("toKey4")))

	modified.DeleteRefsTo([]byte("toKey3"))
	assert.Equal(t, [][]byte{[]byte("toKey4")}, modified.GetRefsFrom([]byte("fromKey2")))
	assert.Equal(t, [][]byte(nil), modified.GetRefsFrom([]byte("fromKey3")))
	assert.Equal(t, [][]byte{[]byte("fromKey1")}, modified.GetRefsTo([]byte("toKey2")))
	assert.Equal(t, [][]byte{[]byte("fromKey1")}, modified.GetRefsTo([]byte("toKey1")))
	assert.Equal(t, [][]byte(nil), modified.GetRefsTo([]byte("toKey3")))
	assert.Equal(t, [][]byte{[]byte("fromKey2")}, modified.GetRefsTo([]byte("toKey4")))

	applied := NewGraph()
	modified.Apply(applied)
	assert.Equal(t, [][]byte{[]byte("toKey4")}, applied.GetRefsFrom([]byte("fromKey2")))
	assert.Equal(t, [][]byte(nil), applied.GetRefsFrom([]byte("fromKey3")))
	assert.Equal(t, [][]byte{[]byte("fromKey1")}, applied.GetRefsTo([]byte("toKey2")))
	assert.Equal(t, [][]byte{[]byte("fromKey1")}, applied.GetRefsTo([]byte("toKey1")))
	assert.Equal(t, [][]byte(nil), applied.GetRefsTo([]byte("toKey3")))
	assert.Equal(t, [][]byte{[]byte("fromKey2")}, applied.GetRefsTo([]byte("toKey4")))
}
