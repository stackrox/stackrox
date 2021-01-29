package dackbox

import (
	clusterDackBox "github.com/stackrox/rox/central/cluster/dackbox"
	cveDackBox "github.com/stackrox/rox/central/cve/dackbox"
	deploymentDackBox "github.com/stackrox/rox/central/deployment/dackbox"
	imageDackBox "github.com/stackrox/rox/central/image/dackbox"
	componentDackBox "github.com/stackrox/rox/central/imagecomponent/dackbox"
	nsDackBox "github.com/stackrox/rox/central/namespace/dackbox"
	nodeDackBox "github.com/stackrox/rox/central/node/dackbox"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/dackbox/keys/transformation"
	"github.com/stackrox/rox/pkg/features"
)

var (
	// CVETransformations holds the transformations to go from a cve id to the ids of the given category.
	CVETransformations = map[v1.SearchCategory]transformation.OneToMany{
		// We want to surface clusters containing the vuln and clusters containing objects with the vuln.
		// Many(
		//      CVE (backwards) Components (backwards) Images (backwards) Deployments (backwards) Namespaces (backwards) Clusters
		//      CVE (backwards) Cluster,
		//      CVE (backwards) Components (backwards) Nodes (backwards) Clusters
		//     )
		v1.SearchCategory_CLUSTERS: transformation.AddPrefix(cveDackBox.Bucket).
			ThenMapToMany(
				transformation.Many(
					transformation.BackwardFromContext().
						Then(transformation.HasPrefix(componentDackBox.Bucket)).
						ThenMapEachToMany(transformation.BackwardFromContext()).
						Then(transformation.HasPrefix(imageDackBox.Bucket)).
						Then(transformation.Dedupe()).
						ThenMapEachToMany(transformation.BackwardFromContext()).
						Then(transformation.HasPrefix(deploymentDackBox.Bucket)).
						Then(transformation.Dedupe()).
						ThenMapEachToMany(transformation.BackwardFromContext()).
						Then(transformation.Dedupe()).
						Then(transformation.HasPrefix(nsDackBox.Bucket)).
						ThenMapEachToMany(transformation.BackwardFromContext()).
						Then(transformation.Dedupe()).
						Then(transformation.HasPrefix(clusterDackBox.Bucket)).
						ThenMapEachToOne(transformation.StripPrefix(clusterDackBox.Bucket)),
					transformation.BackwardFromContext().
						Then(transformation.HasPrefix(clusterDackBox.Bucket)).
						ThenMapEachToOne(transformation.StripPrefix(clusterDackBox.Bucket)),
				),
			),

		// CVE (backwards) Components (backwards) Images (backwards) Deployments (backwards) Namespaces
		v1.SearchCategory_NAMESPACES: transformation.AddPrefix(cveDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext()).
			Then(transformation.HasPrefix(componentDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext()).
			Then(transformation.HasPrefix(imageDackBox.Bucket)).
			Then(transformation.Dedupe()).
			ThenMapEachToMany(transformation.BackwardFromContext()).
			Then(transformation.Dedupe()).
			Then(transformation.HasPrefix(deploymentDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext()).
			Then(transformation.Dedupe()).
			Then(transformation.HasPrefix(nsDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefix(nsDackBox.Bucket)),

		// CVE (backwards) Components (backwards) Images (backwards) Deployments
		v1.SearchCategory_DEPLOYMENTS: transformation.AddPrefix(cveDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext()).
			Then(transformation.HasPrefix(componentDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext()).
			Then(transformation.HasPrefix(imageDackBox.Bucket)).
			Then(transformation.Dedupe()).
			ThenMapEachToMany(transformation.BackwardFromContext()).
			Then(transformation.HasPrefix(deploymentDackBox.Bucket)).
			Then(transformation.Dedupe()).
			ThenMapEachToOne(transformation.StripPrefix(deploymentDackBox.Bucket)),

		// CVE (backwards) Components (backwards) Images
		v1.SearchCategory_IMAGES: transformation.AddPrefix(cveDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext()).
			Then(transformation.HasPrefix(componentDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext()).
			Then(transformation.HasPrefix(imageDackBox.Bucket)).
			Then(transformation.Dedupe()).
			ThenMapEachToOne(transformation.StripPrefix(imageDackBox.Bucket)),

		// CombineReversed ( { k1, k2 }
		//          CVEs,
		//          CVE (backwards) Components (backwards) Image,
		//          )
		v1.SearchCategory_IMAGE_VULN_EDGE: transformation.ReverseEdgeKeys(
			DoNothing,
			transformation.AddPrefix(cveDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext()).
				Then(transformation.HasPrefix(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefix(componentDackBox.Bucket)).
				ThenMapEachToMany(transformation.BackwardFromContext()).
				Then(transformation.HasPrefix(imageDackBox.Bucket)).
				Then(transformation.Dedupe()).
				ThenMapEachToOne(transformation.StripPrefix(imageDackBox.Bucket)),
		),

		// CombineReversed ( { k2, k1 }
		//          CVE (backwards) Components,
		//          Component (backwards) Images,
		//          )
		v1.SearchCategory_IMAGE_COMPONENT_EDGE: transformation.ReverseEdgeKeys(
			transformation.AddPrefix(cveDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext()).
				Then(transformation.HasPrefix(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefix(componentDackBox.Bucket)),
			transformation.AddPrefix(componentDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext()).
				Then(transformation.HasPrefix(imageDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefix(imageDackBox.Bucket)),
		),

		// CVE (backwards) Components
		v1.SearchCategory_IMAGE_COMPONENTS: transformation.AddPrefix(cveDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext()).
			Then(transformation.HasPrefix(componentDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefix(componentDackBox.Bucket)),

		// CombineReversed ( { k2, k1 }
		//          CVE,
		//          CVE (backwards) Components,
		//          )
		v1.SearchCategory_COMPONENT_VULN_EDGE: transformation.ReverseEdgeKeys(
			DoNothing,
			transformation.AddPrefix(cveDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext()).
				Then(transformation.HasPrefix(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefix(componentDackBox.Bucket)),
		),

		// CVE
		v1.SearchCategory_VULNERABILITIES: DoNothing,

		// CombineReversed ( { k2, k1 }
		//          CVE,
		//          CVE (backwards) Clusters,
		//          )
		v1.SearchCategory_CLUSTER_VULN_EDGE: transformation.ReverseEdgeKeys(
			DoNothing,
			transformation.AddPrefix(cveDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext()).
				Then(transformation.HasPrefix(clusterDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefix(clusterDackBox.Bucket)),
		),
	}

	// This transformation specifically excludes images because we need to attribute the cluster IDs from
	// the nodes which contain this cve
	cveNodeClusterSACTransformation = transformation.AddPrefix(cveDackBox.Bucket).
					ThenMapToMany(transformation.BackwardFromContext().
						Then(transformation.HasPrefix(componentDackBox.Bucket)).
						ThenMapEachToMany(transformation.BackwardFromContext()).
						Then(transformation.HasPrefix(nodeDackBox.Bucket)).
						Then(transformation.Dedupe()).
						ThenMapEachToMany(transformation.BackwardFromContext()).
						Then(transformation.Dedupe()).
						Then(transformation.HasPrefix(clusterDackBox.Bucket)).
						ThenMapEachToOne(transformation.StripPrefix(clusterDackBox.Bucket)))
)

func init() {
	if features.HostScanning.Enabled() {
		CVETransformations[v1.SearchCategory_CLUSTERS] = transformation.AddPrefix(cveDackBox.Bucket).
			ThenMapToMany(
				transformation.Many(
					transformation.BackwardFromContext().
						Then(transformation.HasPrefix(componentDackBox.Bucket)).
						ThenMapEachToMany(transformation.BackwardFromContext()).
						Then(transformation.HasPrefix(imageDackBox.Bucket)).
						Then(transformation.Dedupe()).
						ThenMapEachToMany(transformation.BackwardFromContext()).
						Then(transformation.HasPrefix(deploymentDackBox.Bucket)).
						Then(transformation.Dedupe()).
						ThenMapEachToMany(transformation.BackwardFromContext()).
						Then(transformation.Dedupe()).
						Then(transformation.HasPrefix(nsDackBox.Bucket)).
						ThenMapEachToMany(transformation.BackwardFromContext()).
						Then(transformation.Dedupe()).
						Then(transformation.HasPrefix(clusterDackBox.Bucket)).
						ThenMapEachToOne(transformation.StripPrefix(clusterDackBox.Bucket)),
					transformation.BackwardFromContext().
						Then(transformation.HasPrefix(clusterDackBox.Bucket)).
						ThenMapEachToOne(transformation.StripPrefix(clusterDackBox.Bucket)),
					transformation.BackwardFromContext().
						Then(transformation.HasPrefix(componentDackBox.Bucket)).
						ThenMapEachToMany(transformation.BackwardFromContext()).
						Then(transformation.HasPrefix(nodeDackBox.Bucket)).
						Then(transformation.Dedupe()).
						ThenMapEachToMany(transformation.BackwardFromContext()).
						Then(transformation.Dedupe()).
						Then(transformation.HasPrefix(clusterDackBox.Bucket)).
						ThenMapEachToOne(transformation.StripPrefix(clusterDackBox.Bucket)),
				),
			)

		// CVE (backwards) Components (backwards) Nodes
		CVETransformations[v1.SearchCategory_NODES] = transformation.AddPrefix(cveDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext()).
			Then(transformation.HasPrefix(componentDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext()).
			Then(transformation.HasPrefix(nodeDackBox.Bucket)).
			Then(transformation.Dedupe()).
			ThenMapEachToOne(transformation.StripPrefix(nodeDackBox.Bucket))

		// CombineReversed ( { k1, k2 }
		//          CVEs,
		//          CVE (backwards) Components (backwards) Node,
		//          )
		CVETransformations[v1.SearchCategory_NODE_VULN_EDGE] = transformation.ReverseEdgeKeys(
			DoNothing,
			transformation.AddPrefix(cveDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext()).
				Then(transformation.HasPrefix(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefix(componentDackBox.Bucket)).
				ThenMapEachToMany(transformation.BackwardFromContext()).
				Then(transformation.HasPrefix(nodeDackBox.Bucket)).
				Then(transformation.Dedupe()).
				ThenMapEachToOne(transformation.StripPrefix(nodeDackBox.Bucket)),
		)

		// CombineReversed ( { k2, k1 }
		//          CVE (backwards) Components,
		//          Component (backwards) Nodes,
		//          )
		CVETransformations[v1.SearchCategory_NODE_COMPONENT_EDGE] = transformation.ReverseEdgeKeys(
			transformation.AddPrefix(cveDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext()).
				Then(transformation.HasPrefix(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefix(componentDackBox.Bucket)),
			transformation.AddPrefix(componentDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext()).
				Then(transformation.HasPrefix(nodeDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefix(nodeDackBox.Bucket)),
		)
	}
}
