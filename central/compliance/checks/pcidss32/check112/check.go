package check112

import (
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

func init() {
	framework.MustRegisterNewCheck(
		"PCI_DSS_3_2:1_1_2",
		framework.DeploymentKind,
		[]string{"NetworkGraph"},
		checkAllDeploymentsInNetworkGraph)
}

func checkAllDeploymentsInNetworkGraph(ctx framework.ComplianceContext) {
	networkGraph := ctx.Data().NetworkGraph()

	deploymentNodes := set.NewStringSet()
	for _, node := range networkGraph.GetNodes() {
		if node.GetEntity().GetType() != storage.NetworkEntityInfo_DEPLOYMENT {
			continue
		}

		deploymentNodes.Add(node.GetEntity().GetId())
	}

	framework.ForEachDeployment(ctx, func(ctx framework.ComplianceContext, deployment *storage.Deployment) {
		if !deploymentNodes.Contains(deployment.GetId()) {
			framework.FailNowf(ctx, "Deployment %s (%s) is not present in network graph", deployment.GetName(), deployment.GetId())
		}
		framework.PassNowf(ctx, "Deployment %s (%s) is present in network graph", deployment.GetName(), deployment.GetId())
	})
}
