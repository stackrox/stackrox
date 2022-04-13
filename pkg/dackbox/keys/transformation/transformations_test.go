package transformation

import (
	"context"
	"testing"

	graph2 "github.com/stackrox/stackrox/pkg/dackbox/graph"
	"github.com/stackrox/stackrox/pkg/dackbox/graph/testutils"
	"github.com/stackrox/stackrox/pkg/dackbox/sortedkeys"
	"github.com/stretchr/testify/assert"
)

func TestTransformations(t *testing.T) {
	prefix1 := []byte("p1")
	prefix2 := []byte("p2")
	prefix3 := []byte("p3")

	graph := graph2.NewGraph()

	graph.SetRefs([]byte("p1\x00Key1"), sortedkeys.SortedKeys{[]byte("p2\x00Key1")})
	graph.SetRefs([]byte("p1\x00Key2"), sortedkeys.SortedKeys{[]byte("p2\x00Key1"), []byte("p2\x00Key2")})
	graph.SetRefs([]byte("p1\x00Key3"), sortedkeys.SortedKeys{[]byte("p2\x00Key1"), []byte("p2\x00Key2")})

	graph.SetRefs([]byte("p2\x00Key1"), sortedkeys.SortedKeys{[]byte("p4\x00Key1"), []byte("p3\x00Key2")})
	graph.SetRefs([]byte("p2\x00Key2"), sortedkeys.SortedKeys{[]byte("p3\x00Key1"), []byte("p3\x00Key2")})

	walker := ForwardFromContext(prefix2).
		Then(Dedupe()).
		ThenMapEachToMany(ForwardFromContext(prefix3)).
		ThenMapEachToOne(StripPrefixUnchecked(prefix3)).
		Then(Dedupe())
	testutils.DoWithGraph(context.Background(), graph, func(ctx context.Context) {
		assert.Equal(t, [][]byte{[]byte("Key2")}, walker(ctx, []byte("p1\x00Key1")))
		assert.ElementsMatch(t, [][]byte{[]byte("Key1"), []byte("Key2")}, walker(ctx, []byte("p1\x00Key2")))
		assert.ElementsMatch(t, [][]byte{[]byte("Key1"), []byte("Key2")}, walker(ctx, []byte("p1\x00Key3")))
	})

	walker = BackwardFromContext(prefix2).
		Then(Dedupe()).
		ThenMapEachToMany(BackwardFromContext(prefix1)).
		ThenMapEachToOne(StripPrefixUnchecked(prefix1)).
		Then(Dedupe())
	testutils.DoWithGraph(context.Background(), graph, func(ctx context.Context) {
		assert.ElementsMatch(t, [][]byte{[]byte("Key1"), []byte("Key2"), []byte("Key3")}, walker(ctx, []byte("p4\x00Key1")))
		assert.ElementsMatch(t, [][]byte{[]byte("Key1"), []byte("Key2"), []byte("Key3")}, walker(ctx, []byte("p3\x00Key2")))
		assert.ElementsMatch(t, [][]byte{[]byte("Key2"), []byte("Key3")}, walker(ctx, []byte("p3\x00Key1")))
	})
}
