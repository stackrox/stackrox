package dackbox

import (
	acDackBox "github.com/stackrox/rox/central/activecomponent/dackbox"
	clusterDackBox "github.com/stackrox/rox/central/cluster/dackbox"
	cveDackBox "github.com/stackrox/rox/central/cve/dackbox"
	deploymentDackBox "github.com/stackrox/rox/central/deployment/dackbox"
	imageDackBox "github.com/stackrox/rox/central/image/dackbox"
	componentDackBox "github.com/stackrox/rox/central/imagecomponent/dackbox"
	nsDackBox "github.com/stackrox/rox/central/namespace/dackbox"
	nodeDackBox "github.com/stackrox/rox/central/node/dackbox"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/keys/transformation"
)

var (
	// ClusterTransformationPaths holds the transformation paths to go from a cluster id to IDs of the given
	// category.
	ClusterTransformationPaths = map[v1.SearchCategory]dackbox.BucketPath{
		v1.SearchCategory_CLUSTERS: dackbox.BackwardsBucketPath(
			clusterDackBox.BucketHandler,
		),
	}
	// ClusterTransformations holds the transformations to go from a cluster id to the ids of the given category.
	ClusterTransformations = map[v1.SearchCategory]transformation.OneToMany{
		// Cluster
		v1.SearchCategory_CLUSTERS: DoNothing,

		// Cluster (forwards) Namespaces
		v1.SearchCategory_NAMESPACES: transformation.AddPrefix(clusterDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext(nsDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(nsDackBox.Bucket)),

		// Cluster (forwards) Namespaces (forwards) Deployments
		v1.SearchCategory_DEPLOYMENTS: transformation.AddPrefix(clusterDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext(nsDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext(deploymentDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(deploymentDackBox.Bucket)),

		// Cluster (forwards) Namespaces (forwards) Deployments (forwards) ActiveComponents
		v1.SearchCategory_ACTIVE_COMPONENT: transformation.AddPrefix(clusterDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext(nsDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext(deploymentDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext(acDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(acDackBox.Bucket)).
			Then(transformation.Dedupe()),

		// Cluster (forwards) Namespaces (forwards) Deployments (forwards) Images
		v1.SearchCategory_IMAGES: transformation.AddPrefix(clusterDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext(nsDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext(deploymentDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext(imageDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(imageDackBox.Bucket)).
			Then(transformation.Dedupe()),

		// Combine ( { k1, k2 }
		//          Cluster (forwards) Namespaces (forwards) Deployments (forwards) Images,
		//          Image (forwards) Components (forwards) CVEs,
		//          )
		v1.SearchCategory_IMAGE_VULN_EDGE: transformation.ForwardEdgeKeys(
			transformation.AddPrefix(clusterDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(nsDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext(deploymentDackBox.Bucket)).
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
		//          Cluster (forwards) Namespaces (forwards) Deployments (forwards) Images,
		//          Images (forwards) Components,
		//          )
		v1.SearchCategory_IMAGE_COMPONENT_EDGE: transformation.ForwardEdgeKeys(
			transformation.AddPrefix(clusterDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(nsDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext(deploymentDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext(imageDackBox.Bucket)).
				Then(transformation.Dedupe()).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(imageDackBox.Bucket)),
			transformation.AddPrefix(imageDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)),
		),

		// Cluster (forwards) Namespaces (forwards) Deployments (forwards) Images (forwards) Components
		v1.SearchCategory_IMAGE_COMPONENTS: transformation.Many(
			transformation.AddPrefix(clusterDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(nsDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext(deploymentDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext(imageDackBox.Bucket)).
				Then(transformation.Dedupe()).
				ThenMapEachToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)).
				Then(transformation.Dedupe()),
			transformation.AddPrefix(clusterDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(nodeDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)).
				Then(transformation.Dedupe()),
		),

		// Cluster (forwards) Nodes
		v1.SearchCategory_NODES: transformation.AddPrefix(clusterDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext(nodeDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(nodeDackBox.Bucket)),

		// Combine ( { k1, k2 }
		//          Cluster (forwards) Nodes,
		//          Node (forwards) Components (forwards) CVEs,
		//          )
		v1.SearchCategory_NODE_VULN_EDGE: transformation.ForwardEdgeKeys(
			transformation.AddPrefix(clusterDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(nodeDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(nodeDackBox.Bucket)),
			transformation.AddPrefix(nodeDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)).
				Then(transformation.Dedupe()),
		),

		// Combine ( { k1, k2 }
		//          Cluster (forwards) Nodes,
		//          Nodes (forwards) Components,
		//          )
		v1.SearchCategory_NODE_COMPONENT_EDGE: transformation.ForwardEdgeKeys(
			transformation.AddPrefix(clusterDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(nodeDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(nodeDackBox.Bucket)),
			transformation.AddPrefix(nodeDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)),
		),

		// Combine ( { k1, k2 }
		//          Cluster (forwards) Namespaces (forwards) Deployments (forwards) Images (forwards) Components,
		//          Components (forwards) CVEs,
		//          )
		v1.SearchCategory_COMPONENT_VULN_EDGE: transformation.ForwardEdgeKeys(
			transformation.AddPrefix(clusterDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(nsDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext(deploymentDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext(imageDackBox.Bucket)).
				Then(transformation.Dedupe()).
				ThenMapEachToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)).
				Then(transformation.Dedupe()),
			transformation.AddPrefix(componentDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)),
		),

		// We want to surface both Vuln in objects in the cluster, and vulns attributed to the cluster itself.
		// Many(
		//      Cluster (forwards) Namespaces (forwards) Deployments (forwards) Images (forwards) Components (forwards) CVEs
		//      Cluster (forwards) CVEs,
		//      Cluster (forwards) Nodes (forwards) Components (forwards) CVEs,
		//     )
		v1.SearchCategory_VULNERABILITIES: transformation.Many(
			transformation.AddPrefix(clusterDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(nsDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext(deploymentDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext(imageDackBox.Bucket)).
				Then(transformation.Dedupe()).
				ThenMapEachToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
				Then(transformation.Dedupe()).
				ThenMapEachToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)).
				Then(transformation.Dedupe()),
			transformation.AddPrefix(clusterDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)),
			transformation.AddPrefix(clusterDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(nodeDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
				Then(transformation.Dedupe()).
				ThenMapEachToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)).
				Then(transformation.Dedupe()),
		),

		// Combine ( { k1, k2 }
		//          Cluster,
		//          Cluster (forwards) CVEs,
		//          )
		v1.SearchCategory_CLUSTER_VULN_EDGE: transformation.ForwardEdgeKeys(
			DoNothing,
			transformation.AddPrefix(clusterDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)),
		),
	}
)
