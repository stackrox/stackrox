package stateutils

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/pointers"
	"github.com/stackrox/stackrox/pkg/sensorupgrader"
)

func statePtr(state storage.UpgradeProgress_UpgradeState) *storage.UpgradeProgress_UpgradeState {
	return &state
}

func upgradeTypePtr(typ storage.ClusterUpgradeStatus_UpgradeProcessStatus_UpgradeProcessType) *storage.ClusterUpgradeStatus_UpgradeProcessStatus_UpgradeProcessType {
	return &typ
}

var (
	// These define all the valid transitions we handle.
	// Note that the first match wins, so order these transitions with that in mind.
	allTransitions = []transitioner{
		// If the upgrade is in a terminal state, just tell it to clean up.
		{
			currentStateMatch: anyStateFrom(TerminalStates.AsSlice()...),

			noStateChange:     true,
			workflowToExecute: sensorupgrader.CleanupWorkflow,
		},

		// The following transitions handle the situation right after the upgrader comes up.
		// (Indicated by an empty string for the workflow.)
		// Note that the upgrader might restart at any time.
		// So we MUST handle all non-terminal states through the below transitions.

		{
			workflowMatch: pointers.String(""),
			currentStateMatch: anyStateFrom(
				storage.UpgradeProgress_UPGRADE_INITIALIZING, // This should basically never happen, but being defensive can't hurt.

				// This would be a little early to hear from the upgrader, but still possible in case
				// the upgrader happens to reach out before sensor for whatever reason.
				storage.UpgradeProgress_UPGRADER_LAUNCHING,

				storage.UpgradeProgress_UPGRADER_LAUNCHED, // This is the stage where we normally expect to hear from the upgrader.

				// Seeing the below states likely means the upgrader restarted part way through the process. However, we haven't heard
				// from the sensor yet (else we'd say upgrade complete), and the upgrader is idempotent, so tell it to roll-forward anyway.
				storage.UpgradeProgress_PRE_FLIGHT_CHECKS_COMPLETE,
				storage.UpgradeProgress_UPGRADE_OPERATIONS_DONE,
			),

			workflowToExecute: sensorupgrader.RollForwardWorkflow,
			nextState:         statePtr(storage.UpgradeProgress_UPGRADER_LAUNCHED),
		},
		{
			// Upgrader restarted in the middle of rolling back. Tell it to keep rolling back.
			workflowMatch:     pointers.String(""),
			currentStateMatch: anyStateFrom(storage.UpgradeProgress_UPGRADE_ERROR_ROLLING_BACK),

			workflowToExecute: sensorupgrader.RollBackWorkflow,
			nextState:         statePtr(storage.UpgradeProgress_UPGRADE_ERROR_ROLLING_BACK),
		},

		// The following are roll-forward transitions.
		// Note that we don't check the starting state here (we know it's not terminal since that was checked above,
		// and the end state only depends on the upgrader action).
		{
			workflowMatch:    pointers.String(sensorupgrader.RollForwardWorkflow),
			stageMatch:       anyStageFrom(rollForwardStagesBeforePreFlight...),
			errOccurredMatch: pointers.Bool(false),

			workflowToExecute: sensorupgrader.RollForwardWorkflow,
			nextState:         statePtr(storage.UpgradeProgress_UPGRADER_LAUNCHED),
		},
		// An error occurred before we could even do pre-flight checks!
		// Mark it as a fatal error, and tell the upgrader to clean up.
		{
			workflowMatch:    pointers.String(sensorupgrader.RollForwardWorkflow),
			stageMatch:       anyStageFrom(rollForwardStagesBeforePreFlight...),
			errOccurredMatch: pointers.Bool(true),

			workflowToExecute: sensorupgrader.CleanupWorkflow,
			nextState:         statePtr(storage.UpgradeProgress_UPGRADE_INITIALIZATION_ERROR),
			updateDetail:      true,
		},
		// Yay, passed pre-flight checks!
		{
			workflowMatch:    pointers.String(sensorupgrader.RollForwardWorkflow),
			stageMatch:       anyStageFrom(sensorupgrader.PreflightStage),
			errOccurredMatch: pointers.Bool(false),

			workflowToExecute: sensorupgrader.RollForwardWorkflow,
			nextState:         statePtr(storage.UpgradeProgress_PRE_FLIGHT_CHECKS_COMPLETE),
		},
		// Oh no, pre-flight checks failed!
		{
			workflowMatch:    pointers.String(sensorupgrader.RollForwardWorkflow),
			stageMatch:       anyStageFrom(sensorupgrader.PreflightStage),
			errOccurredMatch: pointers.Bool(true),

			workflowToExecute: sensorupgrader.CleanupWorkflow,
			nextState:         statePtr(storage.UpgradeProgress_PRE_FLIGHT_CHECKS_FAILED),
			updateDetail:      true,
		},
		// Ooh yeah, upgrade done from the PoV of the upgrader!
		{
			workflowMatch:    pointers.String(sensorupgrader.RollForwardWorkflow),
			stageMatch:       anyStageFrom(sensorupgrader.ExecuteStage),
			errOccurredMatch: pointers.Bool(false),
			upgradeTypeMatch: upgradeTypePtr(storage.ClusterUpgradeStatus_UpgradeProcessStatus_UPGRADE),

			// For upgrades, tell the upgrader to stay in the roll-forward workflow, and keep polling until
			// we ask it to clean up (after we hear from the sensor).
			workflowToExecute: sensorupgrader.RollForwardWorkflow,
			nextState:         statePtr(storage.UpgradeProgress_UPGRADE_OPERATIONS_DONE),
		},
		// Ooh yeah, upgrade done from the PoV of the upgrader!
		{
			workflowMatch:    pointers.String(sensorupgrader.RollForwardWorkflow),
			stageMatch:       anyStageFrom(sensorupgrader.ExecuteStage),
			errOccurredMatch: pointers.Bool(false),
			upgradeTypeMatch: upgradeTypePtr(storage.ClusterUpgradeStatus_UpgradeProcessStatus_CERT_ROTATION),

			// For cert rotation, when the upgrader says it's done, we mark the upgrade complete.
			workflowToExecute: sensorupgrader.CleanupWorkflow,
			nextState:         statePtr(storage.UpgradeProgress_UPGRADE_COMPLETE),
		},
		// Oh no, upgrade operations failed. :( Tell the upgrader to roll back.
		{
			workflowMatch:    pointers.String(sensorupgrader.RollForwardWorkflow),
			stageMatch:       anyStageFrom(sensorupgrader.ExecuteStage),
			errOccurredMatch: pointers.Bool(true),

			workflowToExecute: sensorupgrader.RollBackWorkflow,
			nextState:         statePtr(storage.UpgradeProgress_UPGRADE_ERROR_ROLLING_BACK),
			updateDetail:      true,
		},

		// The following are roll-back transitions.

		// Rollback still in progress.
		{
			workflowMatch:    pointers.String(sensorupgrader.RollBackWorkflow),
			stageMatch:       anyStageFrom(sensorupgrader.SnapshotForRollbackStage, sensorupgrader.GenerateRollbackPlanStage, sensorupgrader.PreflightNoFailStage),
			errOccurredMatch: pointers.Bool(false),

			workflowToExecute: sensorupgrader.RollBackWorkflow,
			nextState:         statePtr(storage.UpgradeProgress_UPGRADE_ERROR_ROLLING_BACK),
		},
		// Rollback done, now clean up.
		{
			workflowMatch:    pointers.String(sensorupgrader.RollBackWorkflow),
			stageMatch:       anyStageFrom(sensorupgrader.ExecuteStage),
			errOccurredMatch: pointers.Bool(false),
			upgradeTypeMatch: upgradeTypePtr(storage.ClusterUpgradeStatus_UpgradeProcessStatus_UPGRADE),

			workflowToExecute: sensorupgrader.CleanupWorkflow,
			// On upgrades, don't mark as rolled back until the sensor checks in.
			nextState: statePtr(storage.UpgradeProgress_UPGRADE_ERROR_ROLLING_BACK),
		},
		{
			workflowMatch:    pointers.String(sensorupgrader.RollBackWorkflow),
			stageMatch:       anyStageFrom(sensorupgrader.ExecuteStage),
			errOccurredMatch: pointers.Bool(false),
			upgradeTypeMatch: upgradeTypePtr(storage.ClusterUpgradeStatus_UpgradeProcessStatus_CERT_ROTATION),

			workflowToExecute: sensorupgrader.CleanupWorkflow,
			// On cert rotation, we mark as rolled back when the upgrader says it has rolled back.
			nextState: statePtr(storage.UpgradeProgress_UPGRADE_ERROR_ROLLED_BACK),
		},
		// Any error when rolling back => rollback failed. Not much we can do at this point. :(
		{
			workflowMatch:    pointers.String(sensorupgrader.RollBackWorkflow),
			errOccurredMatch: pointers.Bool(true),

			// Upgrader might as well clean up.
			workflowToExecute: sensorupgrader.CleanupWorkflow,
			nextState:         statePtr(storage.UpgradeProgress_UPGRADE_ERROR_ROLLBACK_FAILED),
			updateDetail:      true,
		},

		// The only non-terminal state where we ask the upgrader to clean up is "ROLLING_BACK",
		// since the sensor is the one that successfully reports a rollback. We don't want to ask
		// the upgrader to keep polling because there's not really anything further we can expect from it
		// expect to clean up eventually.
		{
			currentStateMatch: anyStateFrom(storage.UpgradeProgress_UPGRADE_ERROR_ROLLING_BACK),
			workflowMatch:     pointers.String(sensorupgrader.CleanupWorkflow),
			workflowToExecute: sensorupgrader.CleanupWorkflow,
			noStateChange:     true,
		},
	}
)
