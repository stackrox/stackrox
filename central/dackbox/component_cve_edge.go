package dackbox

import (
	clusterDackBox "github.com/stackrox/stackrox/central/cluster/dackbox"
	cveDackBox "github.com/stackrox/stackrox/central/cve/dackbox"
	deploymentDackBox "github.com/stackrox/stackrox/central/deployment/dackbox"
	imageDackBox "github.com/stackrox/stackrox/central/image/dackbox"
	componentDackBox "github.com/stackrox/stackrox/central/imagecomponent/dackbox"
	nsDackBox "github.com/stackrox/stackrox/central/namespace/dackbox"
	nodeDackBox "github.com/stackrox/stackrox/central/node/dackbox"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/dackbox/keys/transformation"
)

var (
	// ComponentCVEEdgeTransformations holds the transformations to go from a component:cve edge id to the ids of the given category.
	ComponentCVEEdgeTransformations = map[v1.SearchCategory]transformation.OneToMany{
		// Many(
		//      Edge (parse first key in pair) Component (backwards) Images (backwards) Deployments (backwards) Namespaces (backwards) Clusters,
		//      Edge (parse first key in pair) Component (backwards) Nodes (backwards) Clusters,
		//     )
		v1.SearchCategory_CLUSTERS: transformation.Split([]byte(":")).
			ThenMapEachToOne(transformation.Decode()).
			Then(transformation.AtIndex(0)).
			ThenMapEachToOne(transformation.AddPrefix(componentDackBox.Bucket)).
			ThenMapEachToMany(transformation.Many(
				transformation.BackwardFromContext(imageDackBox.Bucket).
					ThenMapEachToMany(transformation.BackwardFromContext(deploymentDackBox.Bucket)).
					Then(transformation.Dedupe()).
					ThenMapEachToMany(transformation.BackwardFromContext(nsDackBox.Bucket)).
					Then(transformation.Dedupe()).
					ThenMapEachToMany(transformation.BackwardFromContext(clusterDackBox.Bucket)).
					ThenMapEachToOne(transformation.StripPrefixUnchecked(clusterDackBox.Bucket)),
				transformation.BackwardFromContext(nodeDackBox.Bucket).
					Then(transformation.Dedupe()).
					ThenMapEachToMany(transformation.BackwardFromContext(clusterDackBox.Bucket)).
					ThenMapEachToOne(transformation.StripPrefixUnchecked(clusterDackBox.Bucket)).
					Then(transformation.Dedupe()),
			)),

		// Edge (parse first key in pair) Component (backwards) Images (backwards) Deployments (backwards) Namespaces
		v1.SearchCategory_NAMESPACES: transformation.Split([]byte(":")).
			ThenMapEachToOne(transformation.Decode()).
			Then(transformation.AtIndex(0)).
			ThenMapEachToOne(transformation.AddPrefix(componentDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext(imageDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext(deploymentDackBox.Bucket)).
			Then(transformation.Dedupe()).
			ThenMapEachToMany(transformation.BackwardFromContext(nsDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(nsDackBox.Bucket)).
			Then(transformation.Dedupe()),

		// Edge (parse first key in pair) Component (backwards) Images (backwards) Deployments
		v1.SearchCategory_DEPLOYMENTS: transformation.Split([]byte(":")).
			ThenMapEachToOne(transformation.Decode()).
			Then(transformation.AtIndex(0)).
			ThenMapEachToOne(transformation.AddPrefix(componentDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext(imageDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext(deploymentDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(deploymentDackBox.Bucket)).
			Then(transformation.Dedupe()),

		v1.SearchCategory_ACTIVE_COMPONENT: ReturnNothing,

		// Edge (parse first key in pair) Component (backwards) Images
		v1.SearchCategory_IMAGES: transformation.Split([]byte(":")).
			ThenMapEachToOne(transformation.Decode()).
			Then(transformation.AtIndex(0)).
			ThenMapEachToOne(transformation.AddPrefix(componentDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext(imageDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(imageDackBox.Bucket)),

		// CombineReversed ( { k2, k1 }
		//          Edge (parse second key in pair) CVE,
		//          CVE (backwards) Components (backwards) Images,
		//          )
		v1.SearchCategory_IMAGE_VULN_EDGE: transformation.ReverseEdgeKeys(
			transformation.Split([]byte(":")).
				ThenMapEachToOne(transformation.Decode()).
				Then(transformation.AtIndex(1)),
			transformation.AddPrefix(cveDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToMany(transformation.BackwardFromContext(imageDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(imageDackBox.Bucket)).
				Then(transformation.Dedupe()),
		),

		// CombineReversed ( { k2, k1 }
		//          Edge (parse first key in pair) Component,
		//          Components (backwards) Images,
		//          )
		v1.SearchCategory_IMAGE_COMPONENT_EDGE: transformation.ReverseEdgeKeys(
			transformation.Split([]byte(":")).
				ThenMapEachToOne(transformation.Decode()).
				Then(transformation.AtIndex(0)),
			transformation.AddPrefix(componentDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext(imageDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(imageDackBox.Bucket)),
		),

		// Edge (parse first key in pair) Component (backwards) Nodes
		v1.SearchCategory_NODES: transformation.Split([]byte(":")).
			ThenMapEachToOne(transformation.Decode()).
			Then(transformation.AtIndex(0)).
			ThenMapEachToOne(transformation.AddPrefix(componentDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext(nodeDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(nodeDackBox.Bucket)),

		// CombineReversed ( { k2, k1 }
		//          Edge (parse second key in pair) CVE,
		//          CVE (backwards) Components (backwards) Nodes,
		//          )
		v1.SearchCategory_NODE_VULN_EDGE: transformation.ReverseEdgeKeys(
			transformation.Split([]byte(":")).
				ThenMapEachToOne(transformation.Decode()).
				Then(transformation.AtIndex(1)),
			transformation.AddPrefix(cveDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToMany(transformation.BackwardFromContext(nodeDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(nodeDackBox.Bucket)).
				Then(transformation.Dedupe()),
		),

		// CombineReversed ( { k2, k1 }
		//          Edge (parse first key in pair) Component,
		//          Components (backwards) Nodes,
		//          )
		v1.SearchCategory_NODE_COMPONENT_EDGE: transformation.ReverseEdgeKeys(
			transformation.Split([]byte(":")).
				ThenMapEachToOne(transformation.Decode()).
				Then(transformation.AtIndex(0)),
			transformation.AddPrefix(componentDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext(nodeDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(nodeDackBox.Bucket)),
		),

		// Edge (parse first key in pair) Component
		v1.SearchCategory_IMAGE_COMPONENTS: transformation.Split([]byte(":")).
			ThenMapEachToOne(transformation.Decode()).
			Then(transformation.AtIndex(0)),

		// Edge
		v1.SearchCategory_COMPONENT_VULN_EDGE: DoNothing,

		// Edge (parse second key in pair) CVE
		v1.SearchCategory_VULNERABILITIES: transformation.Split([]byte(":")).
			ThenMapEachToOne(transformation.Decode()).
			Then(transformation.AtIndex(1)),

		// We don't want to surface cluster level CVEs from a component:cve scope.
		v1.SearchCategory_CLUSTER_VULN_EDGE: ReturnNothing,
	}
)
