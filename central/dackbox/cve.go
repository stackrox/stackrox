package dackbox

import (
	acDackbox "github.com/stackrox/rox/central/activecomponent/dackbox"
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
	// CVETransformationPaths will hold the transformations to go from a CVE ID to the IDs of the given category.
	// CAUTION: This is in the process of being migrated, and does not contain an entry for every applicable
	// category yet.
	CVETransformationPaths = map[v1.SearchCategory]dackbox.BucketPath{
		v1.SearchCategory_NAMESPACES: dackbox.BackwardsBucketPath(
			cveDackBox.BucketHandler,
			componentDackBox.BucketHandler,
			imageDackBox.BucketHandler,
			deploymentDackBox.BucketHandler,
			nsDackBox.BucketHandler,
		),
	}

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

		// CVE (backwards) Components (backwards) ActiveComponents
		v1.SearchCategory_ACTIVE_COMPONENT: transformation.AddPrefix(cveDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext(componentDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext(acDackbox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(acDackbox.Bucket)).
			Then(transformation.Dedupe()),

		// CVE (backwards) Components (backwards) Images
		v1.SearchCategory_IMAGES: transformation.AddPrefix(cveDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext(componentDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext(imageDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(imageDackBox.Bucket)).
			Then(transformation.Dedupe()),

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

		// CombineReversed ( { k1, k2 }
		//          CVEs,
		//          CVE (backwards) Image,
		//          )
		v1.SearchCategory_IMAGE_VULN_EDGE: transformation.ReverseEdgeKeys(
			DoNothing,
			transformation.AddPrefix(cveDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext(imageDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(imageDackBox.Bucket)).
				Then(transformation.Dedupe()),
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

	// CVEToImageBucketPath maps a cve to whether or not an image exists that contains that cve
	CVEToImageBucketPath = dackbox.BackwardsBucketPath(
		cveDackBox.BucketHandler,
		componentDackBox.BucketHandler,
		imageDackBox.BucketHandler,
	)

	// CVEToNodeBucketPath maps a cve to whether or not a node exists that contains that cve
	CVEToNodeBucketPath = dackbox.BackwardsBucketPath(
		cveDackBox.BucketHandler,
		componentDackBox.BucketHandler,
		nodeDackBox.BucketHandler,
	)

	// CVEToClusterBucketPath maps a cve to whether or not a cluster exists that contains that cve
	CVEToClusterBucketPath = dackbox.BackwardsBucketPath(
		cveDackBox.BucketHandler,
		clusterDackBox.BucketHandler,
	)
)
