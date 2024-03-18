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
	status := &storage.ComplianceOperatorStatus{
		Phase:        "DONE",
		Result:       "NON-COMPLIANT",
		ErrorMessage: "some error",
		Conditions: []*storage.ComplianceOperatorCondition{
			{
				Type:               "Processing",
				Status:             "False",
				Reason:             "NotRunning",
				Message:            "This is message 1",
				LastTransitionTime: TransitionTimeForCondition1,
			},
			{
				Type:               "Ready",
				Status:             "True",
				Reason:             "Done",
				LastTransitionTime: TransitionTimeForCondition2,
			},
		},
	}

	return &storage.ComplianceOperatorSuiteV2{
		Id:        SuiteUID,
		Name:      "compliancesuitename",
		Status:    status,
		ClusterId: fixtureconsts.Cluster1,
	}
}

// GetSuiteSensorMsg -- returns a suite message from sensor
func GetSuiteSensorMsg(_ *testing.T) *central.ComplianceOperatorSuiteV2 {
	status := &central.ComplianceOperatorStatus{
		Phase:        "DONE",
		Result:       "NON-COMPLIANT",
		ErrorMessage: "some error",
		Conditions: []*central.ComplianceOperatorCondition{
			{
				Type:               "Processing",
				Status:             "False",
				Reason:             "NotRunning",
				Message:            "This is message 1",
				LastTransitionTime: TransitionTimeForCondition1,
			},
			{
				Type:               "Ready",
				Status:             "True",
				Reason:             "Done",
				LastTransitionTime: TransitionTimeForCondition2,
			},
		},
	}

	return &central.ComplianceOperatorSuiteV2{
		Id:     SuiteUID,
		Name:   "compliancesuitename",
		Status: status,
	}
}
