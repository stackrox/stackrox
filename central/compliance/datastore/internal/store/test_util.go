package store

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
)

// GetMockResult returns a mock ComplianceRunResults object
func GetMockResult() (*storage.ComplianceRunResults, *storage.ComplianceDomain) {
	clusterID := "Test cluster ID"
	domainID := "a very good domain ID"

	domain := &storage.ComplianceDomain{
		Id: domainID,
		Cluster: &storage.ComplianceDomain_Cluster{
			Id: clusterID,
		},
	}

	result := &storage.ComplianceRunResults{
		RunMetadata: &storage.ComplianceRunMetadata{
			StandardId:      "yeet",
			Success:         true,
			RunId:           "Test run ID",
			ClusterId:       clusterID,
			FinishTimestamp: protocompat.TimestampNow(),
			DomainId:        domainID,
		},
		ClusterResults: &storage.ComplianceRunResults_EntityResults{
			ControlResults: map[string]*storage.ComplianceResultValue{
				"test": {
					Evidence: []*storage.ComplianceResultValue_Evidence{
						{
							Message: "test",
						},
					},
					OverallState: 0,
				},
			},
		},
		Domain: domain,
	}
	return result, domain
}
