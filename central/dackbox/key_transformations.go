package dackbox

import (
	clusterDackBox "github.com/stackrox/stackrox/central/cluster/dackbox"
	cveDackBox "github.com/stackrox/stackrox/central/cve/dackbox"
	deploymentDackBox "github.com/stackrox/stackrox/central/deployment/dackbox"
	imageDackBox "github.com/stackrox/stackrox/central/image/dackbox"
	componentDackBox "github.com/stackrox/stackrox/central/imagecomponent/dackbox"
	nsDackBox "github.com/stackrox/stackrox/central/namespace/dackbox"
	"github.com/stackrox/stackrox/pkg/dackbox/keys/transformation"
)

var (
	// ClusterToCVETransformation uses a graph context to transform a cluster ID into a cve ID.
	ClusterToCVETransformation = transformation.AddPrefix(clusterDackBox.Bucket).
					ThenMapToMany(transformation.ForwardFromContext(nsDackBox.Bucket)).
					ThenMapEachToMany(transformation.ForwardFromContext(deploymentDackBox.Bucket)).
					ThenMapEachToMany(transformation.ForwardFromContext(imageDackBox.Bucket)).
					Then(transformation.Dedupe()).
					ThenMapEachToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
					Then(transformation.Dedupe()).
					ThenMapEachToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
					ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)).
					Then(transformation.Dedupe())

	// ClusterToClusterCVETransformation uses a graph context to transform a cluster ID into a cluster cve ID.
	ClusterToClusterCVETransformation = transformation.AddPrefix(clusterDackBox.Bucket).
						ThenMapToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
						ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket))

	// DeploymentToImageComponentTransformation uses a graph context to transform a deployment ID into a component ID.
	DeploymentToImageComponentTransformation = transformation.AddPrefix(deploymentDackBox.Bucket).
							ThenMapToMany(transformation.ForwardFromContext(imageDackBox.Bucket)).
							ThenMapEachToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
							ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket)).
							Then(transformation.Dedupe())

	// DeploymentToCVETransformation uses a graph context to transform a deployment ID into a cve ID.
	DeploymentToCVETransformation = transformation.AddPrefix(deploymentDackBox.Bucket).
					ThenMapToMany(transformation.ForwardFromContext(imageDackBox.Bucket)).
					ThenMapEachToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
					Then(transformation.Dedupe()).
					ThenMapEachToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
					ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)).
					Then(transformation.Dedupe())

	// ImageToDeploymentTransformation uses a graph context to transform a image ID into a Deployment ID.
	ImageToDeploymentTransformation = transformation.AddPrefix(imageDackBox.Bucket).
					ThenMapToMany(transformation.BackwardFromContext(deploymentDackBox.Bucket)).
					ThenMapEachToOne(transformation.StripPrefixUnchecked(deploymentDackBox.Bucket))

	// ImageToImageComponentTransformation trasforms an image id to a set of image component ids.
	ImageToImageComponentTransformation = transformation.AddPrefix(imageDackBox.Bucket).
						ThenMapToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
						ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket))

	// ImageToCVETransformation uses a graph context to transform a image ID into a cve ID.
	ImageToCVETransformation = transformation.AddPrefix(imageDackBox.Bucket).
					ThenMapToMany(transformation.ForwardFromContext(componentDackBox.Bucket)).
					ThenMapEachToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
					ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket)).
					Then(transformation.Dedupe())

	// ComponentToDeploymentTransformation uses a graph context to transform a component ID into a Deployment ID.
	ComponentToDeploymentTransformation = transformation.AddPrefix(componentDackBox.Bucket).
						ThenMapToMany(transformation.BackwardFromContext(imageDackBox.Bucket)).
						ThenMapEachToMany(transformation.BackwardFromContext(deploymentDackBox.Bucket)).
						ThenMapEachToOne(transformation.StripPrefixUnchecked(deploymentDackBox.Bucket)).
						Then(transformation.Dedupe())

	// ComponentToImageTransformation uses a graph context to transform a component ID into an image ID.
	ComponentToImageTransformation = transformation.AddPrefix(componentDackBox.Bucket).
					ThenMapToMany(transformation.BackwardFromContext(imageDackBox.Bucket)).
					ThenMapEachToOne(transformation.StripPrefixUnchecked(imageDackBox.Bucket))

	// ComponentToCVETransformation uses a graph context to transform a component ID into a cve ID.
	ComponentToCVETransformation = transformation.AddPrefix(componentDackBox.Bucket).
					ThenMapToMany(transformation.ForwardFromContext(cveDackBox.Bucket)).
					ThenMapEachToOne(transformation.StripPrefixUnchecked(cveDackBox.Bucket))

	// CVEToComponentTransformation uses a graph context to transform a cve ID into a component ID.
	CVEToComponentTransformation = transformation.AddPrefix(cveDackBox.Bucket).
					ThenMapToMany(transformation.BackwardFromContext(componentDackBox.Bucket)).
					ThenMapEachToOne(transformation.StripPrefixUnchecked(componentDackBox.Bucket))

	// CVEToImageTransformation uses a graph context to transform a cve ID into an image ID.
	CVEToImageTransformation = transformation.AddPrefix(cveDackBox.Bucket).
					ThenMapToMany(transformation.BackwardFromContext(componentDackBox.Bucket)).
					ThenMapEachToMany(transformation.BackwardFromContext(imageDackBox.Bucket)).
					Then(transformation.Dedupe()).
					ThenMapEachToOne(transformation.StripPrefixUnchecked(imageDackBox.Bucket))

	// CVEToDeploymentTransformation uses a graph context to transform a cve ID into a deployment ID.
	CVEToDeploymentTransformation = transformation.AddPrefix(cveDackBox.Bucket).
					ThenMapToMany(transformation.BackwardFromContext(componentDackBox.Bucket)).
					ThenMapEachToMany(transformation.BackwardFromContext(imageDackBox.Bucket)).
					Then(transformation.Dedupe()).
					ThenMapEachToMany(transformation.BackwardFromContext(deploymentDackBox.Bucket)).
					ThenMapEachToOne(transformation.StripPrefixUnchecked(deploymentDackBox.Bucket)).
					Then(transformation.Dedupe())

	// ComponentCVEEdgeToCVETransformation transforms a component:cve edge ID into a cve ID.
	ComponentCVEEdgeToCVETransformation = transformation.Split([]byte{':'}).
						Then(transformation.AtIndex(1)).
						ThenMapEachToOne(transformation.Decode())

	// ComponentCVEEdgeToComponentTransformation transforms a component:cve edge ID into a component ID.
	ComponentCVEEdgeToComponentTransformation = transformation.Split([]byte{':'}).
							Then(transformation.AtIndex(0)).
							ThenMapEachToOne(transformation.Decode())

	// ClusterCVEEdgeToCVETransformation transforms a cluster:cve ID into a cve ID.
	ClusterCVEEdgeToCVETransformation = transformation.Split([]byte{':'}).
						Then(transformation.AtIndex(1)).
						ThenMapEachToOne(transformation.Decode())

	// ImageComponentEdgeToImageTransformation transforms a image:component ID into an image ID.
	ImageComponentEdgeToImageTransformation = transformation.Split([]byte{':'}).
						Then(transformation.AtIndex(0)).
						ThenMapEachToOne(transformation.Decode())

	// ImageComponentEdgeToComponentTransformation transforms a image:component ID into a component ID.
	ImageComponentEdgeToComponentTransformation = transformation.Split([]byte{':'}).
							Then(transformation.AtIndex(1)).
							ThenMapEachToOne(transformation.Decode())

	// ImageComponentEdgeToDeploymentTransformation transforms an image:component edge id into a set of deployment ids.
	ImageComponentEdgeToDeploymentTransformation = ImageComponentEdgeToImageTransformation.
							ThenMapEachToMany(ImageToDeploymentTransformation)

	// ImageComponentEdgeToCVETransformation transforms an image:component edge id into a set of cve ids.
	ImageComponentEdgeToCVETransformation = ImageComponentEdgeToComponentTransformation.
						ThenMapEachToMany(ComponentToCVETransformation)

	// ComponentCVEEdgeToDeploymentTransformation transforms a component:cve edge id into a deployment id.
	ComponentCVEEdgeToDeploymentTransformation = ComponentCVEEdgeToComponentTransformation.
							ThenMapEachToMany(ComponentToDeploymentTransformation)

	// ComponentCVEEdgeToImageTransformation transforms a component:cve edge id into an image id.
	ComponentCVEEdgeToImageTransformation = ComponentCVEEdgeToComponentTransformation.
						ThenMapEachToMany(ComponentToImageTransformation)
)
