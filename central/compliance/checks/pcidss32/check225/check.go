package check225

import (
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
	pkgFramework "github.com/stackrox/rox/pkg/compliance/framework"
	"github.com/stackrox/rox/pkg/set"
)

const checkID = "PCI_DSS_3_2:2_2_5"

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 checkID,
			Scope:              pkgFramework.DeploymentKind,
			DataDependencies:   []string{"NetworkFlows"},
			InterpretationText: interpretationText,
		},
		clusterIsCompliant)
}

func clusterIsCompliant(ctx framework.ComplianceContext) {
	// Map deployments to flows where it is the destination.
	deploymentIDToIncomingFlows := make(map[string][]*storage.NetworkFlow)
	for _, flow := range ctx.Data().NetworkFlowsWithDeploymentDst() {
		dst := flow.GetProps().GetDstEntity()
		deploymentIDToIncomingFlows[dst.GetId()] = append(deploymentIDToIncomingFlows[dst.GetId()], flow)
	}
	// Map enabled ports
	framework.ForEachDeployment(ctx, func(ctx framework.ComplianceContext, deployment *storage.Deployment) {
		deploymentIsCompliant(ctx, deployment, deploymentIDToIncomingFlows[deployment.GetId()])
	})
}

func deploymentIsCompliant(ctx framework.ComplianceContext, deployment *storage.Deployment, incomingFlows []*storage.NetworkFlow) {
	// Check that all exposed ports have incoming traffic. Note this is at a deployment level, so if all containers have
	// port 80 exposed for instance, and only one is receiving traffic, it will still pass.
	exposedAndUnused := portsExposedAndNotUsed(deployment, incomingFlows)

	// If we have exposed ports no one is sending traffic to, that's also a fail.
	if exposedAndUnused.Cardinality() > 0 {
		framework.Failf(ctx, failText(exposedAndUnused.AsSlice()))
	} else {
		framework.Passf(ctx, passText())
	}
}

// Note this is at a deployment level, so if all containers have port 80 exposed for instance, and only one is receiving
// traffic, it will not be returned in the exposedAndNotUsed set.
func portsExposedAndNotUsed(deployment *storage.Deployment, incomingFlows []*storage.NetworkFlow) set.Set[uint32] {
	// Get the ports of all flows seen.
	seenPorts := set.NewSet[uint32]()
	for _, flow := range incomingFlows {
		seenPorts.Add(flow.GetProps().GetDstPort())
	}

	// Get all of the ports exposed in the deployment
	exposedPorts := set.NewSet[uint32]()
	for _, portConfig := range deployment.GetPorts() {
		exposedPorts.Add(uint32(portConfig.GetContainerPort()))
	}
	return exposedPorts.Difference(seenPorts)
}
