package dackbox

import (
	clusterMappings "github.com/stackrox/rox/central/cluster/index/mappings"
	componentCVEEdgeMappings "github.com/stackrox/rox/central/componentcveedge/mappings"
	cveMappings "github.com/stackrox/rox/central/cve/mappings"
	componentMappings "github.com/stackrox/rox/central/imagecomponent/mappings"
	imageComponentEdgeMappings "github.com/stackrox/rox/central/imagecomponentedge/mappings"
	imageCVEEdgeMappings "github.com/stackrox/rox/central/imagecveedge/mappings"
	nodeMappings "github.com/stackrox/rox/central/node/index/mappings"
	nodeComponentEdgeMappings "github.com/stackrox/rox/central/nodecomponentedge/mappings"
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
	ImageOnlyOptionsMap = search.Difference(
		imageMappings.OptionsMap,
		search.CombineOptionsMaps(
			imageComponentEdgeMappings.OptionsMap,
			componentMappings.OptionsMap,
			componentCVEEdgeMappings.OptionsMap,
			imageCVEEdgeMappings.OptionsMap,
			cveMappings.OptionsMap,
		),
	)
	NodeOnlyOptionsMap = search.Difference(
		nodeMappings.OptionsMap,
		search.CombineOptionsMaps(
			nodeComponentEdgeMappings.OptionsMap,
			componentMappings.OptionsMap,
			componentCVEEdgeMappings.OptionsMap,
			cveMappings.OptionsMap,
			clusterMappings.OptionsMap,
		),
	)
)
