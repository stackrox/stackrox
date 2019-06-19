package runtime

import (
	"context"

	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection"
)

type whitelistTestingExecutor interface {
	detection.PolicyExecutor

	GetResult() bool
}

func newWhitelistTestingExecutor(executorCtx context.Context, deployments datastore.DataStore, deploymentID string) whitelistTestingExecutor {
	return &whitelistTestingExecutorImpl{
		executorCtx:  executorCtx,
		deploymentID: deploymentID,
		deployments:  deployments,
	}
}

type whitelistTestingExecutorImpl struct {
	executorCtx  context.Context
	deploymentID string
	deployments  datastore.DataStore
	result       bool
}

func (wte *whitelistTestingExecutorImpl) GetResult() bool {
	return wte.result
}

func (wte *whitelistTestingExecutorImpl) Execute(compiled detection.CompiledPolicy) error {
	if compiled.Policy().GetDisabled() {
		wte.result = true
		return nil
	}
	dep, exists, err := wte.deployments.GetDeployment(wte.executorCtx, wte.deploymentID)
	if err != nil {
		return err
	}
	if !exists {
		// Assume it's not whitelisted if it doesn't exist, otherwise runtime alerts for deleted deployments
		// will always get removed every time we update a policy.
		wte.result = false
		return nil
	}
	wte.result = !compiled.AppliesTo(dep)
	return nil
}
