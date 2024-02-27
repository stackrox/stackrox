package fixtures

import (
	"math/rand"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

// GetComplianceCheckResult returns a test compliance check result
func GetComplianceCheckResult(name, clusterID, clusterName, scanName, scanConfigName string) *storage.ComplianceOperatorCheckResultV2 {
	return &storage.ComplianceOperatorCheckResultV2{
		Id:             uuid.NewV4().String(),
		CheckId:        name,
		CheckName:      name,
		ClusterId:      clusterID,
		Status:         storage.ComplianceOperatorCheckResultV2_CheckStatus(rand.Intn(7) + 1),
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
