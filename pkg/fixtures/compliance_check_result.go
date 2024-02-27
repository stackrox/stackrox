package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

func GetComplianceCheckResult(name, clusterID, clusterName, scanName, scanConfigName string) *storage.ComplianceOperatorCheckResultV2 {
	return &storage.ComplianceOperatorCheckResultV2{
		Id:             uuid.NewV4().String(),
		CheckId:        name,
		CheckName:      name,
		ClusterId:      clusterID,
		Status:         0,
		Severity:       0,
		Description:    "test description " + name,
		Instructions:   "test instruction " + name,
		Labels:         nil,
		Annotations:    nil,
		CreatedTime:    nil,
		ValuesUsed:     nil,
		Warnings:       nil,
		ScanName:       scanName,
		ClusterName:    clusterName,
		ScanConfigName: scanConfigName,
		Rationale:      "test rationale " + name,
	}
}
