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
	// ComponentTransformations holds the transformations to go from a component id to the ids of the given category.
	ComponentTransformations = map[v1.SearchCategory]transformation.OneToMany{
		// Many(
		//      Component (backwards) Images (backwards) Deployment (backwards) Namespaces (backwards) Clusters,
		//      Component (backwards) Nodes (backwards) Clusters,
		//     )
		v1.SearchCategory_CLUSTERS: transformation.AddPrefix(componentDackBox.Bucket).
			ThenMapToMany(
				transformation.Many(
					transformation.BackwardFromContext().
						Then(transformation.HasPrefix(imageDackBox.Bucket)).
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
						Then(transformation.HasPrefix(nodeDackBox.Bucket)).
						Then(transformation.Dedupe()).
						ThenMapEachToMany(transformation.BackwardFromContext()).
						Then(transformation.Dedupe()).
						Then(transformation.HasPrefix(clusterDackBox.Bucket)).
						ThenMapEachToOne(transformation.StripPrefix(clusterDackBox.Bucket)),
				),
			),

		// Component (backwards) Images (backwards) Deployment (backwards) Namespaces
		v1.SearchCategory_NAMESPACES: transformation.AddPrefix(componentDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext()).
			Then(transformation.HasPrefix(imageDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext()).
			Then(transformation.HasPrefix(deploymentDackBox.Bucket)).
			Then(transformation.Dedupe()).
			ThenMapEachToMany(transformation.BackwardFromContext()).
			Then(transformation.Dedupe()).
			Then(transformation.HasPrefix(nsDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefix(nsDackBox.Bucket)),

		// Component (backwards) Images (backwards) Deployment
		v1.SearchCategory_DEPLOYMENTS: transformation.AddPrefix(componentDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext()).
			Then(transformation.HasPrefix(imageDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext()).
			Then(transformation.HasPrefix(deploymentDackBox.Bucket)).
			Then(transformation.Dedupe()).
			ThenMapEachToOne(transformation.StripPrefix(deploymentDackBox.Bucket)),

		// Component (backwards) Images
		v1.SearchCategory_IMAGES: transformation.AddPrefix(componentDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext()).
			Then(transformation.HasPrefix(imageDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefix(imageDackBox.Bucket)),

		// Combine ( { k2, k1 }
		//          Components (backwards) Images,
		//          Images (forward) Components (forwards) CVEs,
		//          )
		v1.SearchCategory_IMAGE_VULN_EDGE: transformation.ForwardEdgeKeys(
			transformation.AddPrefix(componentDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext()).
				Then(transformation.HasPrefix(imageDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefix(imageDackBox.Bucket)),
			transformation.AddPrefix(imageDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext()).
				Then(transformation.HasPrefix(componentDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext()).
				Then(transformation.Dedupe()).
				Then(transformation.HasPrefix(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefix(cveDackBox.Bucket)),
		),

		// CombineReversed ( { k2, k1 }
		//          Components,
		//          Components (backwards) Images,
		//          )
		v1.SearchCategory_IMAGE_COMPONENT_EDGE: transformation.ReverseEdgeKeys(
			DoNothing,
			transformation.AddPrefix(componentDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext()).
				Then(transformation.HasPrefix(imageDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefix(imageDackBox.Bucket)),
		),

		// Component (backwards) Nodes
		v1.SearchCategory_NODES: transformation.AddPrefix(componentDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext()).
			Then(transformation.HasPrefix(nodeDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefix(nodeDackBox.Bucket)),

		// Combine ( { k2, k1 }
		//          Components (backwards) Nodes,
		//          Nodes (forwards) Components (forwards) CVEs,
		//          )
		v1.SearchCategory_NODE_VULN_EDGE: transformation.ForwardEdgeKeys(
			transformation.AddPrefix(componentDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext()).
				Then(transformation.HasPrefix(nodeDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefix(nodeDackBox.Bucket)),
			transformation.AddPrefix(nodeDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext()).
				Then(transformation.HasPrefix(componentDackBox.Bucket)).
				ThenMapEachToMany(transformation.ForwardFromContext()).
				Then(transformation.Dedupe()).
				Then(transformation.HasPrefix(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefix(cveDackBox.Bucket)),
		),

		// CombineReversed ( { k2, k1 }
		//          Components,
		//          Components (backwards) Nodes,
		//          )
		v1.SearchCategory_NODE_COMPONENT_EDGE: transformation.ReverseEdgeKeys(
			DoNothing,
			transformation.AddPrefix(componentDackBox.Bucket).
				ThenMapToMany(transformation.BackwardFromContext()).
				Then(transformation.HasPrefix(nodeDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefix(nodeDackBox.Bucket)),
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
				ThenMapToMany(transformation.ForwardFromContext()).
				Then(transformation.HasPrefix(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefix(cveDackBox.Bucket)),
		),

		// We don't want to surface cluster level CVEs from a component, so just surface CVEs from the component itself.
		// Components (forwards) CVEs
		v1.SearchCategory_VULNERABILITIES: transformation.AddPrefix(componentDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext()).
			Then(transformation.HasPrefix(cveDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefix(cveDackBox.Bucket)),

		// We don't want to surface cluster level CVEs from a component scope.
		v1.SearchCategory_CLUSTER_VULN_EDGE: ReturnNothing,
	}

	// This transformation specifically excludes images because we need to attribute the cluster IDs from
	// the nodes which contain this component
	componentNodeClusterSACTransformation = transformation.AddPrefix(componentDackBox.Bucket).
						ThenMapToMany(transformation.BackwardFromContext().
							Then(transformation.HasPrefix(nodeDackBox.Bucket)).
							Then(transformation.Dedupe()).
							ThenMapEachToMany(transformation.BackwardFromContext()).
							Then(transformation.Dedupe()).
							Then(transformation.HasPrefix(clusterDackBox.Bucket)).
							ThenMapEachToOne(transformation.StripPrefix(clusterDackBox.Bucket)))
)
