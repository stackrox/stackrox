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
			ThenMapEachToMany(transformation.BackwardFromContext()).
			Then(transformation.HasPrefix(deploymentDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext()).
			Then(transformation.HasPrefix(nsDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext()).
			Then(transformation.HasPrefix(clusterDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefix(clusterDackBox.Bucket)),

		// Edge (parse first key in pair) Image (backwards) Deployments (backwards) Namespaces
		v1.SearchCategory_NAMESPACES: transformation.Split([]byte(":")).
			ThenMapEachToOne(transformation.Decode()).
			Then(transformation.AtIndex(0)).
			ThenMapEachToOne(transformation.AddPrefix(imageDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext()).
			Then(transformation.HasPrefix(deploymentDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext()).
			Then(transformation.Dedupe()).
			Then(transformation.HasPrefix(nsDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefix(nsDackBox.Bucket)),

		// Edge (parse first key in pair) Image (backwards) Deployments
		v1.SearchCategory_DEPLOYMENTS: transformation.AddPrefix(imageDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext()).
			Then(transformation.HasPrefix(deploymentDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefix(deploymentDackBox.Bucket)),

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
				ThenMapToMany(transformation.ForwardFromContext()).
				Then(transformation.HasPrefix(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefix(componentDackBox.Bucket)),
		),

		// Edge (parse first key in pair) Image (forwards) Components
		v1.SearchCategory_IMAGE_COMPONENTS: transformation.Split([]byte(":")).
			ThenMapEachToOne(transformation.Decode()).
			Then(transformation.AtIndex(0)).
			ThenMapEachToOne(transformation.AddPrefix(imageDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext()).
			Then(transformation.HasPrefix(componentDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefix(componentDackBox.Bucket)),

		// Combine ( { k1, k2 }
		//          Image (forwards) Components,
		//          Components (forwards) CVEs,
		//          )
		v1.SearchCategory_COMPONENT_VULN_EDGE: transformation.ForwardEdgeKeys(
			transformation.Split([]byte(":")).
				ThenMapEachToOne(transformation.Decode()).
				Then(transformation.AtIndex(0)).
				ThenMapEachToOne(transformation.AddPrefix(imageDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext()).
				Then(transformation.Dedupe()).
				Then(transformation.HasPrefix(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefix(componentDackBox.Bucket)),
			transformation.AddPrefix(componentDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext()).
				Then(transformation.HasPrefix(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefix(cveDackBox.Bucket)),
		),

		// Edge (parse second key in pair) CVE
		v1.SearchCategory_VULNERABILITIES: transformation.Split([]byte(":")).
			ThenMapEachToOne(transformation.Decode()).
			Then(transformation.AtIndex(1)),

		// We don't want to surface cluster level CVEs from an image scope.
		v1.SearchCategory_CLUSTER_VULN_EDGE: ReturnNothing,
	}
)
