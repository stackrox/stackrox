package internaltov2storage

import (
	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

var (
	scanTypeToV2 = map[string]storage.ScanType{
		"Node":     storage.ScanType_NODE_SCAN,
		"Platform": storage.ScanType_PLATFORM_SCAN,
	}
)

// ComplianceOperatorScanObject converts internal api V2 compliance scan object to a V2 storage compliance scan object
func ComplianceOperatorScanObject(sensorData *central.ComplianceOperatorScanV2, clusterID string) *storage.ComplianceOperatorScanV2 {
	ps := &storage.ProfileShim{}
	ps.SetProfileId(sensorData.GetProfileId())
	ps.SetProfileRefId(BuildProfileRefID(clusterID, sensorData.GetProfileId(), sensorData.GetScanType()))
	ss := &storage.ScanStatus{}
	ss.SetPhase(sensorData.GetStatus().GetPhase())
	ss.SetResult(sensorData.GetStatus().GetResult())
	ss.SetWarnings(sensorData.GetStatus().GetWarnings())
	cosv2 := &storage.ComplianceOperatorScanV2{}
	cosv2.SetId(sensorData.GetId())
	cosv2.SetScanConfigName(sensorData.GetLabels()[v1alpha1.SuiteLabel])
	cosv2.SetScanName(sensorData.GetName())
	cosv2.SetClusterId(clusterID)
	cosv2.SetErrors(sensorData.GetStatus().GetErrorMessage())
	cosv2.SetProfile(ps)
	cosv2.SetLabels(sensorData.GetLabels())
	cosv2.SetAnnotations(sensorData.GetAnnotations())
	cosv2.SetScanType(scanTypeToV2[sensorData.GetScanType()])
	cosv2.SetStatus(ss)
	cosv2.SetCreatedTime(sensorData.GetStatus().GetStartTime())
	cosv2.SetLastStartedTime(sensorData.GetStatus().GetLastStartTime())
	cosv2.SetLastExecutedTime(sensorData.GetStatus().GetEndTime())
	cosv2.SetProductType(sensorData.GetScanType())
	cosv2.SetScanRefId(BuildNameRefID(clusterID, sensorData.GetName()))
	return cosv2
}
