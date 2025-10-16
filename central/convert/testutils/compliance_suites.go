package testutils

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	// SuiteUID -- compliance suite UID used in test objects
	SuiteUID = uuid.NewV4().String()

	// TransitionTimeForCondition1 -- fixed timestamp used in suite conditions
	TransitionTimeForCondition1 = protocompat.GetProtoTimestampFromSeconds(1706633068)

	// TransitionTimeForCondition2 -- fixed timestamp used in suite conditions
	TransitionTimeForCondition2 = protocompat.GetProtoTimestampFromSeconds(1706634000)
)

// GetSuiteStorage -- returns suite storage
func GetSuiteStorage(_ *testing.T) *storage.ComplianceOperatorSuiteV2 {
	coc := &storage.ComplianceOperatorCondition{}
	coc.SetType("Processing")
	coc.SetStatus("False")
	coc.SetReason("NotRunning")
	coc.SetMessage("This is message 1")
	coc.SetLastTransitionTime(TransitionTimeForCondition1)
	coc2 := &storage.ComplianceOperatorCondition{}
	coc2.SetType("Ready")
	coc2.SetStatus("True")
	coc2.SetReason("Done")
	coc2.SetLastTransitionTime(TransitionTimeForCondition2)
	status := &storage.ComplianceOperatorStatus{}
	status.SetPhase("DONE")
	status.SetResult("NON-COMPLIANT")
	status.SetErrorMessage("some error")
	status.SetConditions([]*storage.ComplianceOperatorCondition{
		coc,
		coc2,
	})

	cosv2 := &storage.ComplianceOperatorSuiteV2{}
	cosv2.SetId(SuiteUID)
	cosv2.SetName("compliancesuitename")
	cosv2.SetStatus(status)
	cosv2.SetClusterId(fixtureconsts.Cluster1)
	return cosv2
}

// GetSuiteSensorMsg -- returns a suite message from sensor
func GetSuiteSensorMsg(_ *testing.T) *central.ComplianceOperatorSuiteV2 {
	coc := &central.ComplianceOperatorCondition{}
	coc.SetType("Processing")
	coc.SetStatus("False")
	coc.SetReason("NotRunning")
	coc.SetMessage("This is message 1")
	coc.SetLastTransitionTime(TransitionTimeForCondition1)
	coc2 := &central.ComplianceOperatorCondition{}
	coc2.SetType("Ready")
	coc2.SetStatus("True")
	coc2.SetReason("Done")
	coc2.SetLastTransitionTime(TransitionTimeForCondition2)
	status := &central.ComplianceOperatorStatus{}
	status.SetPhase("DONE")
	status.SetResult("NON-COMPLIANT")
	status.SetErrorMessage("some error")
	status.SetConditions([]*central.ComplianceOperatorCondition{
		coc,
		coc2,
	})

	cosv2 := &central.ComplianceOperatorSuiteV2{}
	cosv2.SetId(SuiteUID)
	cosv2.SetName("compliancesuitename")
	cosv2.SetStatus(status)
	return cosv2
}
