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
	// ClusterTransformations holds the transformations to go from a cluster id to the ids of the given category.
	ClusterTransformations = map[v1.SearchCategory]transformation.OneToMany{
		// Cluster
		v1.SearchCategory_CLUSTERS: DoNothing,

		// Cluster (forwards) Namespaces
		v1.SearchCategory_NAMESPACES: transformation.AddPrefix(clusterDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext()).
			Then(transformation.HasPrefix(nsDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefix(nsDackBox.Bucket)),

		// Cluster (forwards) Namespaces (forwards) Deployments
		v1.SearchCategory_DEPLOYMENTS: transformation.AddPrefix(clusterDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext()).
			Then(transformation.HasPrefix(nsDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext()).
			Then(transformation.HasPrefix(deploymentDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefix(deploymentDackBox.Bucket)),

		// Cluster (forwards) Namespaces (forwards) Deployments (forwards) Images
		v1.SearchCategory_IMAGES: transformation.AddPrefix(clusterDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext()).
			Then(transformation.HasPrefix(nsDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext()).
			Then(transformation.HasPrefix(deploymentDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext()).
			Then(transformation.Dedupe()).
			Then(transformation.HasPrefix(imageDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefix(imageDackBox.Bucket)),

		// Combine ( { k1, k2 }
		//          Cluster (forwards) Namespaces (forwards) Deployments (forwards) Images,
		//          Images (forwards) Components,
		//          )
		v1.SearchCategory_IMAGE_COMPONENT_EDGE: transformation.ForwardEdgeKeys(
			transformation.AddPrefix(clusterDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext()).
				Then(transformation.HasPrefix(nsDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext()).
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

		// Cluster (forwards) Namespaces (forwards) Deployments (forwards) Images (forwards) Components
		v1.SearchCategory_IMAGE_COMPONENTS: transformation.AddPrefix(clusterDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext()).
			Then(transformation.HasPrefix(nsDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext()).
			Then(transformation.HasPrefix(deploymentDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext()).
			Then(transformation.Dedupe()).
			Then(transformation.HasPrefix(imageDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext()).
			Then(transformation.Dedupe()).
			Then(transformation.HasPrefix(componentDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefix(componentDackBox.Bucket)),

		// Combine ( { k1, k2 }
		//          Cluster (forwards) Namespaces (forwards) Deployments (forwards) Images (forwards) Components,
		//          Components (forwards) CVEs,
		//          )
		v1.SearchCategory_COMPONENT_VULN_EDGE: transformation.ForwardEdgeKeys(
			transformation.AddPrefix(clusterDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext()).
				Then(transformation.HasPrefix(nsDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext()).
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

		// We want to surface both Vuln in objects in the cluster, and vulns attributed to the cluster itself.
		// Both(
		//      Cluster (forwards) Namespaces (forwards) Deployments (forwards) Images (forwards) Components (forwards) CVEs
		//      Cluster (forwards) CVEs,
		//     )
		v1.SearchCategory_VULNERABILITIES: transformation.Both(
			transformation.AddPrefix(clusterDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext()).
				Then(transformation.HasPrefix(nsDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext()).
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
			transformation.AddPrefix(clusterDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext()).
				Then(transformation.HasPrefix(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefix(cveDackBox.Bucket)),
		),

		// Combine ( { k1, k2 }
		//          Cluster,
		//          Cluster (forwards) CVEs,
		//          )
		v1.SearchCategory_CLUSTER_VULN_EDGE: transformation.ForwardEdgeKeys(
			DoNothing,
			transformation.AddPrefix(clusterDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext()).
				Then(transformation.HasPrefix(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefix(cveDackBox.Bucket)),
		),
	}
)
