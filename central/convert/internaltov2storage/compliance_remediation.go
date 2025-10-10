package internaltov2storage

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

// ComplianceOperatorRemediation converts internal api V2 remediation to a storage V2 remediation
func ComplianceOperatorRemediation(sensorData *central.ComplianceOperatorRemediationV2, clusterID string) *storage.ComplianceOperatorRemediationV2 {
	id := sensorData.GetId()
	name := sensorData.GetName()
	complianceCheckResultName := sensorData.GetComplianceCheckResultName()
	enforcementType := sensorData.GetEnforcementType()
	outdatedObject := sensorData.GetOutdatedObject()
	currentObject := sensorData.GetCurrentObject()
	apply := sensorData.GetApply()

	return storage.ComplianceOperatorRemediationV2_builder{
		Id:                        &id,
		Name:                      &name,
		ComplianceCheckResultName: &complianceCheckResultName,
		EnforcementType:           &enforcementType,
		OutdatedObject:            &outdatedObject,
		CurrentObject:             &currentObject,
		ClusterId:                 &clusterID,
		Apply:                     &apply,
	}.Build()
}
