package dackbox

import (
	acDackBox "github.com/stackrox/stackrox/central/activecomponent/dackbox"
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
	// NamespaceTransformationPaths holds the transformations to go from a namespace id to the ids of the given category.
	// NOT A COMPLETE REPLACEMENT OF TRANSFORMATIONS BELOW.
	NamespaceTransformationPaths = map[v1.SearchCategory]dackbox.BucketPath{
		v1.SearchCategory_NAMESPACES: dackbox.BackwardsBucketPath(nsDackBox.BucketHandler),
	}

	// NamespaceTransformations holds the transformations to go from a namespace id to the ids of the given category.
	NamespaceTransformations = map[v1.SearchCategory]transformation.OneToMany{
		// Namespace (backwards) Clusters
		v1.SearchCategory_CLUSTERS: transformation.AddPrefix(nsDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext(clusterDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(clusterDackBox.Bucket)).
			Then(transformation.Dedupe()),

		// Namespace
		v1.SearchCategory_NAMESPACES: DoNothing,

		// Namespace (forwards) Deployments
		v1.SearchCategory_DEPLOYMENTS: transformation.AddPrefix(nsDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext(deploymentDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(deploymentDackBox.Bucket)),

		// Namespace (forwards) Deployments (forwards) ActiveComponents
		v1.SearchCategory_ACTIVE_COMPONENT: transformation.AddPrefix(nsDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext(deploymentDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext(acDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(acDackBox.Bucket)).
			Then(transformation.Dedupe()),

		// Namespace (forwards) Deployments (forwards) Images
		v1.SearchCategory_IMAGES: transformation.AddPrefix(nsDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext(deploymentDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext(imageDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(imageDackBox.Bucket)).
			Then(transformation.Dedupe()),

		// Combine ( { k1, k2 }
		//          Namespaces (forwards) Deployments (forwards) Images,
		//          Image (forwards) Components (forwards) CVEs,
		//          )
		v1.SearchCategory_IMAGE_VULN_EDGE: transformation.ForwardEdgeKeys(
			transformation.AddPrefix(nsDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(deploymentDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext(imageDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(imageDackBox.Bucket)).
				Then(transformation.Dedupe()),
			transformation.AddPrefix(imageDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)).
				Then(transformation.Dedupe()),
		),

		// Combine ( { k1, k2 }
		//          Namespace (forwards) Deployment (forwards) Images,
		//          Images (forwards) Components,
		//          )
		v1.SearchCategory_IMAGE_COMPONENT_EDGE: transformation.ForwardEdgeKeys(
			transformation.AddPrefix(nsDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(deploymentDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext(imageDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(imageDackBox.Bucket)).
				Then(transformation.Dedupe()),
			transformation.AddPrefix(imageDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)),
		),

		// Namespace (forwards) Deployment (forwards) Images (forwards) Components
		v1.SearchCategory_IMAGE_COMPONENTS: transformation.AddPrefix(nsDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext(deploymentDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext(imageDackBox.Bucket)).
			Then(transformation.Dedupe()).
			ThenMapEachToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)).
			Then(transformation.Dedupe()),

		// Combine ( { k1, k2 }
		//          Namespace (forwards) Deployment (forwards) Images (forwards) Components,
		//          Components (forwards) CVEs,
		//          )
		v1.SearchCategory_COMPONENT_VULN_EDGE: transformation.ForwardEdgeKeys(
			transformation.AddPrefix(nsDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(deploymentDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext(imageDackBox.Bucket)).
				Then(transformation.Dedupe()).
				ThenMapEachToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)).
				Then(transformation.Dedupe()),
			transformation.AddPrefix(componentDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)),
		),

		// We don't want to surface cluster level CVEs from a deployment, so just surface CVEs from images it has.
		// Namespace (forwards) Deployments (forwards) Images (forwards) Components (forwards) CVEs
		v1.SearchCategory_VULNERABILITIES: transformation.AddPrefix(nsDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext(deploymentDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext(imageDackBox.Bucket)).
			Then(transformation.Dedupe()).
			ThenMapEachToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
			Then(transformation.Dedupe()).
			ThenMapEachToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)).
			Then(transformation.Dedupe()),

		// We don't want to surface cluster level CVEs from a namespace scope.
		v1.SearchCategory_CLUSTER_VULN_EDGE: ReturnNothing,

		v1.SearchCategory_NODES:               ReturnNothing,
		v1.SearchCategory_NODE_COMPONENT_EDGE: ReturnNothing,
		v1.SearchCategory_NODE_VULN_EDGE:      ReturnNothing,
	}
)
