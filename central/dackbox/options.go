package dackbox

import (
	clusterMappings "github.com/stackrox/stackrox/central/cluster/index/mappings"
	componentCVEEdgeMappings "github.com/stackrox/stackrox/central/componentcveedge/mappings"
	cveMappings "github.com/stackrox/stackrox/central/cve/mappings"
	componentMappings "github.com/stackrox/stackrox/central/imagecomponent/mappings"
	imageComponentEdgeMappings "github.com/stackrox/stackrox/central/imagecomponentedge/mappings"
	imageCVEEdgeMappings "github.com/stackrox/stackrox/central/imagecveedge/mappings"
	nodeMappings "github.com/stackrox/stackrox/central/node/index/mappings"
	nodeComponentEdgeMappings "github.com/stackrox/stackrox/central/nodecomponentedge/mappings"
	"github.com/stackrox/stackrox/pkg/search"
	deploymentMappings "github.com/stackrox/stackrox/pkg/search/options/deployments"
	imageMappings "github.com/stackrox/stackrox/pkg/search/options/images"
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
