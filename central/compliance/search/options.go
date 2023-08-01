package search

import (
	"github.com/stackrox/rox/central/compliance/standards/index"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/options/deployments"
)

// Options is exposed for e2e test
var Options = []search.FieldLabel{
	search.Cluster,
	search.Control,
	search.Namespace,
	search.Node,
	search.Standard,
	search.DeploymentName,
}

// SearchOptionsMultiMap is the OptionsMultiMap for compliance (which is a little bit of a special snowflake when
// it comes to search).
// Careful: This needs to be kept in sync with the options accessed in `getSearchFuncs` in
// `compliance/aggregation/aggregation.go`.
var SearchOptionsMultiMap = search.MultiMapFromMapsFiltered(
	Options,
	index.StandardOptions,
	schema.ClustersSchema.OptionsMap,
	schema.NodesSchema.OptionsMap,
	schema.NamespacesSchema.OptionsMap,
	index.ControlOptions,
	deployments.OptionsMap,
)
