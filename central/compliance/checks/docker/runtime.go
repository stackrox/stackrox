package docker

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
	pkgFramework "github.com/stackrox/rox/pkg/compliance/framework"
)

func init() {
	framework.MustRegisterChecks(
		sshCheck(),
	)
}

func sshCheck() framework.Check {
	md := framework.CheckMetadata{
		ID:                 "CIS_Docker_v1_2_0:5_6",
		Scope:              pkgFramework.DeploymentKind,
		InterpretationText: "StackRox checks that every running container in each deployment does not have ssh process running",
		DataDependencies:   []string{"ProcessIndicators"},
	}
	checkFunc := func(ctx framework.ComplianceContext) {
		framework.ForEachDeployment(ctx, ssh)
	}
	return framework.NewCheckFromFunc(md, checkFunc)
}

func ssh(ctx framework.ComplianceContext, deployment *storage.Deployment) {
	var fail bool
	runningContainerIDs := getRunningContainerIDs(deployment, ctx.Domain().Pods())
	if len(runningContainerIDs) == 0 {
		framework.Passf(ctx, "Deployment %s has no running containers", deployment.GetName())
		return
	}
	for runningContainerID, containerName := range runningContainerIDs {
		for _, indicator := range ctx.Data().SSHProcessIndicators() {
			// indicator.GetSignal().GetContainerId() only returns the first 12 characters of the container ID.
			if strings.HasPrefix(runningContainerID, indicator.GetSignal().GetContainerId()) {
				fail = true
				processWithArgs := fmt.Sprintf("%s %s", indicator.GetSignal().GetExecFilePath(), indicator.GetSignal().GetArgs())
				framework.Failf(ctx, "Container %q has ssh process running: %q", containerName, processWithArgs)
			}
		}
		if !fail {
			framework.Passf(ctx, "Container %q has no ssh process running", containerName)
		}
	}
}

func getRunningContainerIDs(deployment *storage.Deployment, pods []*storage.Pod) map[string]string {
	runningContainerIDs := make(map[string]string)
	for _, pod := range pods {
		if pod.GetDeploymentId() != deployment.GetId() {
			continue
		}
		for _, runningInstance := range pod.GetLiveInstances() {
			runningContainerIDs[runningInstance.GetInstanceId().GetId()] = fmt.Sprintf("%s:%s", pod.GetName(), runningInstance.GetInstanceId().GetId())
		}
	}

	return runningContainerIDs
}
