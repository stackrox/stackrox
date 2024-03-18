package testutils

import (
	"testing"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/central/convert/internaltov2storage"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	// ScanUID -- scan UID used in test objects
	ScanUID = uuid.NewV4().String()

	startTime = protocompat.TimestampNow()
	endTime   = protocompat.TimestampNow()
)

// GetScanV2SensorMsg -- returns a V2 message from sensor
func GetScanV2SensorMsg(_ *testing.T) *central.ComplianceOperatorScanV2 {
	return &central.ComplianceOperatorScanV2{
		Id:          ScanUID,
		Name:        "ocp-cis",
		ProfileId:   profileID,
		Labels:      map[string]string{v1alpha1.SuiteLabel: "ocp-cis"},
		Annotations: nil,
		ScanType:    "",
		Status: &central.ComplianceOperatorScanStatusV2{
			Phase:            "",
			Result:           "FAIL",
			ErrorMessage:     "",
			CurrentIndex:     0,
			Warnings:         "",
			RemainingRetries: 0,
			StartTime:        startTime,
			EndTime:          endTime,
		},
	}
}

// GetScanV1Storage -- returns V1 storage scan object
func GetScanV1Storage(_ *testing.T) *storage.ComplianceOperatorScan {
	return &storage.ComplianceOperatorScan{
		Id:          ScanUID,
		Name:        "ocp-cis",
		ClusterId:   fixtureconsts.Cluster1,
		ProfileId:   profileID,
		Labels:      map[string]string{v1alpha1.SuiteLabel: "ocp-cis"},
		Annotations: nil,
	}
}

// GetScanV2Storage -- returns V2 storage scan object
func GetScanV2Storage(_ *testing.T) *storage.ComplianceOperatorScanV2 {
	return &storage.ComplianceOperatorScanV2{
		Id:             ScanUID,
		ScanConfigName: "ocp-cis",
		ScanName:       "ocp-cis",
		ClusterId:      fixtureconsts.Cluster1,
		Errors:         "",
		Warnings:       "",
		Profile: &storage.ProfileShim{
			ProfileId: internaltov2storage.BuildProfileRefID(fixtureconsts.Cluster1, profileID, ""),
		},
		Labels:       map[string]string{v1alpha1.SuiteLabel: "ocp-cis"},
		Annotations:  nil,
		ScanType:     0,
		NodeSelector: 0,
		Status: &storage.ScanStatus{
			Phase:    "",
			Result:   "FAIL",
			Warnings: "",
		},
		CreatedTime:      startTime,
		LastExecutedTime: endTime,
		ProductType:      "",
		ScanRefId:        internaltov2storage.BuildScanRefID(fixtureconsts.Cluster1, "ocp-cis"),
	}
}
