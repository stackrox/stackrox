package dackbox

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// NodeParts represents the pieces of data in an node.
type NodeParts struct {
	node *storage.Node

	children []*ComponentParts
	// nodeCVEEdges stores CVE ID to *storage.NodeCVEEdge object mappings
	nodeCVEEdges map[string]*storage.NodeCVEEdge
}

// ComponentParts represents the pieces of data in a component.
type ComponentParts struct {
	edge      *storage.NodeComponentEdge
	component *storage.ImageComponent

	children []*CVEParts
}

// CVEParts represents the pieces of data in a CVE.
type CVEParts struct {
	edge *storage.ComponentCVEEdge
	cve  *storage.CVE
}
