package internaltov2storage

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

// ComplianceOperatorRemediation converts internal api V2 remediation to a storage V2 remediation
func ComplianceOperatorRemediation(sensorData *central.ComplianceOperatorRemediationV2, clusterID string) *storage.ComplianceOperatorRemediationV2 {
	return &storage.ComplianceOperatorRemediationV2{
		Id:                        sensorData.GetId(),
		Name:                      sensorData.GetName(),
		ComplianceCheckResultName: sensorData.GetComplianceCheckResultName(),
		EnforcementType:           sensorData.GetEnforcementType(),
		OutdatedObject:            sensorData.GetOutdatedObject(),
		CurrentObject:             sensorData.GetCurrentObject(),
		ClusterId:                 clusterID,
		Apply:                     sensorData.GetApply(),
	}
}
