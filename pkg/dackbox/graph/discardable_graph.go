package graph

// NewDiscardableGraph returns an instance of a DiscardableRGraph using the input RGraph and discard function.
func NewDiscardableGraph(rGraph RGraph, discard func()) DiscardableRGraph {
	return &discardableGraphImpl{
		RGraph:  rGraph,
		discard: discard,
	}
}

type discardableGraphImpl struct {
	RGraph

	discard func()
}

// Discard dumps all of the transaction's changes.
func (b *discardableGraphImpl) Discard() {
	b.discard()
}
