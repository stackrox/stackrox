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

// AllTargets returns all targets in the given domain.
func AllTargets(domain ComplianceDomain) []ComplianceTarget {
	nodeTargets := domain.Nodes()
	deploymentTargets := domain.Deployments()
	result := make([]ComplianceTarget, 0, len(nodeTargets)+len(deploymentTargets)+1)
	result = append(result, domain.Cluster())
	result = append(result, nodeTargets...)
	result = append(result, deploymentTargets...)
	return result
}
