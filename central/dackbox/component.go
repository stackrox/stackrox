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
	// ComponentTransformationPaths holds the paths to go from a component id to the ids of the given category.
	// NOT A COMPLETE REPLACEMENT OF TRANSFORMATIONS BELOW
	ComponentTransformationPaths = map[v1.SearchCategory]dackbox.BucketPath{
		v1.SearchCategory_NAMESPACES: dackbox.BackwardsBucketPath(
			componentDackBox.BucketHandler,
			imageDackBox.BucketHandler,
			deploymentDackBox.BucketHandler,
			nsDackBox.BucketHandler,
		),
	}

	// ComponentTransformations holds the transformations to go from a component id to the ids of the given category.
	ComponentTransformations = map[v1.SearchCategory]transformation.OneToMany{
		// Many(
		//      Component (backwards) Images (backwards) Deployment (backwards) Namespaces (backwards) Clusters,
		//      Component (backwards) Nodes (backwards) Clusters,
		//     )
		v1.SearchCategory_CLUSTERS: transformation.AddPrefix(componentDackBox.Bucket).
			ThenMapToMany(
				transformation.Many(
					transformation.BackwardFromContext(imageDackBox.Bucket).
						ThenMapEachToMany(transformation.BackwardFromContext(deploymentDackBox.Bucket)).
						Then(transformation.Dedupe()).
						ThenMapEachToMany(transformation.BackwardFromContext(nsDackBox.Bucket)).
						Then(transformation.Dedupe()).
						ThenMapEachToMany(transformation.BackwardFromContext(clusterDackBox.Bucket)).
						ThenMapEachToOne(transformation.StripPrefixUnchecked(clusterDackBox.Bucket)).
						Then(transformation.Dedupe()),
					transformation.BackwardFromContext(nodeDackBox.Bucket).
						Then(transformation.Dedupe()).
						ThenMapEachToMany(transformation.BackwardFromContext(clusterDackBox.Bucket)).
						ThenMapEachToOne(transformation.StripPrefixUnchecked(clusterDackBox.Bucket)).
						Then(transformation.Dedupe()),
				),
			),

		// Component (backwards) Images (backwards) Deployment (backwards) Namespaces
		v1.SearchCategory_NAMESPACES: transformation.AddPrefix(componentDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext(imageDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext(deploymentDackBox.Bucket)).
			Then(transformation.Dedupe()).
			ThenMapEachToMany(transformation.BackwardFromContext(nsDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(nsDackBox.Bucket)).
			Then(transformation.Dedupe()),

		// Component (backwards) Images (backwards) Deployment
		v1.SearchCategory_DEPLOYMENTS: transformation.AddPrefix(componentDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext(imageDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext(deploymentDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(deploymentDackBox.Bucket)).
			Then(transformation.Dedupe()),

		// Component (backwards) ActiveComponents
		v1.SearchCategory_ACTIVE_COMPONENT: transformation.AddPrefix(componentDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext(acDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(acDackBox.Bucket)).
			Then(transformation.Dedupe()),

		// Component (backwards) Images
		v1.SearchCategory_IMAGES: transformation.AddPrefix(componentDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext(imageDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(imageDackBox.Bucket)),

		// Combine ( { k2, k1 }
		//          Components (backwards) Images,
		//          Images (forward) Components (forwards) CVEs,
		//          )
		v1.SearchCategory_IMAGE_VULN_EDGE: transformation.ForwardEdgeKeys(
			transformation.AddPrefix(componentDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext(imageDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(imageDackBox.Bucket)),
			transformation.AddPrefix(imageDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)).
				Then(transformation.Dedupe()),
		),

		// CombineReversed ( { k2, k1 }
		//          Components,
		//          Components (backwards) Images,
		//          )
		v1.SearchCategory_IMAGE_COMPONENT_EDGE: transformation.ReverseEdgeKeys(
			DoNothing,
			transformation.AddPrefix(componentDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext(imageDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(imageDackBox.Bucket)),
		),

		// Component (backwards) Nodes
		v1.SearchCategory_NODES: transformation.AddPrefix(componentDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext(nodeDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(nodeDackBox.Bucket)),

		// Combine ( { k2, k1 }
		//          Components (backwards) Nodes,
		//          Nodes (forwards) Components (forwards) CVEs,
		//          )
		v1.SearchCategory_NODE_VULN_EDGE: transformation.ForwardEdgeKeys(
			transformation.AddPrefix(componentDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext(nodeDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(nodeDackBox.Bucket)),
			transformation.AddPrefix(nodeDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)).
				Then(transformation.Dedupe()),
		),

		v1.SearchCategory_NODE_COMPONENT_EDGE: transformation.ReverseEdgeKeys(
			DoNothing,
			transformation.AddPrefix(componentDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext(nodeDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(nodeDackBox.Bucket)),
		),

		// Components
		v1.SearchCategory_IMAGE_COMPONENTS: DoNothing,

		// Combine ( { k1, k2 }
		//          Components,
		//          Components (forwards) CVEs,
		//          )
		v1.SearchCategory_COMPONENT_VULN_EDGE: transformation.ForwardEdgeKeys(
			DoNothing,
			transformation.AddPrefix(componentDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)),
		),

		// We don't want to surface cluster level CVEs from a component, so just surface CVEs from the component itself.
		// Components (forwards) CVEs
		v1.SearchCategory_VULNERABILITIES: transformation.AddPrefix(componentDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)),

		// We don't want to surface cluster level CVEs from a component scope.
		v1.SearchCategory_CLUSTER_VULN_EDGE: ReturnNothing,
	}

	// ComponentToImageBucketPath maps a component to whether an image exists that contains that component.
	ComponentToImageBucketPath = dackbox.BackwardsBucketPath(
		componentDackBox.BucketHandler,
		imageDackBox.BucketHandler,
	)

	// ComponentToNodeBucketPath maps a component to whether a node exists that contains that component.
	ComponentToNodeBucketPath = dackbox.BackwardsBucketPath(
		componentDackBox.BucketHandler,
		nodeDackBox.BucketHandler,
	)
)
