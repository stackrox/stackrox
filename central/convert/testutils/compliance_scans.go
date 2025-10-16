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

	createTime = protocompat.TimestampNow()
	startTime  = protocompat.TimestampNow()
	endTime    = protocompat.TimestampNow()
)

// GetScanV2SensorMsg -- returns a V2 message from sensor
func GetScanV2SensorMsg(_ *testing.T) *central.ComplianceOperatorScanV2 {
	cossv2 := &central.ComplianceOperatorScanStatusV2{}
	cossv2.SetPhase("")
	cossv2.SetResult("FAIL")
	cossv2.SetErrorMessage("")
	cossv2.SetCurrentIndex(0)
	cossv2.SetWarnings("")
	cossv2.SetRemainingRetries(0)
	cossv2.SetLastStartTime(startTime)
	cossv2.SetStartTime(createTime)
	cossv2.SetEndTime(endTime)
	cosv2 := &central.ComplianceOperatorScanV2{}
	cosv2.SetId(ScanUID)
	cosv2.SetName("ocp-cis")
	cosv2.SetProfileId(profileID)
	cosv2.SetLabels(map[string]string{v1alpha1.SuiteLabel: "ocp-cis"})
	cosv2.SetAnnotations(nil)
	cosv2.SetScanType("")
	cosv2.SetStatus(cossv2)
	return cosv2
}

// GetScanV1Storage -- returns V1 storage scan object
func GetScanV1Storage(_ *testing.T) *storage.ComplianceOperatorScan {
	cos := &storage.ComplianceOperatorScan{}
	cos.SetId(ScanUID)
	cos.SetName("ocp-cis")
	cos.SetClusterId(fixtureconsts.Cluster1)
	cos.SetProfileId(profileID)
	cos.SetLabels(map[string]string{v1alpha1.SuiteLabel: "ocp-cis"})
	cos.SetAnnotations(nil)
	return cos
}

// GetScanV2Storage -- returns V2 storage scan object
func GetScanV2Storage(_ *testing.T) *storage.ComplianceOperatorScanV2 {
	ps := &storage.ProfileShim{}
	ps.SetProfileId(profileID)
	ps.SetProfileRefId(internaltov2storage.BuildProfileRefID(fixtureconsts.Cluster1, profileID, ""))
	ss := &storage.ScanStatus{}
	ss.SetPhase("")
	ss.SetResult("FAIL")
	ss.SetWarnings("")
	cosv2 := &storage.ComplianceOperatorScanV2{}
	cosv2.SetId(ScanUID)
	cosv2.SetScanConfigName("ocp-cis")
	cosv2.SetScanName("ocp-cis")
	cosv2.SetClusterId(fixtureconsts.Cluster1)
	cosv2.SetErrors("")
	cosv2.SetWarnings("")
	cosv2.SetProfile(ps)
	cosv2.SetLabels(map[string]string{v1alpha1.SuiteLabel: "ocp-cis"})
	cosv2.SetAnnotations(nil)
	cosv2.SetScanType(0)
	cosv2.SetNodeSelector(0)
	cosv2.SetStatus(ss)
	cosv2.SetCreatedTime(createTime)
	cosv2.SetLastStartedTime(startTime)
	cosv2.SetLastExecutedTime(endTime)
	cosv2.SetProductType("")
	cosv2.SetScanRefId(internaltov2storage.BuildNameRefID(fixtureconsts.Cluster1, "ocp-cis"))
	return cosv2
}
