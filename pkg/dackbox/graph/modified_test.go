package graph

import (
	"testing"

	"github.com/stackrox/rox/pkg/dackbox/sortedkeys"
	"github.com/stretchr/testify/assert"
)

func TestModifiedGraph(t *testing.T) {
	refState := NewModifiedGraph(NewGraph())

	_ = refState.SetRefs([]byte("fromKey1"), sortedkeys.SortedKeys{[]byte("toKey1"), []byte("toKey2")})
	_ = refState.SetRefs([]byte("fromKey2"), sortedkeys.SortedKeys{[]byte("toKey1"), []byte("toKey2")})
	_ = refState.SetRefs([]byte("fromKey3"), sortedkeys.SortedKeys{[]byte("toKey1"), []byte("toKey2")})

	_ = refState.SetRefs([]byte("fromKey2"), sortedkeys.SortedKeys{[]byte("toKey3"), []byte("toKey4")}) // resets
	_ = refState.SetRefs([]byte("fromKey3"), sortedkeys.SortedKeys{[]byte("toKey1"), []byte("toKey2")}) // resets

	assert.Equal(t, [][]byte{[]byte("toKey1"), []byte("toKey2")}, refState.GetRefsFrom([]byte("fromKey1")))
	assert.Equal(t, [][]byte{[]byte("toKey3"), []byte("toKey4")}, refState.GetRefsFrom([]byte("fromKey2")))
	assert.Equal(t, [][]byte{[]byte("toKey1"), []byte("toKey2")}, refState.GetRefsFrom([]byte("fromKey3")))

	assert.Equal(t, [][]byte{[]byte("fromKey1"), []byte("fromKey3")}, refState.GetRefsTo([]byte("toKey1")))
	assert.Equal(t, [][]byte{[]byte("fromKey1"), []byte("fromKey3")}, refState.GetRefsTo([]byte("toKey2")))
	assert.Equal(t, [][]byte{[]byte("fromKey2")}, refState.GetRefsTo([]byte("toKey3")))
	assert.Equal(t, [][]byte{[]byte("fromKey2")}, refState.GetRefsTo([]byte("toKey4")))

	assert.Equal(t, sortedkeys.SortedKeys{[]byte("fromKey1"), []byte("fromKey2"), []byte("fromKey3")}, refState.modifiedFrom)
	assert.Equal(t, sortedkeys.SortedKeys{[]byte("toKey1"), []byte("toKey2"), []byte("toKey3"), []byte("toKey4")}, refState.modifiedTo)

	_ = refState.DeleteRefs([]byte("fromKey3"))
	assert.Equal(t, [][]byte{[]byte("toKey3"), []byte("toKey4")}, refState.GetRefsFrom([]byte("fromKey2")))
	assert.Equal(t, [][]byte{[]byte("fromKey1")}, refState.GetRefsTo([]byte("toKey2")))
	assert.Equal(t, [][]byte{[]byte("fromKey1")}, refState.GetRefsTo([]byte("toKey1")))
	assert.Equal(t, [][]byte(nil), refState.GetRefsFrom([]byte("fromKey3")))
}
