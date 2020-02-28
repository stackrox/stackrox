package dackbox

import (
	clusterDackBox "github.com/stackrox/rox/central/cluster/dackbox"
	cveDackBox "github.com/stackrox/rox/central/cve/dackbox"
	deploymentDackBox "github.com/stackrox/rox/central/deployment/dackbox"
	imageDackBox "github.com/stackrox/rox/central/image/dackbox"
	componentDackBox "github.com/stackrox/rox/central/imagecomponent/dackbox"
	nsDackBox "github.com/stackrox/rox/central/namespace/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/keys/transformation"
)

var (
	// ClusterToCVETransformation uses a graph context to transform a cluster ID into a cve ID.
	ClusterToCVETransformation = transformation.AddPrefix(clusterDackBox.Bucket).
					ThenMapToMany(transformation.ForwardFromContext()).
					Then(transformation.HasPrefix(nsDackBox.Bucket)).
					ThenMapEachToMany(transformation.ForwardFromContext()).
					Then(transformation.HasPrefix(deploymentDackBox.Bucket)).
					ThenMapEachToMany(transformation.ForwardFromContext()).
					Then(transformation.HasPrefix(imageDackBox.Bucket)).
					Then(transformation.Dedupe()).
					ThenMapEachToMany(transformation.ForwardFromContext()).
					Then(transformation.HasPrefix(componentDackBox.Bucket)).
					Then(transformation.Dedupe()).
					ThenMapEachToMany(transformation.ForwardFromContext()).
					Then(transformation.HasPrefix(cveDackBox.Bucket)).
					Then(transformation.Dedupe()).
					ThenMapEachToOne(transformation.StripPrefix(cveDackBox.Bucket))

	// ClusterToClusterCVETransformation uses a graph context to transform a cluster ID into a cluster cve ID.
	ClusterToClusterCVETransformation = transformation.AddPrefix(clusterDackBox.Bucket).
						ThenMapToMany(transformation.ForwardFromContext()).
						Then(transformation.HasPrefix(cveDackBox.Bucket)).
						ThenMapEachToOne(transformation.StripPrefix(cveDackBox.Bucket))

	// DeploymentToImageComponentTransformation uses a graph context to transform a deployment ID into a component ID.
	DeploymentToImageComponentTransformation = transformation.AddPrefix(deploymentDackBox.Bucket).
							ThenMapToMany(transformation.ForwardFromContext()).
							Then(transformation.HasPrefix(imageDackBox.Bucket)).
							ThenMapEachToMany(transformation.ForwardFromContext()).
							Then(transformation.HasPrefix(componentDackBox.Bucket)).
							Then(transformation.Dedupe()).
							ThenMapEachToOne(transformation.StripPrefix(componentDackBox.Bucket))

	// DeploymentToCVETransformation uses a graph context to transform a deployment ID into a cve ID.
	DeploymentToCVETransformation = transformation.AddPrefix(deploymentDackBox.Bucket).
					ThenMapToMany(transformation.ForwardFromContext()).
					Then(transformation.HasPrefix(imageDackBox.Bucket)).
					ThenMapEachToMany(transformation.ForwardFromContext()).
					Then(transformation.HasPrefix(componentDackBox.Bucket)).
					Then(transformation.Dedupe()).
					ThenMapEachToMany(transformation.ForwardFromContext()).
					Then(transformation.HasPrefix(cveDackBox.Bucket)).
					Then(transformation.Dedupe()).
					ThenMapEachToOne(transformation.StripPrefix(cveDackBox.Bucket))

	// ImageToDeploymentTransformation uses a graph context to transform a image ID into a Deployment ID.
	ImageToDeploymentTransformation = transformation.AddPrefix(imageDackBox.Bucket).
					ThenMapToMany(transformation.BackwardFromContext()).
					Then(transformation.HasPrefix(deploymentDackBox.Bucket)).
					ThenMapEachToOne(transformation.StripPrefix(deploymentDackBox.Bucket))

	// ImageToImageComponentTransformation trasforms an image id to a set of image component ids.
	ImageToImageComponentTransformation = transformation.AddPrefix(imageDackBox.Bucket).
						ThenMapToMany(transformation.ForwardFromContext()).
						Then(transformation.HasPrefix(componentDackBox.Bucket)).
						ThenMapEachToOne(transformation.StripPrefix(componentDackBox.Bucket))

	// ImageToCVETransformation uses a graph context to transform a image ID into a cve ID.
	ImageToCVETransformation = transformation.AddPrefix(imageDackBox.Bucket).
					ThenMapToMany(transformation.ForwardFromContext()).
					Then(transformation.HasPrefix(componentDackBox.Bucket)).
					ThenMapEachToMany(transformation.ForwardFromContext()).
					Then(transformation.HasPrefix(cveDackBox.Bucket)).
					Then(transformation.Dedupe()).
					ThenMapEachToOne(transformation.StripPrefix(cveDackBox.Bucket))

	// ComponentToDeploymentTransformation uses a graph context to transform a component ID into a Deployment ID.
	ComponentToDeploymentTransformation = transformation.AddPrefix(componentDackBox.Bucket).
						ThenMapToMany(transformation.BackwardFromContext()).
						Then(transformation.HasPrefix(imageDackBox.Bucket)).
						ThenMapEachToMany(transformation.BackwardFromContext()).
						Then(transformation.HasPrefix(deploymentDackBox.Bucket)).
						Then(transformation.Dedupe()).
						ThenMapEachToOne(transformation.StripPrefix(deploymentDackBox.Bucket))

	// ComponentToImageTransformation uses a graph context to transform a component ID into an image ID.
	ComponentToImageTransformation = transformation.AddPrefix(componentDackBox.Bucket).
					ThenMapToMany(transformation.BackwardFromContext()).
					Then(transformation.HasPrefix(imageDackBox.Bucket)).
					ThenMapEachToOne(transformation.StripPrefix(imageDackBox.Bucket))

	// ComponentToCVETransformation uses a graph context to transform a component ID into a cve ID.
	ComponentToCVETransformation = transformation.AddPrefix(componentDackBox.Bucket).
					ThenMapToMany(transformation.ForwardFromContext()).
					Then(transformation.HasPrefix(cveDackBox.Bucket)).
					ThenMapEachToOne(transformation.StripPrefix(cveDackBox.Bucket))

	// CVEToComponentTransformation uses a graph context to transform a cve ID into a component ID.
	CVEToComponentTransformation = transformation.AddPrefix(cveDackBox.Bucket).
					ThenMapToMany(transformation.BackwardFromContext()).
					Then(transformation.HasPrefix(componentDackBox.Bucket)).
					ThenMapEachToOne(transformation.StripPrefix(componentDackBox.Bucket))

	// CVEToImageTransformation uses a graph context to transform a cve ID into an image ID.
	CVEToImageTransformation = transformation.AddPrefix(cveDackBox.Bucket).
					ThenMapToMany(transformation.BackwardFromContext()).
					Then(transformation.HasPrefix(componentDackBox.Bucket)).
					ThenMapEachToMany(transformation.BackwardFromContext()).
					Then(transformation.HasPrefix(imageDackBox.Bucket)).
					Then(transformation.Dedupe()).
					ThenMapEachToOne(transformation.StripPrefix(imageDackBox.Bucket))

	// CVEToDeploymentTransformation uses a graph context to transform a cve ID into a deployment ID.
	CVEToDeploymentTransformation = transformation.AddPrefix(cveDackBox.Bucket).
					ThenMapToMany(transformation.BackwardFromContext()).
					Then(transformation.HasPrefix(componentDackBox.Bucket)).
					ThenMapEachToMany(transformation.BackwardFromContext()).
					Then(transformation.HasPrefix(imageDackBox.Bucket)).
					Then(transformation.Dedupe()).
					ThenMapEachToMany(transformation.BackwardFromContext()).
					Then(transformation.HasPrefix(deploymentDackBox.Bucket)).
					Then(transformation.Dedupe()).
					ThenMapEachToOne(transformation.StripPrefix(deploymentDackBox.Bucket))

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
