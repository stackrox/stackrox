package stateutils

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sensorupgrader"
	"github.com/stackrox/rox/pkg/set"
)

func anyStateFrom(desiredStates ...storage.UpgradeProgress_UpgradeState) *set.Set[storage.UpgradeProgress_UpgradeState] {
	s := set.NewSet(desiredStates...)
	return &s
}

func anyStageFrom(desiredStages ...sensorupgrader.Stage) *set.Set[sensorupgrader.Stage] {
	s := set.NewSet(desiredStages...)
	return &s
}
