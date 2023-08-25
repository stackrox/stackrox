package graph

import (
	"testing"

	"github.com/stackrox/rox/pkg/concurrency/sortedkeys"
	"github.com/stretchr/testify/assert"
)

func TestHistory(t *testing.T) {
	graph := NewGraph()
	history := NewHistory(graph)

	emptyTS := history.Hold()

	modification1 := NewModifiedGraph(NewRemoteGraph(NewGraph(), func(fn func(g RGraph)) {
		fn(history.View(emptyTS))
	}))
	modification1.SetRefs([]byte("fromKey1"), sortedkeys.SortedKeys{[]byte("toKey1"), []byte("toKey2")})
	modification1.SetRefs([]byte("fromKey2"), sortedkeys.SortedKeys{[]byte("toKey1"), []byte("toKey2")})
	modification1.SetRefs([]byte("fromKey3"), sortedkeys.SortedKeys{[]byte("toKey1"), []byte("toKey2")})
	history.Apply(modification1)

	firstStateVew1 := history.Hold()
	firstStateVew2 := history.Hold()

	modification2 := NewModifiedGraph(NewRemoteGraph(NewGraph(), func(fn func(g RGraph)) {
		fn(history.View(firstStateVew1))
	}))
	modification2.SetRefs([]byte("fromKey2"), sortedkeys.SortedKeys{[]byte("toKey3"), []byte("toKey4")}) // resets
	modification2.SetRefs([]byte("fromKey3"), sortedkeys.SortedKeys{[]byte("toKey1"), []byte("toKey2")}) // resets
	history.Apply(modification2)

	allModifications := history.Hold()

	// View before any modifications should be empty.
	assert.Equal(t, [][]byte(nil), history.View(emptyTS).GetRefsFrom([]byte("fromKey1")))
	assert.Equal(t, [][]byte(nil), history.View(emptyTS).GetRefsFrom([]byte("fromKey2")))
	assert.Equal(t, [][]byte(nil), history.View(emptyTS).GetRefsFrom([]byte("fromKey3")))
	assert.Equal(t, [][]byte(nil), history.View(emptyTS).GetRefsTo([]byte("toKey3")))
	assert.Equal(t, [][]byte(nil), history.View(emptyTS).GetRefsTo([]byte("toKey4")))
	history.Release(emptyTS)

	// View after the first set of modifications.
	assert.Equal(t, [][]byte{[]byte("toKey1"), []byte("toKey2")}, history.View(firstStateVew1).GetRefsFrom([]byte("fromKey1")))
	assert.Equal(t, [][]byte{[]byte("toKey1"), []byte("toKey2")}, history.View(firstStateVew1).GetRefsFrom([]byte("fromKey2")))
	assert.Equal(t, [][]byte{[]byte("toKey1"), []byte("toKey2")}, history.View(firstStateVew1).GetRefsFrom([]byte("fromKey3")))

	assert.Equal(t, [][]byte{[]byte("fromKey1"), []byte("fromKey2"), []byte("fromKey3")}, history.View(firstStateVew1).GetRefsTo([]byte("toKey1")))
	assert.Equal(t, [][]byte{[]byte("fromKey1"), []byte("fromKey2"), []byte("fromKey3")}, history.View(firstStateVew1).GetRefsTo([]byte("toKey2")))
	assert.Equal(t, [][]byte(nil), history.View(firstStateVew1).GetRefsTo([]byte("toKey3")))
	assert.Equal(t, [][]byte(nil), history.View(firstStateVew1).GetRefsTo([]byte("toKey4")))
	history.Release(firstStateVew1)

	// Should still be able to access the view after another overlapping view is released.
	assert.Equal(t, [][]byte{[]byte("toKey1"), []byte("toKey2")}, history.View(firstStateVew2).GetRefsFrom([]byte("fromKey1")))
	assert.Equal(t, [][]byte{[]byte("toKey1"), []byte("toKey2")}, history.View(firstStateVew2).GetRefsFrom([]byte("fromKey2")))
	assert.Equal(t, [][]byte{[]byte("toKey1"), []byte("toKey2")}, history.View(firstStateVew2).GetRefsFrom([]byte("fromKey3")))

	assert.Equal(t, [][]byte{[]byte("fromKey1"), []byte("fromKey2"), []byte("fromKey3")}, history.View(firstStateVew2).GetRefsTo([]byte("toKey1")))
	assert.Equal(t, [][]byte{[]byte("fromKey1"), []byte("fromKey2"), []byte("fromKey3")}, history.View(firstStateVew2).GetRefsTo([]byte("toKey2")))
	assert.Equal(t, [][]byte(nil), history.View(firstStateVew2).GetRefsTo([]byte("toKey3")))
	assert.Equal(t, [][]byte(nil), history.View(firstStateVew2).GetRefsTo([]byte("toKey4")))
	history.Release(firstStateVew2)

	// View after all the modifications were submitted.
	assert.Equal(t, [][]byte{[]byte("toKey1"), []byte("toKey2")}, history.View(allModifications).GetRefsFrom([]byte("fromKey1")))
	assert.Equal(t, [][]byte{[]byte("toKey3"), []byte("toKey4")}, history.View(allModifications).GetRefsFrom([]byte("fromKey2")))
	assert.Equal(t, [][]byte{[]byte("toKey1"), []byte("toKey2")}, history.View(allModifications).GetRefsFrom([]byte("fromKey3")))

	assert.Equal(t, [][]byte{[]byte("fromKey1"), []byte("fromKey3")}, history.View(allModifications).GetRefsTo([]byte("toKey1")))
	assert.Equal(t, [][]byte{[]byte("fromKey1"), []byte("fromKey3")}, history.View(allModifications).GetRefsTo([]byte("toKey2")))
	assert.Equal(t, [][]byte{[]byte("fromKey2")}, history.View(allModifications).GetRefsTo([]byte("toKey3")))
	assert.Equal(t, [][]byte{[]byte("fromKey2")}, history.View(allModifications).GetRefsTo([]byte("toKey4")))
	history.Release(allModifications)

	// Final view of master.
	finalView := history.Hold()
	assert.Equal(t, [][]byte{[]byte("toKey1"), []byte("toKey2")}, history.View(finalView).GetRefsFrom([]byte("fromKey1")))
	assert.Equal(t, [][]byte{[]byte("toKey3"), []byte("toKey4")}, history.View(finalView).GetRefsFrom([]byte("fromKey2")))
	assert.Equal(t, [][]byte{[]byte("toKey1"), []byte("toKey2")}, history.View(finalView).GetRefsFrom([]byte("fromKey3")))

	assert.Equal(t, [][]byte{[]byte("fromKey1"), []byte("fromKey3")}, history.View(finalView).GetRefsTo([]byte("toKey1")))
	assert.Equal(t, [][]byte{[]byte("fromKey1"), []byte("fromKey3")}, history.View(finalView).GetRefsTo([]byte("toKey2")))
	assert.Equal(t, [][]byte{[]byte("fromKey2")}, history.View(finalView).GetRefsTo([]byte("toKey3")))
	assert.Equal(t, [][]byte{[]byte("fromKey2")}, history.View(finalView).GetRefsTo([]byte("toKey4")))
	history.Release(finalView)
}
