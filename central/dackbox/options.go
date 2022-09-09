package dackbox

import (
	clusterMappings "github.com/stackrox/rox/central/cluster/index/mappings"
	nodeMappings "github.com/stackrox/rox/central/node/index/mappings"
	"github.com/stackrox/rox/pkg/search"
	deploymentMappings "github.com/stackrox/rox/pkg/search/options/deployments"
	imageMappings "github.com/stackrox/rox/pkg/search/options/images"
)

// These options maps are used in searchers where the first match is the searcher used
var (
	DeploymentOnlyOptionsMap = search.Difference(
		deploymentMappings.OptionsMap,
		search.CombineOptionsMaps(
			imageMappings.OptionsMap,
			clusterMappings.OptionsMap,
		),
	)
	ImageOnlyOptionsMap = imageMappings.OptionsMap
	NodeOnlyOptionsMap  = nodeMappings.OptionsMap
)
