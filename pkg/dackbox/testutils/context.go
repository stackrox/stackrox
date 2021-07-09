package testutils

import (
	"context"

	"github.com/stackrox/rox/pkg/dackbox/graph"
)

type fixedGraphProvider struct {
	graph graph.RGraph
}

func (p fixedGraphProvider) NewGraphView() graph.DiscardableRGraph {
	return graph.NewDiscardableGraph(p.graph, func() {})
}

// DoWithGraph executes the given function in a context that has the given graph injected.
func DoWithGraph(ctx context.Context, g graph.RGraph, fn func(ctx context.Context)) {
	graph.Context(ctx, fixedGraphProvider{graph: g}, fn)
}
