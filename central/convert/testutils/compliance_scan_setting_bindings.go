package testutils

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	// ScanSettingUID -- scan UID used in test objects
	ScanSettingUID = uuid.NewV4().String()
	// TransitionTime -- transition time used in test objects
	TransitionTime = protocompat.TimestampNow()
)

// GetScanSettingBindingV1Storage -- returns V1 storage scan setting binding storage object
func GetScanSettingBindingV1Storage(_ *testing.T, clusterID string) *storage.ComplianceOperatorScanSettingBinding {
	return &storage.ComplianceOperatorScanSettingBinding{
		Id:        ScanSettingUID,
		Name:      "ocp-scan-setting-binding-name",
		ClusterId: clusterID,
	}
}

// GetScanSettingBindingV2Storage -- returns V2 storage scan setting binding storage object
func GetScanSettingBindingV2Storage(_ *testing.T, clusterID string) *storage.ComplianceOperatorScanSettingBindingV2 {
	return &storage.ComplianceOperatorScanSettingBindingV2{
		Id:              ScanSettingUID,
		ClusterId:       clusterID,
		Name:            "ocp-scan-setting-binding-name",
		ProfileNames:    []string{"profile-1", "profile-2"},
		ScanSettingName: "ocp-scan-setting-name",
		Status: &storage.ComplianceOperatorStatus{
			Phase: "Ready",
			Conditions: []*storage.ComplianceOperatorCondition{
				{
					Type:               "Ready",
					Status:             "True",
					Reason:             "Processed",
					Message:            "This is a message",
					LastTransitionTime: TransitionTime,
				},
			},
		},
	}
}

// GetScanSettingBindingV2SensorMsg -- returns V2 internal scan setting binding storage object
func GetScanSettingBindingV2SensorMsg(_ *testing.T) *central.ComplianceOperatorScanSettingBindingV2 {
	return &central.ComplianceOperatorScanSettingBindingV2{
		Id:              ScanSettingUID,
		Name:            "ocp-scan-setting-binding-name",
		ProfileNames:    []string{"profile-1", "profile-2"},
		ScanSettingName: "ocp-scan-setting-name",
		Status: &central.ComplianceOperatorStatus{
			Phase: "Ready",
			Conditions: []*central.ComplianceOperatorCondition{
				{
					Type:               "Ready",
					Status:             "True",
					Reason:             "Processed",
					Message:            "This is a message",
					LastTransitionTime: TransitionTime,
				},
			},
		},
	}
}
