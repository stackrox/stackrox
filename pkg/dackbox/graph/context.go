package graph

import "context"

// Context executes the input function using the input graph provider with a context that provides a lazy generated
// graph.
func Context(ctx context.Context, provider Provider, fn func(hasGraphConext context.Context)) {
	inner := context.WithValue(ctx, graphContextKey{}, &graphContext{
		provider: provider,
	})
	defer discardGraph(inner)
	fn(inner)
}

// ContextWithStatus functions the same as GraphContext, but returns an error value, which is returned from the
// input function to be executed.
func ContextWithStatus(ctx context.Context, provider Provider, fn func(hasGraphConext context.Context) error) error {
	inner := context.WithValue(ctx, graphContextKey{}, &graphContext{
		provider: provider,
	})
	defer discardGraph(inner)
	return fn(inner)
}

// GetGraph returns the RGraph from the input context, or nil if the input context has no graph context.
func GetGraph(hasGraphContext context.Context) RGraph {
	inter := hasGraphContext.Value(graphContextKey{})
	if inter == nil {
		return nil
	}
	gc := inter.(*graphContext)
	if gc.graph == nil {
		gc.graph = gc.provider.NewGraphView()
	}
	return gc.graph
}

// Key for fetching graph instance from the contenxt.
type graphContextKey struct{}

type graphContext struct {
	provider Provider
	graph    DiscardableRGraph
}

func discardGraph(ctx context.Context) {
	inter := ctx.Value(graphContextKey{})
	if inter == nil {
		return
	}
	gc := inter.(*graphContext)
	if gc.graph == nil {
		return
	}
	gc.graph.Discard()
}
