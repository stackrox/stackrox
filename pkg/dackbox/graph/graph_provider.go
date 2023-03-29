package graph

// Provider is an interface that allows us to interact with an RGraph for the duration of a function's execution.
//
//go:generate mockgen-wrapper
type Provider interface {
	NewGraphView() DiscardableRGraph
}
