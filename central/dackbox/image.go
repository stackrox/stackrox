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
	// ImageTransformations holds the transformations to go from an image id to the ids of the given category.
	ImageTransformations = map[v1.SearchCategory]transformation.OneToMany{
		// Image (backwards) Deployments (backwards) Namespaces (backwards) Clusters
		v1.SearchCategory_CLUSTERS: transformation.AddPrefix(imageDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext()).
			Then(transformation.HasPrefix(deploymentDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext()).
			Then(transformation.HasPrefix(nsDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext()).
			Then(transformation.HasPrefix(clusterDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefix(clusterDackBox.Bucket)),

		// Image (backwards) Deployments (backwards) Namespaces
		v1.SearchCategory_NAMESPACES: transformation.AddPrefix(imageDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext()).
			Then(transformation.HasPrefix(deploymentDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext()).
			Then(transformation.HasPrefix(nsDackBox.Bucket)),

		// Image (backwards) Deployments
		v1.SearchCategory_DEPLOYMENTS: transformation.AddPrefix(imageDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext()).
			Then(transformation.HasPrefix(deploymentDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefix(deploymentDackBox.Bucket)),

		// Image
		v1.SearchCategory_IMAGES: DoNothing,

		// Combine ( { k1, k2 }
		//          Image,
		//          Image (forwards) Components,
		//          )
		v1.SearchCategory_IMAGE_COMPONENT_EDGE: transformation.ForwardEdgeKeys(
			DoNothing,
			transformation.AddPrefix(imageDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext()).
				Then(transformation.HasPrefix(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefix(componentDackBox.Bucket)),
		),

		// Image (forwards) Components
		v1.SearchCategory_IMAGE_COMPONENTS: transformation.AddPrefix(imageDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext()).
			Then(transformation.HasPrefix(componentDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefix(componentDackBox.Bucket)),

		// Combine ( { k1, k2 }
		//          Image (forwards) Components,
		//          Components (forwards) CVEs,
		//          )
		v1.SearchCategory_COMPONENT_VULN_EDGE: transformation.ForwardEdgeKeys(
			transformation.AddPrefix(imageDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext()).
				Then(transformation.Dedupe()).
				Then(transformation.HasPrefix(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefix(componentDackBox.Bucket)),
			transformation.AddPrefix(componentDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext()).
				Then(transformation.HasPrefix(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefix(cveDackBox.Bucket)),
		),

		// We don't want to surface cluster level CVEs from an image scope, so we just descend to the CVEs.
		// Image (forwards) Components (forwards) CVEs
		v1.SearchCategory_VULNERABILITIES: transformation.AddPrefix(imageDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext()).
			Then(transformation.HasPrefix(componentDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext()).
			Then(transformation.Dedupe()).
			Then(transformation.HasPrefix(cveDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefix(cveDackBox.Bucket)),

		// We don't want to surface cluster level CVEs from an image scope.
		v1.SearchCategory_CLUSTER_VULN_EDGE: ReturnNothing,
	}
)
