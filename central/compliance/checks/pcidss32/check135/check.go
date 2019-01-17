package check135

import (
	"strings"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
)

func init() {
	framework.MustRegisterNewCheck(
		"PCI_DSS_3_2:1_3_5",
		framework.DeploymentKind,
		nil,
		clusterIsCompliant)
}

func clusterIsCompliant(ctx framework.ComplianceContext) {
	framework.ForEachDeployment(ctx, func(ctx framework.ComplianceContext, deployment *storage.Deployment) {
		deploymentIsCompliant(ctx, deployment)
	})
}

func deploymentIsCompliant(ctx framework.ComplianceContext, deployment *storage.Deployment) {
	for _, container := range deployment.GetContainers() {
		for _, portConfig := range container.GetPorts() {
			if strings.ToLower(portConfig.GetProtocol()) == "udp" {
				framework.FailNowf(ctx, "Deployment %s (%s) uses UDP, which allows data exchange without an established connection", deployment.GetName(), deployment.GetId())
			}
		}
	}
	framework.Passf(ctx, "Deployment %s (%s) does not use UDP", deployment.GetName(), deployment.GetId())
}
