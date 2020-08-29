package store

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
)

// GetMockResult returns a mock ComplianceRunResults object
func GetMockResult() *storage.ComplianceRunResults {
	return &storage.ComplianceRunResults{
		Domain: &storage.ComplianceDomain{},
		RunMetadata: &storage.ComplianceRunMetadata{
			StandardId:      "yeet",
			Success:         true,
			RunId:           "Test run ID",
			ClusterId:       "Test cluster ID",
			FinishTimestamp: types.TimestampNow(),
		},
		ClusterResults: &storage.ComplianceRunResults_EntityResults{},
	}
}
