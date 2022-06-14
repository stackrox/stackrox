package testutils

import (
	"github.com/stackrox/stackrox/pkg/dackbox"
	"github.com/stackrox/stackrox/pkg/dackbox/graph"
)

// AddPathsToGraph adds the given paths to the graph.
func AddPathsToGraph(g graph.RWGraph, paths ...dackbox.Path) {
	for _, path := range paths {
		var prev []byte
		for _, elem := range path.Path {
			g.AddRefs(elem)
			if prev != nil {
				if path.ForwardTraversal {
					g.AddRefs(prev, elem)
				} else {
					g.AddRefs(elem, prev)
				}
			}
			prev = elem
		}
	}
}

// GraphFromPaths returns a graph constructed from the given paths.
func GraphFromPaths(paths ...dackbox.Path) *graph.Graph {
	g := graph.NewGraph()
	AddPathsToGraph(g, paths...)
	return g
}
