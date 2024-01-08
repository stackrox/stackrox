package internaltov1storage

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

// ComplianceOperatorScanObject converts internal api V2 check result to a V1 storage check result
func ComplianceOperatorScanObject(internalScan *central.ComplianceOperatorScanV2, clusterID string) *storage.ComplianceOperatorScan {
	return &storage.ComplianceOperatorScan{
		Id:          internalScan.GetId(),
		Name:        internalScan.GetName(),
		ClusterId:   clusterID,
		ProfileId:   internalScan.GetProfileId(),
		Labels:      internalScan.GetLabels(),
		Annotations: internalScan.GetAnnotations(),
	}
}
