package stateutils

import (
	"github.com/stackrox/rox/pkg/sensorupgrader"
)

var (
	rollForwardStagesBeforePreFlight = []sensorupgrader.Stage{
		sensorupgrader.CleanupForeignStateStage,
		sensorupgrader.SnapshotForRollForwardStage,
		sensorupgrader.FetchBundleStage,
		sensorupgrader.InstantiateBundleStage,
		sensorupgrader.GeneratePlanStage,
	}
)
