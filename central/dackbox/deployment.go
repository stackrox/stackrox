package dackbox

import (
	acDackBox "github.com/stackrox/rox/central/activecomponent/dackbox"
	clusterDackBox "github.com/stackrox/rox/central/cluster/dackbox"
	cveDackBox "github.com/stackrox/rox/central/cve/dackbox"
	deploymentDackBox "github.com/stackrox/rox/central/deployment/dackbox"
	imageDackBox "github.com/stackrox/rox/central/image/dackbox"
	componentDackBox "github.com/stackrox/rox/central/imagecomponent/dackbox"
	nsDackBox "github.com/stackrox/rox/central/namespace/dackbox"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/keys/transformation"
)

var (
	// DeploymentTransformationPaths holds the paths to go from a deployment id to the ids of the given category.
	// NOT A COMPLETE REPLACEMENT OF TRANSFORMATIONS BELOW.
	DeploymentTransformationPaths = map[v1.SearchCategory]dackbox.BucketPath{
		v1.SearchCategory_NAMESPACES: dackbox.BackwardsBucketPath(
			deploymentDackBox.BucketHandler,
			nsDackBox.BucketHandler,
		),
	}

	// DeploymentTransformations holds the transformations to go from a deployment id to the ids of the given category.
	DeploymentTransformations = map[v1.SearchCategory]transformation.OneToMany{
		// Deployment (backwards) Namespaces (backwards) Clusters
		v1.SearchCategory_CLUSTERS: transformation.AddPrefix(deploymentDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext(nsDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext(clusterDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(clusterDackBox.Bucket)),

		// Deployment (backwards) Namespaces
		v1.SearchCategory_NAMESPACES: transformation.AddPrefix(deploymentDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext(nsDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(nsDackBox.Bucket)),

		v1.SearchCategory_DEPLOYMENTS: DoNothing,

		// Deployment (forwards) ActiveComponents
		v1.SearchCategory_ACTIVE_COMPONENT: transformation.AddPrefix(deploymentDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext(acDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(acDackBox.Bucket)).
			Then(transformation.Dedupe()),

		// Deployment (forwards) Images
		v1.SearchCategory_IMAGES: transformation.AddPrefix(deploymentDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext(imageDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(imageDackBox.Bucket)),

		// Combine ( { k1, k2 }
		//          Deployments (forwards) Images,
		//          Image (forwards) Components (forwards) CVEs,
		//          )
		v1.SearchCategory_IMAGE_VULN_EDGE: transformation.ForwardEdgeKeys(
			transformation.AddPrefix(deploymentDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(imageDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(imageDackBox.Bucket)),
			transformation.AddPrefix(imageDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)).
				Then(transformation.Dedupe()),
		),

		// Combine ( { k1, k2 }
		//          Deployment (forwards) Images,
		//          Images (forwards) Components,
		//          )
		v1.SearchCategory_IMAGE_COMPONENT_EDGE: transformation.ForwardEdgeKeys(
			transformation.AddPrefix(deploymentDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(imageDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(imageDackBox.Bucket)),
			transformation.AddPrefix(imageDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)),
		),

		// Deployment (forwards) Images (forwards) Components
		v1.SearchCategory_IMAGE_COMPONENTS: transformation.AddPrefix(deploymentDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext(imageDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)).
			Then(transformation.Dedupe()),

		// Combine ( { k1, k2 }
		//          Deployment (forwards) Images (forwards) Components,
		//          Components (forwards) CVEs,
		//          )
		v1.SearchCategory_COMPONENT_VULN_EDGE: transformation.ForwardEdgeKeys(
			transformation.AddPrefix(deploymentDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(imageDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)).
				Then(transformation.Dedupe()),
			transformation.AddPrefix(componentDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)),
		),

		// We don't want to surface cluster level CVEs from a deployment, so just surface CVEs from images it has.
		// Deployment (forwards) Images (forwards) Components (forwards) CVEs
		v1.SearchCategory_VULNERABILITIES: transformation.AddPrefix(deploymentDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext(imageDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
			Then(transformation.Dedupe()).
			ThenMapEachToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)).
			Then(transformation.Dedupe()),

		// We don't want to surface cluster level CVEs from a deployment scope.
		v1.SearchCategory_CLUSTER_VULN_EDGE: ReturnNothing,

		v1.SearchCategory_NODES:               ReturnNothing,
		v1.SearchCategory_NODE_COMPONENT_EDGE: ReturnNothing,
		v1.SearchCategory_NODE_VULN_EDGE:      ReturnNothing,
	}
)
