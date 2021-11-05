package dackbox

import (
	clusterDackBox "github.com/stackrox/rox/central/cluster/dackbox"
	cveDackBox "github.com/stackrox/rox/central/cve/dackbox"
	deploymentDackBox "github.com/stackrox/rox/central/deployment/dackbox"
	imageDackBox "github.com/stackrox/rox/central/image/dackbox"
	componentDackBox "github.com/stackrox/rox/central/imagecomponent/dackbox"
	nsDackBox "github.com/stackrox/rox/central/namespace/dackbox"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/dackbox/keys/transformation"
)

var (
	// ImageCVEEdgeTransformations holds the transformations to go from an image:cve edge id to the ids of the given category.
	ImageCVEEdgeTransformations = map[v1.SearchCategory]transformation.OneToMany{
		// Edge (parse first key in pair) Image (backwards) Deployments (backwards) Namespaces (backwards) Clusters
		v1.SearchCategory_CLUSTERS: transformation.Split([]byte(":")).
			ThenMapEachToOne(transformation.Decode()).
			Then(transformation.AtIndex(0)).
			ThenMapEachToOne(transformation.AddPrefix(imageDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext(deploymentDackBox.Bucket)).
			Then(transformation.Dedupe()).
			ThenMapEachToMany(transformation.BackwardFromContext(nsDackBox.Bucket)).
			Then(transformation.Dedupe()).
			ThenMapEachToMany(transformation.BackwardFromContext(clusterDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(clusterDackBox.Bucket)).
			Then(transformation.Dedupe()),

		// Edge (parse first key in pair) Image (backwards) Deployments (backwards) Namespaces
		v1.SearchCategory_NAMESPACES: transformation.Split([]byte(":")).
			ThenMapEachToOne(transformation.Decode()).
			Then(transformation.AtIndex(0)).
			ThenMapEachToOne(transformation.AddPrefix(imageDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext(deploymentDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext(nsDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(nsDackBox.Bucket)).
			Then(transformation.Dedupe()),

		// Edge (parse first key in pair) Image (backwards) Deployments
		v1.SearchCategory_DEPLOYMENTS: transformation.Split([]byte(":")).
			ThenMapEachToOne(transformation.Decode()).
			Then(transformation.AtIndex(0)).
			ThenMapEachToOne(transformation.AddPrefix(imageDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext(deploymentDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(deploymentDackBox.Bucket)).
			Then(transformation.Dedupe()),

		// ActiveComponent does not have deployment context, so return nothing.
		v1.SearchCategory_ACTIVE_COMPONENT: ReturnNothing,

		// Edge (parse first key in pair) Image
		v1.SearchCategory_IMAGES: transformation.Split([]byte(":")).
			ThenMapEachToOne(transformation.Decode()).
			Then(transformation.AtIndex(0)),

		// Edge
		v1.SearchCategory_IMAGE_VULN_EDGE: DoNothing,

		// Combine ( { k1, k2 }
		//          Edge (parse first key in pair) Image,
		//          Image (forwards) Components,
		//          )
		v1.SearchCategory_IMAGE_COMPONENT_EDGE: transformation.ForwardEdgeKeys(
			transformation.Split([]byte(":")).
				ThenMapEachToOne(transformation.Decode()).
				Then(transformation.AtIndex(0)),
			transformation.AddPrefix(imageDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)),
		),

		// Edge (parse first key in pair) Image (forwards) Components
		v1.SearchCategory_IMAGE_COMPONENTS: transformation.Split([]byte(":")).
			ThenMapEachToOne(transformation.Decode()).
			Then(transformation.AtIndex(0)).
			ThenMapEachToOne(transformation.AddPrefix(imageDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)),

		// Combine ( { k1, k2 }
		//          Image (forwards) Components,
		//          Components (forwards) CVEs,
		//          )
		v1.SearchCategory_COMPONENT_VULN_EDGE: transformation.ForwardEdgeKeys(
			transformation.Split([]byte(":")).
				ThenMapEachToOne(transformation.Decode()).
				Then(transformation.AtIndex(0)).
				ThenMapEachToOne(transformation.AddPrefix(imageDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)).
				Then(transformation.Dedupe()),
			transformation.AddPrefix(componentDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)).
				Then(transformation.Dedupe()),
		),

		// Edge (parse second key in pair) CVE
		v1.SearchCategory_VULNERABILITIES: transformation.Split([]byte(":")).
			ThenMapEachToOne(transformation.Decode()).
			Then(transformation.AtIndex(1)).
			Then(transformation.Dedupe()),

		// We don't want to surface cluster level CVEs from an image scope.
		v1.SearchCategory_CLUSTER_VULN_EDGE: ReturnNothing,

		v1.SearchCategory_NODES:               ReturnNothing,
		v1.SearchCategory_NODE_COMPONENT_EDGE: ReturnNothing,
		v1.SearchCategory_NODE_VULN_EDGE:      ReturnNothing,
	}
)
