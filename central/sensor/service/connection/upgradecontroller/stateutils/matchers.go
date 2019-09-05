package stateutils

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sensorupgrader"
	"github.com/stackrox/rox/pkg/set"
)

func anyStateFrom(desiredStates ...storage.UpgradeProgress_UpgradeState) *set.StorageUpgradeProgress_UpgradeStateSet {
	s := set.NewStorageUpgradeProgress_UpgradeStateSet(desiredStates...)
	return &s
}

func anyStageFrom(desiredStages ...sensorupgrader.Stage) *sensorupgrader.StageSet {
	s := sensorupgrader.NewStageSet(desiredStages...)
	return &s
}
