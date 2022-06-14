package common

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// NodeParts represents the pieces of data in an node.
type NodeParts struct {
	Node *storage.Node

	Children []*ComponentParts
}

// ComponentParts represents the pieces of data in a component.
type ComponentParts struct {
	Edge      *storage.NodeComponentEdge
	Component *storage.ImageComponent

	Children []*CVEParts
}

// CVEParts represents the pieces of data in a CVE.
type CVEParts struct {
	Edge *storage.ComponentCVEEdge
	CVE  *storage.CVE
}
