package transformation

import (
	"context"
	"testing"

	graph2 "github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/dackbox/sortedkeys"
	"github.com/stretchr/testify/assert"
)

func TestTransformations(t *testing.T) {
	prefix1 := []byte("p1")
	prefix2 := []byte("p2")
	prefix3 := []byte("p3")

	graph := graph2.NewGraph()

	_ = graph.SetRefs([]byte("p1\x00Key1"), sortedkeys.SortedKeys{[]byte("p2\x00Key1")})
	_ = graph.SetRefs([]byte("p1\x00Key2"), sortedkeys.SortedKeys{[]byte("p2\x00Key1"), []byte("p2\x00Key2")})
	_ = graph.SetRefs([]byte("p1\x00Key3"), sortedkeys.SortedKeys{[]byte("p2\x00Key1"), []byte("p2\x00Key2")})

	_ = graph.SetRefs([]byte("p2\x00Key1"), sortedkeys.SortedKeys{[]byte("p4\x00Key1"), []byte("p3\x00Key2")})
	_ = graph.SetRefs([]byte("p2\x00Key2"), sortedkeys.SortedKeys{[]byte("p3\x00Key1"), []byte("p3\x00Key2")})

	walker := Forward(graph).
		Then(HasPrefix(prefix2)).
		Then(Dedupe()).
		ThenMapEachToMany(Forward(graph)).
		Then(HasPrefix(prefix3)).
		ThenMapEachToOne(StripPrefix(prefix1)).
		Then(Dedupe()).
		Then(Sort())
	assert.Equal(t, [][]byte{[]byte("Key2")}, walker(context.Background(), []byte("p1\x00Key1")))
	assert.Equal(t, [][]byte{[]byte("Key1"), []byte("Key2")}, walker(context.Background(), []byte("p1\x00Key2")))
	assert.Equal(t, [][]byte{[]byte("Key1"), []byte("Key2")}, walker(context.Background(), []byte("p1\x00Key3")))

	walker = Backward(graph).
		Then(HasPrefix(prefix2)).
		Then(Dedupe()).
		ThenMapEachToMany(Backward(graph)).
		Then(HasPrefix(prefix1)).
		ThenMapEachToOne(StripPrefix(prefix1)).
		Then(Dedupe()).
		Then(Sort())
	assert.Equal(t, [][]byte{[]byte("Key1"), []byte("Key2"), []byte("Key3")}, walker(context.Background(), []byte("p4\x00Key1")))
	assert.Equal(t, [][]byte{[]byte("Key1"), []byte("Key2"), []byte("Key3")}, walker(context.Background(), []byte("p3\x00Key2")))
	assert.Equal(t, [][]byte{[]byte("Key2"), []byte("Key3")}, walker(context.Background(), []byte("p3\x00Key1")))
}
