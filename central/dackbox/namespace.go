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
	// NamespaceTransformations holds the transformations to go from a namespace id to the ids of the given category.
	NamespaceTransformations = map[v1.SearchCategory]transformation.OneToMany{
		// Namespace (backwards) Clusters
		v1.SearchCategory_CLUSTERS: transformation.AddPrefix(nsDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext()).
			Then(transformation.Dedupe()).
			Then(transformation.HasPrefix(clusterDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefix(clusterDackBox.Bucket)),

		// Namespace
		v1.SearchCategory_NAMESPACES: DoNothing,

		// Namespace (forwards) Deployments
		v1.SearchCategory_DEPLOYMENTS: transformation.AddPrefix(nsDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext()).
			Then(transformation.HasPrefix(deploymentDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefix(deploymentDackBox.Bucket)),

		// Namespace (forwards) Deployments (forwards) Images
		v1.SearchCategory_IMAGES: transformation.AddPrefix(nsDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext()).
			Then(transformation.HasPrefix(deploymentDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext()).
			Then(transformation.Dedupe()).
			Then(transformation.HasPrefix(imageDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefix(imageDackBox.Bucket)),

		// Combine ( { k1, k2 }
		//          Namespace (forwards) Deployment (forwards) Images,
		//          Images (forwards) Components,
		//          )
		v1.SearchCategory_IMAGE_COMPONENT_EDGE: transformation.ForwardEdgeKeys(
			transformation.AddPrefix(nsDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext()).
				Then(transformation.HasPrefix(deploymentDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext()).
				Then(transformation.Dedupe()).
				Then(transformation.HasPrefix(imageDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefix(imageDackBox.Bucket)),
			transformation.AddPrefix(imageDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext()).
				Then(transformation.HasPrefix(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefix(componentDackBox.Bucket)),
		),

		// Namespace (forwards) Deployment (forwards) Images (forwards) Components
		v1.SearchCategory_IMAGE_COMPONENTS: transformation.AddPrefix(nsDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext()).
			Then(transformation.HasPrefix(deploymentDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext()).
			Then(transformation.Dedupe()).
			Then(transformation.HasPrefix(imageDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext()).
			Then(transformation.Dedupe()).
			Then(transformation.HasPrefix(componentDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefix(componentDackBox.Bucket)),

		// Combine ( { k1, k2 }
		//          Namespace (forwards) Deployment (forwards) Images (forwards) Components,
		//          Components (forwards) CVEs,
		//          )
		v1.SearchCategory_COMPONENT_VULN_EDGE: transformation.ForwardEdgeKeys(
			transformation.AddPrefix(nsDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext()).
				Then(transformation.HasPrefix(deploymentDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext()).
				Then(transformation.Dedupe()).
				Then(transformation.HasPrefix(imageDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext()).
				Then(transformation.Dedupe()).
				Then(transformation.HasPrefix(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefix(componentDackBox.Bucket)),
			transformation.AddPrefix(componentDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext()).
				Then(transformation.HasPrefix(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefix(cveDackBox.Bucket)),
		),

		// We don't want to surface cluster level CVEs from a deployment, so just surface CVEs from images it has.
		// Namespace (forwards) Deployments (forwards) Images (forwards) Components (forwards) CVEs
		v1.SearchCategory_VULNERABILITIES: transformation.AddPrefix(nsDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext()).
			Then(transformation.HasPrefix(deploymentDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext()).
			Then(transformation.Dedupe()).
			Then(transformation.HasPrefix(imageDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext()).
			Then(transformation.Dedupe()).
			Then(transformation.HasPrefix(componentDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext()).
			Then(transformation.Dedupe()).
			Then(transformation.HasPrefix(cveDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefix(cveDackBox.Bucket)),

		// We don't want to surface cluster level CVEs from a namespace scope.
		v1.SearchCategory_CLUSTER_VULN_EDGE: ReturnNothing,
	}
)
