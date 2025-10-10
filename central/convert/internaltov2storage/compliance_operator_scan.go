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
	id := sensorData.GetId()
	scanConfigName := sensorData.GetLabels()[v1alpha1.SuiteLabel]
	scanName := sensorData.GetName()
	errors := sensorData.GetStatus().GetErrorMessage()
	labels := sensorData.GetLabels()
	annotations := sensorData.GetAnnotations()
	scanType := scanTypeToV2[sensorData.GetScanType()]
	createdTime := sensorData.GetStatus().GetStartTime()
	lastStartedTime := sensorData.GetStatus().GetLastStartTime()
	lastExecutedTime := sensorData.GetStatus().GetEndTime()
	productType := sensorData.GetScanType()
	scanRefId := BuildNameRefID(clusterID, sensorData.GetName())

	profileId := sensorData.GetProfileId()
	profileRefId := BuildProfileRefID(clusterID, sensorData.GetProfileId(), sensorData.GetScanType())
	profile := storage.ProfileShim_builder{
		ProfileId:    &profileId,
		ProfileRefId: &profileRefId,
	}.Build()

	phase := sensorData.GetStatus().GetPhase()
	result := sensorData.GetStatus().GetResult()
	warnings := sensorData.GetStatus().GetWarnings()
	status := storage.ScanStatus_builder{
		Phase:    &phase,
		Result:   &result,
		Warnings: &warnings,
	}.Build()

	return storage.ComplianceOperatorScanV2_builder{
		Id:               &id,
		ScanConfigName:   &scanConfigName,
		ScanName:         &scanName,
		ClusterId:        &clusterID,
		Errors:           &errors,
		Profile:          profile,
		Labels:           labels,
		Annotations:      annotations,
		ScanType:         &scanType,
		Status:           status,
		CreatedTime:      createdTime,
		LastStartedTime:  lastStartedTime,
		LastExecutedTime: lastExecutedTime,
		ProductType:      &productType,
		ScanRefId:        &scanRefId,
	}.Build()
}
