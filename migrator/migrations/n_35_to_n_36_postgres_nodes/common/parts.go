package common

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/loghelper"
)

var (
	log = loghelper.LogWrapper{}
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
