package dackbox

import (
	"github.com/stackrox/rox/pkg/dackbox/graph"
)

// DiscardableRGraph is an RGraph (read only view of the ID->[]ID map layer) that needs to be discarded when finished.
type DiscardableRGraph interface {
	graph.RGraph

	Discard()
}

type discardableGraphImpl struct {
	*graph.RemoteGraph

	ts      uint64
	discard RemoteDiscard
}

// Discard dumps all of the transaction's changes.
func (b *discardableGraphImpl) Discard() {
	b.discard(b.ts, nil)
}
