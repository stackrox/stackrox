package framework

import (
	"fmt"
	"runtime/debug"

	"github.com/stackrox/rox/generated/storage"
)

// Abort aborts the current compliance check, optionally setting an error.
func Abort(ctx ComplianceContext, err error) {
	panic(err)
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

// ForEachNodeCheck takes a CheckFunc operating on the `Node` scope and converts it into a check function operating
// on the `Cluster` scope.
func ForEachNodeCheck(check CheckFunc) CheckFunc {
	return func(ctx ComplianceContext) {
		forEachNode(ctx, check)
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

// ForEachDeploymentCheck takes a CheckFunc operating on the `Deployment` scope and converts it into a CheckFunc
// operating on the `Cluster` scope.
func ForEachDeploymentCheck(check CheckFunc) CheckFunc {
	return func(ctx ComplianceContext) {
		forEachDeployment(ctx, check)
	}
}

// ForEachDeployment runs the given deployment-scoped check function for every deployment in the compliance domain.
func ForEachDeployment(ctx ComplianceContext, checkFn func(ComplianceContext, *storage.Deployment)) {
	forEachDeployment(ctx, func(ctx ComplianceContext) {
		deployment := ctx.Target().Deployment()
		checkFn(ctx, deployment)
	})
}

// finalize catches any panic that occurred when running a compliance check, and propagates it, if needed.
func finalize(ctx ComplianceContext) {
	var err error
	if action := recover(); action != nil {
		log.Infof("finalize: %+v", action)
		debug.PrintStack()
		switch a := action.(type) {
		case error:
			err = a
		default:
			err = fmt.Errorf("caught panic: %+v", a)
		}
	}

	ctx.Finalize(err)
}

// doRun runs a compliance check, handling any panics that might arise.
func doRun(ctx ComplianceContext, check CheckFunc) {
	defer finalize(ctx)
	check(ctx)
}
