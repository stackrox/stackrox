package dackbox

import (
	clusterDackBox "github.com/stackrox/stackrox/central/cluster/dackbox"
	cveDackBox "github.com/stackrox/stackrox/central/cve/dackbox"
	deploymentDackBox "github.com/stackrox/stackrox/central/deployment/dackbox"
	imageDackBox "github.com/stackrox/stackrox/central/image/dackbox"
	componentDackBox "github.com/stackrox/stackrox/central/imagecomponent/dackbox"
	nsDackBox "github.com/stackrox/stackrox/central/namespace/dackbox"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/dackbox"
	"github.com/stackrox/stackrox/pkg/dackbox/keys/transformation"
)

var (
	// ImageTransformationPaths holds the paths to go from an image id to the ids of the given category.
	// NOT A COMPLETE REPLACEMENT OF TRANSFORMATIONS BELOW
	ImageTransformationPaths = map[v1.SearchCategory]dackbox.BucketPath{
		v1.SearchCategory_NAMESPACES: dackbox.BackwardsBucketPath(
			imageDackBox.BucketHandler,
			deploymentDackBox.BucketHandler,
			nsDackBox.BucketHandler,
		),
	}
	// ImageTransformations holds the transformations to go from an image id to the ids of the given category.
	ImageTransformations = map[v1.SearchCategory]transformation.OneToMany{
		// Image (backwards) Deployments (backwards) Namespaces (backwards) Clusters
		v1.SearchCategory_CLUSTERS: transformation.AddPrefix(imageDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext(deploymentDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext(nsDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext(clusterDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(clusterDackBox.Bucket)),

		// Image (backwards) Deployments (backwards) Namespaces
		v1.SearchCategory_NAMESPACES: transformation.AddPrefix(imageDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext(deploymentDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext(nsDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(nsDackBox.Bucket)),

		// Image (backwards) Deployments
		v1.SearchCategory_DEPLOYMENTS: transformation.AddPrefix(imageDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext(deploymentDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(deploymentDackBox.Bucket)),

		// Image does not have deployment context, so return nothing.
		v1.SearchCategory_ACTIVE_COMPONENT: ReturnNothing,

		// Image
		v1.SearchCategory_IMAGES: DoNothing,

		// Combine ( { k1, k2 }
		//          Image,
		//          Image (forwards) Components (forwards) CVEs,
		//          )
		v1.SearchCategory_IMAGE_VULN_EDGE: transformation.ForwardEdgeKeys(
			DoNothing,
			transformation.AddPrefix(imageDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)).
				Then(transformation.Dedupe())),

		// Combine ( { k1, k2 }
		//          Image,
		//          Image (forwards) Components,
		//          )
		v1.SearchCategory_IMAGE_COMPONENT_EDGE: transformation.ForwardEdgeKeys(
			DoNothing,
			transformation.AddPrefix(imageDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)),
		),

		// Image (forwards) Components
		v1.SearchCategory_IMAGE_COMPONENTS: transformation.AddPrefix(imageDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)),

		// Combine ( { k1, k2 }
		//          Image (forwards) Components,
		//          Components (forwards) CVEs,
		//          )
		v1.SearchCategory_COMPONENT_VULN_EDGE: transformation.ForwardEdgeKeys(
			transformation.AddPrefix(imageDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
				Then(transformation.Dedupe()).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)),
			transformation.AddPrefix(componentDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)).
				Then(transformation.Dedupe()),
		),

		// We don't want to surface cluster level CVEs from an image scope, so we just descend to the CVEs.
		// Image (forwards) Components (forwards) CVEs
		v1.SearchCategory_VULNERABILITIES: transformation.AddPrefix(imageDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)).
			Then(transformation.Dedupe()),

		// We don't want to surface cluster level CVEs from an image scope.
		v1.SearchCategory_CLUSTER_VULN_EDGE: ReturnNothing,

		v1.SearchCategory_NODES:               ReturnNothing,
		v1.SearchCategory_NODE_COMPONENT_EDGE: ReturnNothing,
		v1.SearchCategory_NODE_VULN_EDGE:      ReturnNothing,
	}
)
