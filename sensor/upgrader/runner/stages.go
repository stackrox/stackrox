package runner

import (
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/k8sutil/k8sobjects"
	"github.com/stackrox/stackrox/pkg/sensorupgrader"
	"github.com/stackrox/stackrox/sensor/upgrader/bundle"
	"github.com/stackrox/stackrox/sensor/upgrader/cleanup"
	"github.com/stackrox/stackrox/sensor/upgrader/execution"
	"github.com/stackrox/stackrox/sensor/upgrader/plan"
	"github.com/stackrox/stackrox/sensor/upgrader/preflight"
	"github.com/stackrox/stackrox/sensor/upgrader/snapshot"
)

type stage struct {
	description string
	run         func() error
}

func (r *runner) Stages() map[sensorupgrader.Stage]stage {
	return map[sensorupgrader.Stage]stage{
		sensorupgrader.CleanupForeignStateStage: {
			description: "Clean up state left over by other upgrade processes",
			run:         r.cleanupForeignState,
		},
		sensorupgrader.SnapshotForRollForwardStage: {
			description: "Take or read state snapshot",
			run:         r.snapshotForRollForward,
		},
		sensorupgrader.SnapshotForRollbackStage: {
			description: "Read existing state snapshot",
			run:         r.snapshotForRollback,
		},
		sensorupgrader.SnapshotForDryRunStage: {
			description: "Take or read state snapshot (do not store)",
			run:         r.snapshotForDryRun,
		},
		sensorupgrader.FetchBundleStage: {
			description: "Fetch sensor bundle",
			run:         r.fetchBundle,
		},
		sensorupgrader.InstantiateBundleStage: {
			description: "Instantiate objects from sensor bundle",
			run:         r.instantiateBundle,
		},
		sensorupgrader.GeneratePlanStage: {
			description: "Generate execution plan",
			run:         r.generatePlan,
		},
		sensorupgrader.GenerateRollbackPlanStage: {
			description: "Generate rollback execution plan",
			run:         r.generateRollbackPlan,
		},
		sensorupgrader.PreflightStage: {
			description: "Run preflight checks",
			run:         r.preflightChecks,
		},
		sensorupgrader.PreflightNoFailStage: {
			description: "Run preflight checks (informative only)",
			run:         r.preflightChecksNoFail,
		},
		sensorupgrader.ExecuteStage: {
			description: "Execute plan",
			run:         r.executePlan,
		},
		sensorupgrader.CleanupOwnerStage: {
			description: "Clean up owning deployment",
			run:         r.cleanupOwner,
		},
		sensorupgrader.WaitForDeletionStage: {
			description: "Wait for deletion to take effect",
			run:         r.waitForDeletion,
		},
	}
}

func (r *runner) snapshotForRollForward() error {
	preUpgradeObjs, err := snapshot.TakeOrReadSnapshot(r.ctx, snapshot.Options{Store: true})
	if err != nil {
		return err
	}
	r.preUpgradeObjs = preUpgradeObjs
	r.preUpgradeState = k8sobjects.BuildObjectMap(preUpgradeObjs)
	return nil
}

func (r *runner) snapshotForDryRun() error {
	preUpgradeObjs, err := snapshot.TakeOrReadSnapshot(r.ctx, snapshot.Options{})
	if err != nil {
		return err
	}
	r.preUpgradeObjs = preUpgradeObjs
	r.preUpgradeState = k8sobjects.BuildObjectMap(preUpgradeObjs)
	return nil
}

func (r *runner) snapshotForRollback() error {
	preUpgradeObjs, err := snapshot.TakeOrReadSnapshot(r.ctx, snapshot.Options{MustExist: true})
	if err != nil {
		return err
	}
	r.preUpgradeObjs = preUpgradeObjs
	r.preUpgradeState = k8sobjects.BuildObjectMap(preUpgradeObjs)
	return nil
}

func (r *runner) fetchBundle() error {
	var err error
	if *localBundle != "" {
		r.bundleContents, err = bundle.LoadBundle(*localBundle)
	} else {
		r.bundleContents, err = bundle.FetchBundle(r.ctx)
	}
	if err != nil {
		return err
	}
	return nil
}

func (r *runner) instantiateBundle() error {
	postUpgradeObjs, err := bundle.InstantiateBundle(r.ctx, r.bundleContents)
	if err != nil {
		return err
	}
	transferMetadata(postUpgradeObjs, r.preUpgradeState)
	r.postUpgradeObjs = postUpgradeObjs
	return nil
}

func (r *runner) generatePlan() error {
	executionPlan, err := plan.GenerateExecutionPlan(r.ctx, r.postUpgradeObjs, false)
	if err != nil {
		return err
	}
	r.executionPlan = executionPlan
	return nil
}

func (r *runner) generateRollbackPlan() error {
	executionPlan, err := plan.GenerateExecutionPlan(r.ctx, r.preUpgradeObjs, true)
	if err != nil {
		return err
	}
	r.executionPlan = executionPlan
	return nil
}

func (r *runner) preflightChecks() error {
	if err := preflight.PerformChecks(r.ctx, r.executionPlan); err != nil {
		return err
	}
	return nil
}

func (r *runner) preflightChecksNoFail() error {
	if err := preflight.PerformChecks(r.ctx, r.executionPlan); err != nil {
		log.Error("Attempting to continue despite errors in preflight checks")
	}
	return nil
}

func (r *runner) executePlan() error {
	if err := execution.ExecutePlan(r.ctx, r.executionPlan); err != nil {
		return err
	}
	return nil
}

func (r *runner) cleanupForeignState() error {
	if err := cleanup.ForeignState(r.ctx); err != nil {
		return err
	}
	return nil
}

func (r *runner) cleanupOwner() error {
	if err := cleanup.Owner(r.ctx); err != nil {
		return err
	}
	return nil
}

func (r *runner) waitForDeletion() error {
	const deletionMaxGracePeriod = 30 * time.Second

	if concurrency.WaitWithTimeout(r.ctx.Context(), deletionMaxGracePeriod) {
		return errors.Wrap(r.ctx.Context().Err(), "context error waiting for deletion")
	}
	return errors.Errorf("still alive %v after supposed deletion, this doesn't seem right", deletionMaxGracePeriod)
}
