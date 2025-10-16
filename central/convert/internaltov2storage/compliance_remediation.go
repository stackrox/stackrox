package internaltov2storage

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

// ComplianceOperatorRemediation converts internal api V2 remediation to a storage V2 remediation
func ComplianceOperatorRemediation(sensorData *central.ComplianceOperatorRemediationV2, clusterID string) *storage.ComplianceOperatorRemediationV2 {
	corv2 := &storage.ComplianceOperatorRemediationV2{}
	corv2.SetId(sensorData.GetId())
	corv2.SetName(sensorData.GetName())
	corv2.SetComplianceCheckResultName(sensorData.GetComplianceCheckResultName())
	corv2.SetEnforcementType(sensorData.GetEnforcementType())
	corv2.SetOutdatedObject(sensorData.GetOutdatedObject())
	corv2.SetCurrentObject(sensorData.GetCurrentObject())
	corv2.SetClusterId(clusterID)
	corv2.SetApply(sensorData.GetApply())
	return corv2
}
