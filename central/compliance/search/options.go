package search

import (
	"github.com/stackrox/rox/pkg/search"
)

// Options is exposed for e2e test
var Options = []search.FieldLabel{
	search.Cluster,
	search.Control,
	search.Namespace,
	search.Node,
	search.Standard,
}
