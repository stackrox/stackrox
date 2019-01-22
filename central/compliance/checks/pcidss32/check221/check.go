package check221

import (
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
)

var log = logging.LoggerForModule()

const checkID = "PCI_DSS_3_2:2_2_1"

func init() {
	framework.MustRegisterNewCheck(
		checkID,
		framework.DeploymentKind,
		[]string{"ProcessIndicators"},
		checkClusterIsCompliant)
}

// Cluster is compliant if all deployments are compliant.
func checkClusterIsCompliant(ctx framework.ComplianceContext) {
	// Map process indicators to deployments.
	deploymentIDToIndicators := make(map[string][]*storage.ProcessIndicator)
	for _, indicator := range ctx.Data().ProcessIndicators() {
		indicators := deploymentIDToIndicators[indicator.GetDeploymentId()]
		indicators = append(indicators, indicator)
		deploymentIDToIndicators[indicator.GetDeploymentId()] = indicators
	}

	// Check every deployment.
	framework.ForEachDeployment(ctx, func(ctx framework.ComplianceContext, deployment *storage.Deployment) {
		checkDeploymentIsCompliant(ctx, deployment, deploymentIDToIndicators[deployment.GetId()])
	})
}

// Deployment is compliant if all containers are compliant.
func checkDeploymentIsCompliant(ctx framework.ComplianceContext, deployment *storage.Deployment, indicators []*storage.ProcessIndicator) {
	// Map process indicators to containers.
	containerNameToIndicators := make(map[string][]*storage.ProcessIndicator)
	for _, indicator := range indicators {
		indicators := containerNameToIndicators[indicator.GetContainerName()]
		indicators = append(indicators, indicator)
		containerNameToIndicators[indicator.GetContainerName()] = indicators
	}

	// Check that every container is running a single binary.
	var failedContainers uint32
	for _, container := range deployment.GetContainers() {
		if countProcesses(container, containerNameToIndicators[container.GetName()]) > 1 {
			framework.Failf(ctx, failText(container))
			failedContainers = failedContainers + 1
		}
	}

	// If no containers failed, then the deployment passes.
	if failedContainers == 0 {
		framework.Pass(ctx, passText())
	}
}

// A server should have a single executable running (possible many process IDs, but the binary should be the same).
func countProcesses(container *storage.Container, indicators []*storage.ProcessIndicator) int {
	if len(indicators) == 0 {
		log.Errorf("found a container (Name: %s ID: %s) with no processes", container.GetName(), container.GetId())
		return 0
	}

	// We want to dedupe by process name, so that worker processes of the same binary aren't considered separately.
	processes := set.NewStringSet()
	for _, indicator := range indicators {
		processes.Add(indicator.GetSignal().GetName())
	}
	return processes.Cardinality()
}
