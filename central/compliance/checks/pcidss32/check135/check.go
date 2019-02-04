package check135

import (
	"strings"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
)

const checkID = "PCI_DSS_3_2:1_3_5"

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 checkID,
			Scope:              framework.DeploymentKind,
			InterpretationText: interpretationText,
		},
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
				framework.Fail(ctx, failText())
				return
			}
		}
	}
	framework.Pass(ctx, passText())
}
