package framework

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
)

// Abort aborts the current compliance check, optionally setting an error.
func Abort(_ ComplianceContext, err error) {
	halt(err)
}

// RunForTarget runs the given check function for the given compliance target.
func RunForTarget(ctx ComplianceContext, target ComplianceTarget, check CheckFunc) {
	childCtx := ctx.ForObject(target)
	doRun(childCtx, check)
}

func forEachNode(ctx ComplianceContext, check CheckFunc) {
	for _, node := range ctx.Domain().Nodes() {
		RunForTarget(ctx, node, check)
	}
}

// ForEachNode runs the given node-scoped check function for every node in the compliance domain.
func ForEachNode(ctx ComplianceContext, checkFn func(ComplianceContext, *storage.Node)) {
	forEachNode(ctx, func(ctx ComplianceContext) {
		node := ctx.Target().Node()
		checkFn(ctx, node)
	})
}

func forEachDeployment(ctx ComplianceContext, check CheckFunc) {
	for _, deployment := range ctx.Domain().Deployments() {
		RunForTarget(ctx, deployment, check)
	}
}

// ForEachDeployment runs the given deployment-scoped check function for every deployment in the compliance domain.
func ForEachDeployment(ctx ComplianceContext, checkFn func(ComplianceContext, *storage.Deployment)) {
	forEachDeployment(ctx, func(ctx ComplianceContext) {
		deployment := ctx.Target().Deployment()
		checkFn(ctx, deployment)
	})
}

func forEachMachineConfig(ctx ComplianceContext, check CheckFunc) {
	for _, mc := range ctx.Domain().MachineConfigs()[ctx.StandardName()] {
		RunForTarget(ctx, mc, check)
	}
}

// ForEachMachineConfig runs the given machineconfig-scoped check function for every machine config in the compliance domain.
func ForEachMachineConfig(ctx ComplianceContext, checkFn func(ComplianceContext, string)) {
	forEachMachineConfig(ctx, func(ctx ComplianceContext) {
		mc := ctx.Target().MachineConfig()
		checkFn(ctx, mc)
	})
}

// finalize catches any panic that occurred when running a compliance check, and propagates it, if needed.
func finalize(ctx ComplianceContext, panicked *bool) {
	var err error
	if action := recover(); action != nil || *panicked {
		log.Debugf("finalize: %+v", action)

		halted := false
		switch a := action.(type) {
		case haltSignal:
			err = a.err
			halted = true
		case error:
			err = a
		default:
			err = fmt.Errorf("caught panic: %+v", a)
		}

		if !halted {
			utils.Should(err)
		}
	}

	ctx.Finalize(err)
}

// doRun runs a compliance check, handling any panics that might arise.
func doRun(ctx ComplianceContext, check CheckFunc) {
	panicked := true
	defer finalize(ctx, &panicked)
	check(ctx)
	panicked = false
}
