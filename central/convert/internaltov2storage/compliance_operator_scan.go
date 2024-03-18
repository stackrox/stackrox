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
	return &storage.ComplianceOperatorScanV2{
		Id:             sensorData.GetId(),
		ScanConfigName: sensorData.GetLabels()[v1alpha1.SuiteLabel],
		ScanName:       sensorData.GetName(),
		ClusterId:      clusterID,
		Errors:         sensorData.GetStatus().ErrorMessage,
		Profile: &storage.ProfileShim{
			ProfileId: BuildProfileRefID(clusterID, sensorData.GetProfileId(), sensorData.GetScanType()),
		},
		Labels:      sensorData.GetLabels(),
		Annotations: sensorData.GetAnnotations(),
		ScanType:    scanTypeToV2[sensorData.GetScanType()],
		Status: &storage.ScanStatus{
			Phase:    sensorData.GetStatus().GetPhase(),
			Result:   sensorData.GetStatus().GetResult(),
			Warnings: sensorData.GetStatus().GetWarnings(),
		},
		CreatedTime:      sensorData.GetStatus().GetStartTime(),
		LastExecutedTime: sensorData.GetStatus().GetEndTime(),
		ProductType:      sensorData.GetScanType(),
		ScanRefId:        BuildScanRefID(clusterID, sensorData.GetName()),
	}
}
