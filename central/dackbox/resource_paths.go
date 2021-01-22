package dackbox

import (
	clusterDackBox "github.com/stackrox/rox/central/cluster/dackbox"
	cveDackBox "github.com/stackrox/rox/central/cve/dackbox"
	deploymentDackBox "github.com/stackrox/rox/central/deployment/dackbox"
	imageDackBox "github.com/stackrox/rox/central/image/dackbox"
	componentDackBox "github.com/stackrox/rox/central/imagecomponent/dackbox"
	nsDackBox "github.com/stackrox/rox/central/namespace/dackbox"
	nodeDackBox "github.com/stackrox/rox/central/node/dackbox"
	"github.com/stackrox/rox/pkg/dackbox"
)

var (
	// ClusterToNamespace defines prefix path to go from clusters to namespaces
	ClusterToNamespace = dackbox.Path{
		Path: [][]byte{
			clusterDackBox.Bucket,
			nsDackBox.Bucket,
		},
		ForwardTraversal: true,
	}

	// ClusterToDeployment defines prefix path to go from clusters to deployments
	ClusterToDeployment = dackbox.Path{
		Path: [][]byte{
			clusterDackBox.Bucket,
			nsDackBox.Bucket,
			deploymentDackBox.Bucket,
		},
		ForwardTraversal: true,
	}

	// ClusterToCVE defines prefix path to go from clusters to cves
	ClusterToCVE = dackbox.Path{
		Path: [][]byte{
			clusterDackBox.Bucket,
			nsDackBox.Bucket,
			deploymentDackBox.Bucket,
			imageDackBox.Bucket,
			componentDackBox.Bucket,
			cveDackBox.Bucket,
		},
		ForwardTraversal: true,
	}

	// ClusterToClusterCVE defines prefix path to from cluster to cluster cves in dackbox
	ClusterToClusterCVE = dackbox.Path{
		Path: [][]byte{
			clusterDackBox.Bucket,
			cveDackBox.Bucket,
		},
		ForwardTraversal: true,
	}

	// NamespaceToDeploymentPath defines prefix path to go from namespaces to deployments in dackbox
	NamespaceToDeploymentPath = dackbox.Path{
		Path: [][]byte{
			nsDackBox.Bucket,
			deploymentDackBox.Bucket,
		},
		ForwardTraversal: true,
	}

	// NamespaceToImagePath defines prefix path to go from namespaces to images in dackbox
	NamespaceToImagePath = dackbox.Path{
		Path: [][]byte{
			nsDackBox.Bucket,
			deploymentDackBox.Bucket,
			imageDackBox.Bucket,
		},
		ForwardTraversal: true,
	}

	// NamespaceToCVEPath defines prefix path to go from namespaces to cves in dackbox
	NamespaceToCVEPath = dackbox.Path{
		Path: [][]byte{
			nsDackBox.Bucket,
			deploymentDackBox.Bucket,
			imageDackBox.Bucket,
			componentDackBox.Bucket,
			cveDackBox.Bucket,
		},
		ForwardTraversal: true,
	}

	// DeploymentToImage defines prefix path to go from deployments to images in dackbox
	DeploymentToImage = dackbox.Path{
		Path: [][]byte{
			deploymentDackBox.Bucket,
			imageDackBox.Bucket,
		},
		ForwardTraversal: true,
	}

	// DeploymentToImageComponent defines prefix path to from deployments to components in dackbox
	DeploymentToImageComponent = dackbox.Path{
		Path: [][]byte{
			deploymentDackBox.Bucket,
			imageDackBox.Bucket,
			componentDackBox.Bucket,
		},
		ForwardTraversal: true,
	}

	// DeploymentToCVE defines prefix path to from deployments to cves in dackbox
	DeploymentToCVE = dackbox.Path{
		Path: [][]byte{
			deploymentDackBox.Bucket,
			imageDackBox.Bucket,
			componentDackBox.Bucket,
			cveDackBox.Bucket,
		},
		ForwardTraversal: true,
	}

	// ImageToDeploymentPath defines prefix path to go from images to deployments in dackbox
	ImageToDeploymentPath = dackbox.Path{
		Path: [][]byte{
			imageDackBox.Bucket,
			deploymentDackBox.Bucket,
		},
		ForwardTraversal: false,
	}

	// ImageToComponentPath defines prefix path to go from images to components in dackbox
	ImageToComponentPath = dackbox.Path{
		Path: [][]byte{
			imageDackBox.Bucket,
			componentDackBox.Bucket,
		},
		ForwardTraversal: true,
	}

	// ImageToCVEPath defines prefix path to go from images to cves in dackbox
	ImageToCVEPath = dackbox.Path{
		Path: [][]byte{
			imageDackBox.Bucket,
			componentDackBox.Bucket,
			cveDackBox.Bucket,
		},
		ForwardTraversal: true,
	}

	// ComponentToDeploymentPath defines prefix path to traverse from components to deployments in dackbox
	ComponentToDeploymentPath = dackbox.Path{
		Path: [][]byte{
			componentDackBox.Bucket,
			imageDackBox.Bucket,
			deploymentDackBox.Bucket,
		},
		ForwardTraversal: false,
	}

	// ComponentToImagePath defines prefix path to traverse from components to images in dackbox
	ComponentToImagePath = dackbox.Path{
		Path: [][]byte{
			componentDackBox.Bucket,
			imageDackBox.Bucket,
		},
		ForwardTraversal: false,
	}

	// ComponentToCVEPath defines prefix path to go from components to cves in dackbox
	ComponentToCVEPath = dackbox.Path{
		Path: [][]byte{
			componentDackBox.Bucket,
			cveDackBox.Bucket,
		},
		ForwardTraversal: true,
	}

	// CVEToComponentPath defines prefix path to go from cves to components in dackbox
	CVEToComponentPath = dackbox.Path{
		Path: [][]byte{
			cveDackBox.Bucket,
			componentDackBox.Bucket,
		},
		ForwardTraversal: false,
	}

	// CVEToNodePath defines prefix path to go from cves to nodes in dackbox
	CVEToNodePath = dackbox.Path{
		Path: [][]byte{
			cveDackBox.Bucket,
			componentDackBox.Bucket,
			nodeDackBox.Bucket,
		},
		ForwardTraversal: false,
	}

	// CVEToImagePath defines prefix path to go from cves to images in dackbox
	CVEToImagePath = dackbox.Path{
		Path: [][]byte{
			cveDackBox.Bucket,
			componentDackBox.Bucket,
			imageDackBox.Bucket,
		},
		ForwardTraversal: false,
	}

	// CVEToDeploymentPath defines prefix path to go from cves to deployments in dackbox
	CVEToDeploymentPath = dackbox.Path{
		Path: [][]byte{
			cveDackBox.Bucket,
			componentDackBox.Bucket,
			imageDackBox.Bucket,
			deploymentDackBox.Bucket,
		},
		ForwardTraversal: false,
	}
)
