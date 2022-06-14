package framework

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

// ComplianceDomain is the domain (i.e., the set of all potential target objects) for a compliance run.
type ComplianceDomain interface {
	ID() string
	Cluster() ComplianceTarget
	Nodes() []ComplianceTarget
	Deployments() []ComplianceTarget
	Pods() []*storage.Pod
	MachineConfigs() map[string][]ComplianceTarget
}

type complianceDomain struct {
	domainID       string
	cluster        clusterTarget
	nodes          []nodeTarget
	deployments    []deploymentTarget
	pods           []*storage.Pod
	machineConfigs map[string][]machineConfigTarget
}

func newComplianceDomain(cluster *storage.Cluster, nodes []*storage.Node, deployments []*storage.Deployment, pods []*storage.Pod, machineConfigs map[string][]string) *complianceDomain {
	clusterTarget := targetForCluster(cluster)
	nodeTargets := make([]nodeTarget, len(nodes))
	for i, node := range nodes {
		nodeTargets[i] = targetForNode(node)
	}
	deploymentTargets := make([]deploymentTarget, len(deployments))
	for i, deployment := range deployments {
		deploymentTargets[i] = targetForDeployment(deployment)
	}
	machineConfigTargets := make(map[string][]machineConfigTarget)
	for standard, scans := range machineConfigs {
		for _, scan := range scans {
			machineConfigTargets[standard] = append(machineConfigTargets[standard], targetForMachineConfig(scan))
		}
	}

	return &complianceDomain{
		domainID:       uuid.NewV4().String(),
		cluster:        clusterTarget,
		nodes:          nodeTargets,
		deployments:    deploymentTargets,
		pods:           pods,
		machineConfigs: machineConfigTargets,
	}
}

func (d *complianceDomain) ID() string {
	return d.domainID
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

func (d *complianceDomain) MachineConfigs() map[string][]ComplianceTarget {
	results := make(map[string][]ComplianceTarget)
	for k, v := range d.machineConfigs {
		for _, target := range v {
			results[k] = append(results[k], target)
		}
	}
	return results
}

func (d *complianceDomain) Pods() []*storage.Pod {
	result := make([]*storage.Pod, len(d.pods))
	copy(result, d.pods)
	return result
}

// NewComplianceDomain creates a new compliance domain from the given cluster, list of nodes and list of deployments.
func NewComplianceDomain(cluster *storage.Cluster, nodes []*storage.Node, deployments []*storage.Deployment, pods []*storage.Pod, machineConfigs map[string][]string) ComplianceDomain {
	return newComplianceDomain(cluster, nodes, deployments, pods, machineConfigs)
}
