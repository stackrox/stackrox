package sensorupgrader

// A Stage represents a stage in the sensor upgrader process.
//
//go:generate stringer -type=Stage
type Stage int

var (
	stageByName = func() map[string]Stage {
		m := make(map[string]Stage)
		for stage := UnsetStage + 1; stage < placeholderEndStage; stage++ {
			m[stage.String()] = stage
		}
		return m
	}()
)

// This block enumerates all known stages.
const (
	UnsetStage Stage = iota
	CleanupForeignStateStage
	SnapshotForRollForwardStage
	SnapshotForRollbackStage
	SnapshotForDryRunStage
	FetchBundleStage
	InstantiateBundleStage
	GeneratePlanStage
	GenerateRollbackPlanStage
	PreflightStage
	PreflightNoFailStage
	ExecuteStage
	CleanupOwnerStage
	WaitForDeletionStage

	placeholderEndStage
)

// GetStage gets the stage corresponding to the current name.
// It returns UnsetStage if it the name doesn't match any stage
// (or, of course, if name is "UnsetStage").
func GetStage(name string) Stage {
	stage, ok := stageByName[name]
	if !ok {
		return UnsetStage
	}
	return stage
}
