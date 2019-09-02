package runner

import (
	"github.com/stackrox/rox/sensor/upgrader/bundle"
	"github.com/stackrox/rox/sensor/upgrader/execution"
	"github.com/stackrox/rox/sensor/upgrader/k8sobjects"
	"github.com/stackrox/rox/sensor/upgrader/plan"
	"github.com/stackrox/rox/sensor/upgrader/preflight"
	"github.com/stackrox/rox/sensor/upgrader/snapshot"
)

type stage struct {
	name string
	run  func() error
}

func (r *runner) Workflows() map[string][]string {
	return map[string][]string{
		"roll-forward": {
			"snapshot",
			"fetch-bundle",
			"instantiate-bundle",
			"generate-plan",
			"preflight",
			"execute",
		},
		"roll-back": {
			"snapshot-nostore",
			"generate-rollback-plan",
			"preflight-nofail",
			"execute",
		},
		"dry-run": {
			"snapshot-nostore",
			"fetch-bundle",
			"instantiate-bundle",
			"generate-plan",
			"preflight",
		},
		"validate-bundle": {
			"fetch-bundle",
			"instantiate-bundle",
		},
	}
}

func (r *runner) Stages() map[string]stage {
	return map[string]stage{
		"snapshot": {
			name: "Take or read state snapshot",
			run:  r.takeOrReadSnapshot,
		},
		"snapshot-nostore": {
			name: "Take or read state snapshot (do not store)",
			run:  r.takeOrReadSnapshotNoStore,
		},
		"fetch-bundle": {
			name: "Fetch sensor bundle",
			run:  r.fetchBundle,
		},
		"instantiate-bundle": {
			name: "Instantiate objects from sensor bundle",
			run:  r.instantiateBundle,
		},
		"generate-plan": {
			name: "Generate execution plan",
			run:  r.generatePlan,
		},
		"generate-rollback-plan": {
			name: "Generate rollback execution plan",
			run:  r.generateRollbackPlan,
		},
		"preflight": {
			name: "Run preflight checks",
			run:  r.preflightChecks,
		},
		"preflight-nofail": {
			name: "Run preflight checks (informative only)",
			run:  r.preflightChecksNoFail,
		},
		"execute": {
			name: "Execute plan",
			run:  r.executePlan,
		},
	}
}

func (r *runner) takeOrReadSnapshot() error {
	preUpgradeObjs, err := snapshot.TakeOrReadSnapshot(r.ctx, true)
	if err != nil {
		return err
	}
	r.preUpgradeObjs = preUpgradeObjs
	r.preUpgradeState = k8sobjects.BuildObjectMap(preUpgradeObjs)
	return nil
}

func (r *runner) takeOrReadSnapshotNoStore() error {
	preUpgradeObjs, err := snapshot.TakeOrReadSnapshot(r.ctx, false)
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
		log.Errorf("Attempting to continue despite errors in preflight checks")
	}
	return nil
}

func (r *runner) executePlan() error {
	if err := execution.ExecutePlan(r.ctx, r.executionPlan); err != nil {
		return err
	}
	return nil
}
