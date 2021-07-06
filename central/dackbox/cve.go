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
					transformation.BackwardFromContext(componentDackBox.Bucket).
						ThenMapEachToMany(transformation.BackwardFromContext(imageDackBox.Bucket)).
						Then(transformation.Dedupe()).
						ThenMapEachToMany(transformation.BackwardFromContext(deploymentDackBox.Bucket)).
						Then(transformation.Dedupe()).
						ThenMapEachToMany(transformation.BackwardFromContext(nsDackBox.Bucket)).
						Then(transformation.Dedupe()).
						ThenMapEachToMany(transformation.BackwardFromContext(clusterDackBox.Bucket)).
						ThenMapEachToOne(transformation.StripPrefixUnchecked(clusterDackBox.Bucket)).
						Then(transformation.Dedupe()),
					transformation.BackwardFromContext(clusterDackBox.Bucket).
						ThenMapEachToOne(transformation.StripPrefixUnchecked(clusterDackBox.Bucket)),
					transformation.BackwardFromContext(componentDackBox.Bucket).
						ThenMapEachToMany(transformation.BackwardFromContext(nodeDackBox.Bucket)).
						Then(transformation.Dedupe()).
						ThenMapEachToMany(transformation.BackwardFromContext(clusterDackBox.Bucket)).
						ThenMapEachToOne(transformation.StripPrefixUnchecked(clusterDackBox.Bucket)).
						Then(transformation.Dedupe()),
				),
			),

		// CVE (backwards) Components (backwards) Images (backwards) Deployments (backwards) Namespaces
		v1.SearchCategory_NAMESPACES: transformation.AddPrefix(cveDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext(componentDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext(imageDackBox.Bucket)).
			Then(transformation.Dedupe()).
			ThenMapEachToMany(transformation.BackwardFromContext(deploymentDackBox.Bucket)).
			Then(transformation.Dedupe()).
			ThenMapEachToMany(transformation.BackwardFromContext(nsDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(nsDackBox.Bucket)).
			Then(transformation.Dedupe()),

		// CVE (backwards) Components (backwards) Images (backwards) Deployments
		v1.SearchCategory_DEPLOYMENTS: transformation.AddPrefix(cveDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext(componentDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext(imageDackBox.Bucket)).
			Then(transformation.Dedupe()).
			ThenMapEachToMany(transformation.BackwardFromContext(deploymentDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(deploymentDackBox.Bucket)).
			Then(transformation.Dedupe()),

		// CVE (backwards) Components (backwards) Images
		v1.SearchCategory_IMAGES: transformation.AddPrefix(cveDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext(componentDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext(imageDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(imageDackBox.Bucket)).
			Then(transformation.Dedupe()),

		// CombineReversed ( { k1, k2 }
		//          CVEs,
		//          CVE (backwards) Components (backwards) Image,
		//          )
		v1.SearchCategory_IMAGE_VULN_EDGE: transformation.ReverseEdgeKeys(
			DoNothing,
			transformation.AddPrefix(cveDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)).
				ThenMapEachToMany(transformation.BackwardFromContext(imageDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(imageDackBox.Bucket)).
				Then(transformation.Dedupe()),
		),

		// CombineReversed ( { k2, k1 }
		//          CVE (backwards) Components,
		//          Component (backwards) Images,
		//          )
		v1.SearchCategory_IMAGE_COMPONENT_EDGE: transformation.ReverseEdgeKeys(
			transformation.AddPrefix(cveDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)),
			transformation.AddPrefix(componentDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext(imageDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(imageDackBox.Bucket)),
		),

		// CVE (backwards) Components (backwards) Nodes
		v1.SearchCategory_NODES: transformation.AddPrefix(cveDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext(componentDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext(nodeDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(nodeDackBox.Bucket)).
			Then(transformation.Dedupe()),

		// CombineReversed ( { k1, k2 }
		//          CVEs,
		//          CVE (backwards) Components (backwards) Node,
		//          )
		v1.SearchCategory_NODE_VULN_EDGE: transformation.ReverseEdgeKeys(
			DoNothing,
			transformation.AddPrefix(cveDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)).
				ThenMapEachToMany(transformation.BackwardFromContext(nodeDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(nodeDackBox.Bucket)).
				Then(transformation.Dedupe()),
		),

		// CombineReversed ( { k2, k1 }
		//          CVE (backwards) Components,
		//          Component (backwards) Nodes,
		//          )
		v1.SearchCategory_NODE_COMPONENT_EDGE: transformation.ReverseEdgeKeys(
			transformation.AddPrefix(cveDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)),
			transformation.AddPrefix(componentDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext(nodeDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(nodeDackBox.Bucket)),
		),

		// CVE (backwards) Components
		v1.SearchCategory_IMAGE_COMPONENTS: transformation.AddPrefix(cveDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext(componentDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)),

		// CombineReversed ( { k2, k1 }
		//          CVE,
		//          CVE (backwards) Components,
		//          )
		v1.SearchCategory_COMPONENT_VULN_EDGE: transformation.ReverseEdgeKeys(
			DoNothing,
			transformation.AddPrefix(cveDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)),
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
				ThenMapToMany(transformation.BackwardFromContext(clusterDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(clusterDackBox.Bucket)),
		),
	}

	// This transformation specifically excludes images because we need to attribute the cluster IDs from
	// the nodes which contain this cve
	cveNodeClusterSACTransformation = transformation.AddPrefix(cveDackBox.Bucket).
					ThenMapToMany(transformation.BackwardFromContext(componentDackBox.Bucket)).
					ThenMapEachToMany(transformation.BackwardFromContext(nodeDackBox.Bucket)).
					Then(transformation.Dedupe()).
					ThenMapEachToMany(transformation.BackwardFromContext(clusterDackBox.Bucket)).
					ThenMapEachToOne(transformation.StripPrefixUnchecked(clusterDackBox.Bucket)).
					Then(transformation.Dedupe())

	// CVEToImageExistenceTransformation maps a cve to whether or not an image exists that contains that cve
	CVEToImageExistenceTransformation = transformation.AddPrefix(cveDackBox.Bucket).
						ThenMapToMany(transformation.BackwardFromContext(componentDackBox.Bucket)).
						ThenMapEachToBool(transformation.BackwardExistence(imageDackBox.Bucket))

	// CVEToNodeExistenceTransformation maps a cve to whether or not a node exists that contains that cve
	CVEToNodeExistenceTransformation = transformation.AddPrefix(cveDackBox.Bucket).
						ThenMapToMany(transformation.BackwardFromContext(componentDackBox.Bucket)).
						ThenMapEachToBool(transformation.BackwardExistence(nodeDackBox.Bucket))

	// CVEToClusterExistenceTransformation maps a cve to whether or not a cluster exists that contains that cve
	CVEToClusterExistenceTransformation = transformation.AddPrefix(cveDackBox.Bucket).
						ThenMapToBool(transformation.BackwardExistence(clusterDackBox.Bucket))
)
