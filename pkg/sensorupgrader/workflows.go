package sensorupgrader

// This block enumerates all known workflows.
const (
	RollForwardWorkflow    = "roll-forward"
	RollBackWorkflow       = "roll-back"
	DryRunWorkflow         = "dry-run"
	ValidateBundleWorkflow = "validate-bundle"
	CleanupWorkflow        = "cleanup"
)

// Workflows defines all valid workflows for the upgrader,
// and maps them to an ordered list of stage names.
func Workflows() map[string][]Stage {
	return map[string][]Stage{
		RollForwardWorkflow: {
			CleanupForeignStateStage,
			SnapshotForRollForwardStage,
			FetchBundleStage,
			InstantiateBundleStage,
			GeneratePlanStage,
			PreflightStage,
			ExecuteStage,
		},
		RollBackWorkflow: {
			SnapshotForRollbackStage,
			GenerateRollbackPlanStage,
			PreflightNoFailStage,
			ExecuteStage,
		},
		DryRunWorkflow: {
			SnapshotForDryRunStage,
			FetchBundleStage,
			InstantiateBundleStage,
			GeneratePlanStage,
			PreflightStage,
		},
		ValidateBundleWorkflow: {
			FetchBundleStage,
			InstantiateBundleStage,
		},
		CleanupWorkflow: {
			CleanupOwnerStage,
			WaitForDeletionStage,
		},
	}
}
