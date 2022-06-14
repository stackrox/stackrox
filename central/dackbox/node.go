package dackbox

import (
	clusterDackBox "github.com/stackrox/stackrox/central/cluster/dackbox"
	cveDackBox "github.com/stackrox/stackrox/central/cve/dackbox"
	componentDackBox "github.com/stackrox/stackrox/central/imagecomponent/dackbox"
	nodeDackBox "github.com/stackrox/stackrox/central/node/dackbox"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/dackbox"
	"github.com/stackrox/stackrox/pkg/dackbox/keys/transformation"
)

var (
	// NodeTransformationPaths holds the paths to go from a node id to the ids of the given category.
	// NOT A COMPLETE REPLACEMENT OF TRANSFORMATIONS BELOW.
	NodeTransformationPaths = map[v1.SearchCategory]dackbox.BucketPath{
		v1.SearchCategory_CLUSTERS: dackbox.BackwardsBucketPath(
			nodeDackBox.BucketHandler,
			clusterDackBox.BucketHandler,
		),
	}

	// NodeTransformations holds the transformations to go from a node id to the ids of the given category.
	NodeTransformations = map[v1.SearchCategory]transformation.OneToMany{
		// Node (backwards) Clusters
		v1.SearchCategory_CLUSTERS: transformation.AddPrefix(nodeDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext(clusterDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(clusterDackBox.Bucket)).
			Then(transformation.Dedupe()),

		// Node
		v1.SearchCategory_NODES: DoNothing,

		// Combine ( { k1, k2 }
		//          Node,
		//          Node (forwards) Components (forwards) CVEs,
		//          )
		v1.SearchCategory_NODE_VULN_EDGE: transformation.ForwardEdgeKeys(
			DoNothing,
			transformation.AddPrefix(nodeDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)).
				Then(transformation.Dedupe()),
		),

		// Combine ( { k1, k2 }
		//          Node,
		//          Node (forwards) Components,
		//          )
		v1.SearchCategory_NODE_COMPONENT_EDGE: transformation.ForwardEdgeKeys(
			DoNothing,
			transformation.AddPrefix(nodeDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)),
		),

		// Node (forwards) Components
		v1.SearchCategory_IMAGE_COMPONENTS: transformation.AddPrefix(nodeDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)),

		// Combine ( { k1, k2 }
		//          Node (forwards) Components,
		//          Components (forwards) CVEs,
		//          )
		v1.SearchCategory_COMPONENT_VULN_EDGE: transformation.ForwardEdgeKeys(
			transformation.AddPrefix(nodeDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)).
				Then(transformation.Dedupe()),
			transformation.AddPrefix(componentDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)),
		),

		// We don't want to surface cluster level CVEs from a node scope, so we just descend to the CVEs.
		// Node (forwards) Components (forwards) CVEs
		v1.SearchCategory_VULNERABILITIES: transformation.AddPrefix(nodeDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)).
			Then(transformation.Dedupe()),

		// We don't want to surface cluster level CVEs from a node scope.
		v1.SearchCategory_CLUSTER_VULN_EDGE: ReturnNothing,

		v1.SearchCategory_NAMESPACES:           ReturnNothing,
		v1.SearchCategory_DEPLOYMENTS:          ReturnNothing,
		v1.SearchCategory_ACTIVE_COMPONENT:     ReturnNothing,
		v1.SearchCategory_IMAGES:               ReturnNothing,
		v1.SearchCategory_IMAGE_COMPONENT_EDGE: ReturnNothing,
		v1.SearchCategory_IMAGE_VULN_EDGE:      ReturnNothing,
	}
)
