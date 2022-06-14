package mappings

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
)

// OptionsMap defines the search options for Vulnerabilities stored in nodes.
var OptionsMap = search.Walk(v1.SearchCategory_NODE_COMPONENT_EDGE, "nodecomponentedge", (*storage.NodeComponentEdge)(nil))
