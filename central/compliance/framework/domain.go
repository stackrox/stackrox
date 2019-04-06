package framework

import "github.com/stackrox/rox/generated/storage"

// ComplianceDomain is the domain (i.e., the set of all potential target objects) for a compliance run.
type ComplianceDomain interface {
	Cluster() ComplianceTarget
	Nodes() []ComplianceTarget
	Deployments() []ComplianceTarget
}

type complianceDomain struct {
	cluster     clusterTarget
	nodes       []nodeTarget
	deployments []deploymentTarget
}

func newComplianceDomain(cluster *storage.Cluster, nodes []*storage.Node, deployments []*storage.Deployment) *complianceDomain {
	clusterTarget := targetForCluster(cluster)
	nodeTargets := make([]nodeTarget, len(nodes))
	for i, node := range nodes {
		nodeTargets[i] = targetForNode(node)
	}
	deploymentTargets := make([]deploymentTarget, len(deployments))
	for i, deployment := range deployments {
		deploymentTargets[i] = targetForDeployment(deployment)
	}
	return &complianceDomain{
		cluster:     clusterTarget,
		nodes:       nodeTargets,
		deployments: deploymentTargets,
	}
}

func (d *complianceDomain) Cluster() ComplianceTarget {
	return d.cluster
}

func (d *complianceDomain) Nodes() []ComplianceTarget {
	result := make([]ComplianceTarget, len(d.nodes))
	for i, node := range d.nodes {
		result[i] = node
	}
	return result
}

func (d *complianceDomain) Deployments() []ComplianceTarget {
	result := make([]ComplianceTarget, len(d.deployments))
	for i, deployment := range d.deployments {
		result[i] = deployment
	}
	return result
}

// NewComplianceDomain creates a new compliance domain from the given cluster, list of nodes and list of deployments.
func NewComplianceDomain(cluster *storage.Cluster, nodes []*storage.Node, deployments []*storage.Deployment) ComplianceDomain {
	return newComplianceDomain(cluster, nodes, deployments)
}
