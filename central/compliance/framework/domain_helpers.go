package framework

import "github.com/stackrox/rox/generated/storage"

// Nodes returns a slice of all node objects in the domain.
func Nodes(domain ComplianceDomain) []*storage.Node {
	nodeTargets := domain.Nodes()
	nodes := make([]*storage.Node, len(nodeTargets))
	for i, nodeTarget := range nodeTargets {
		nodes[i] = nodeTarget.Node()
	}
	return nodes
}

// Deployments returns a slice of all deployment objects in the domain.
func Deployments(domain ComplianceDomain) []*storage.Deployment {
	deploymentTargets := domain.Deployments()
	deployments := make([]*storage.Deployment, len(deploymentTargets))
	for i, deploymentTarget := range deploymentTargets {
		deployments[i] = deploymentTarget.Deployment()
	}
	return deployments
}
