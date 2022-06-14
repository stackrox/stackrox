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
	// ActiveComponentTransformationPaths holds the paths to go from an active component id to the ids of the given category.
	// NOT A COMPLETE REPLACEMENT OF TRANSFORMATIONS BELOW.
	ActiveComponentTransformationPaths = map[v1.SearchCategory]dackbox.BucketPath{
		v1.SearchCategory_NAMESPACES: dackbox.BackwardsBucketPath(
			acDackBox.BucketHandler,
			deploymentDackBox.BucketHandler,
			nsDackBox.BucketHandler,
		),
	}

	// ActiveComponentTransformations holds the transformations to go from a deployment:image_component id to the ids of the given category.
	ActiveComponentTransformations = map[v1.SearchCategory]transformation.OneToMany{

		// ActiveComponent (backwards) Deployment (backwards) Namespaces (backwards) Clusters
		v1.SearchCategory_CLUSTERS: transformation.AddPrefix(acDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext(deploymentDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext(nsDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext(clusterDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(clusterDackBox.Bucket)),

		// ActiveComponent (backwards) Deployment (backwards) Namespace
		v1.SearchCategory_NAMESPACES: transformation.AddPrefix(acDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext(deploymentDackBox.Bucket)).
			ThenMapEachToMany(transformation.BackwardFromContext(nsDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(nsDackBox.Bucket)),

		v1.SearchCategory_NODES:               ReturnNothing,
		v1.SearchCategory_NODE_COMPONENT_EDGE: ReturnNothing,
		v1.SearchCategory_NODE_VULN_EDGE:      ReturnNothing,

		// ActiveComponent (backwards) Deployment
		v1.SearchCategory_DEPLOYMENTS: transformation.AddPrefix(acDackBox.Bucket).
			ThenMapToMany(transformation.BackwardFromContext(deploymentDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(deploymentDackBox.Bucket)),

		// ActiveComponent
		v1.SearchCategory_ACTIVE_COMPONENT: DoNothing,

		// Intersect (
		//    ActiveComponent (backwards) deployment (forwards) Images
		//    ActiveComponent (forwards) component (backwards) Images
		// )
		v1.SearchCategory_IMAGES: transformation.AddPrefix(acDackBox.Bucket).
			ThenMapToMany(
				transformation.Intersect(
					transformation.BackwardFromContext(deploymentDackBox.Bucket).
						ThenMapEachToMany(transformation.ForwardFromContext(imageDackBox.Bucket)).
						ThenMapEachToOne(transformation.StripPrefixUnchecked(imageDackBox.Bucket)),
					transformation.ForwardFromContext(componentDackBox.Bucket).
						ThenMapEachToMany(transformation.BackwardFromContext(imageDackBox.Bucket)).
						ThenMapEachToOne(transformation.StripPrefixUnchecked(imageDackBox.Bucket)),
				),
			),

		v1.SearchCategory_IMAGE_VULN_EDGE:      ReturnNothing,
		v1.SearchCategory_IMAGE_COMPONENT_EDGE: ReturnNothing,

		// ActiveComponent (forwards) Component
		v1.SearchCategory_IMAGE_COMPONENTS: transformation.AddPrefix(acDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)),

		// Combine ( { k1, k2 }
		//          ActiveComponent (forwards) Component,
		//          Component (forwards) CVEs,
		//          )
		v1.SearchCategory_COMPONENT_VULN_EDGE: transformation.ForwardEdgeKeys(
			transformation.AddPrefix(acDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)),
			transformation.AddPrefix(componentDackBox.Bucket).
				ThenMapToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
				ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)),
		),

		// ActiveComponent (forwards) Component (forward) CVEs
		v1.SearchCategory_VULNERABILITIES: transformation.AddPrefix(acDackBox.Bucket).
			ThenMapToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
			ThenMapEachToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
			ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)),

		v1.SearchCategory_CLUSTER_VULN_EDGE: ReturnNothing,
	}
)
