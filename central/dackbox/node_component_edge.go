package dackbox

import (
	clusterDackBox "github.com/stackrox/rox/central/cluster/dackbox"
	cveDackBox "github.com/stackrox/rox/central/cve/dackbox"
	componentDackBox "github.com/stackrox/rox/central/imagecomponent/dackbox"
	nodeDackBox "github.com/stackrox/rox/central/node/dackbox"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/dackbox/keys/transformation"
)

var (
	// NodeComponentEdgeTransformations holds the transformations to go from a node:component edge id to the ids of the given category.
	NodeComponentEdgeTransformations = map[v1.SearchCategory]transformation.OneToMany{
		// Edge (parse first key in pair) Node (backwards) Clusters
		v1.SearchCategory_CLUSTERS: transformation.Split([]byte(":")).
			ThenMapEachToOne(transformation.Decode()).
			Then(transformation.AtIndex(0)).
			ThenMapEachToOne(transformation.AddPrefix(nodeDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext(clusterDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(clusterDackBox.Bucket)).
			Then(transformation.Dedupe()),

		// Edge (parse first key in pair) Node
		v1.SearchCategory_NODES: transformation.Split([]byte(":")).
			ThenMapEachToOne(transformation.Decode()).
			Then(transformation.AtIndex(0)),

		// Combine ( { k1, k2 }
		//          Edge (parse first key in pair) Node,
		//          Components (forwards) CVEs,
		//          )
		v1.SearchCategory_NODE_VULN_EDGE: transformation.ForwardEdgeKeys(
			transformation.Split([]byte(":")).
				ThenMapEachToOne(transformation.Decode()).
				Then(transformation.AtIndex(0)),
			transformation.AddPrefix(nodeDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)).
				Then(transformation.Dedupe()),
		),

		// Edge
		v1.SearchCategory_NODE_COMPONENT_EDGE: DoNothing,

		// Edge (parse second key in pair) Component
		v1.SearchCategory_IMAGE_COMPONENTS: transformation.Split([]byte(":")).
			ThenMapEachToOne(transformation.Decode()).
			Then(transformation.AtIndex(1)),

		// Combine ( { k1, k2 }
		//          Edge (parse second key in pair) Component,
		//          Components (forwards) CVEs,
		//          )
		v1.SearchCategory_COMPONENT_VULN_EDGE: transformation.ForwardEdgeKeys(
			transformation.Split([]byte(":")).
				ThenMapEachToOne(transformation.Decode()).
				Then(transformation.AtIndex(1)),
			transformation.AddPrefix(componentDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)),
		),

		// Edge (parse second key in pair) Component (forward) CVE
		v1.SearchCategory_VULNERABILITIES: transformation.Split([]byte(":")).
			ThenMapEachToOne(transformation.Decode()).
			Then(transformation.AtIndex(1)).
			ThenMapEachToOne(transformation.AddPrefix(componentDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)).
			Then(transformation.Dedupe()),

		// We don't want to surface cluster level vulns from a node:component scope.
		v1.SearchCategory_CLUSTER_VULN_EDGE: ReturnNothing,

		v1.SearchCategory_NAMESPACES:           ReturnNothing,
		v1.SearchCategory_DEPLOYMENTS:          ReturnNothing,
		v1.SearchCategory_ACTIVE_COMPONENT:     ReturnNothing,
		v1.SearchCategory_IMAGES:               ReturnNothing,
		v1.SearchCategory_IMAGE_COMPONENT_EDGE: ReturnNothing,
		v1.SearchCategory_IMAGE_VULN_EDGE:      ReturnNothing,
	}
)
