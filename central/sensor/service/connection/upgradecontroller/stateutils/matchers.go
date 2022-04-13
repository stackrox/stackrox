package stateutils

import (
	"github.com/stackrox/stackrox/generated/set"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/sensorupgrader"
)

func anyStateFrom(desiredStates ...storage.UpgradeProgress_UpgradeState) *set.StorageUpgradeProgress_UpgradeStateSet {
	s := set.NewStorageUpgradeProgress_UpgradeStateSet(desiredStates...)
	return &s
}

func anyStageFrom(desiredStages ...sensorupgrader.Stage) *sensorupgrader.StageSet {
	s := sensorupgrader.NewStageSet(desiredStages...)
	return &s
}
