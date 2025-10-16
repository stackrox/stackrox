package store

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
)

// GetMockResult returns a mock ComplianceRunResults object
func GetMockResult() (*storage.ComplianceRunResults, *storage.ComplianceDomain) {
	clusterID := "Test cluster ID"
	domainID := "a very good domain ID"

	cc := &storage.ComplianceDomain_Cluster{}
	cc.SetId(clusterID)
	domain := &storage.ComplianceDomain{}
	domain.SetId(domainID)
	domain.SetCluster(cc)

	result := storage.ComplianceRunResults_builder{
		RunMetadata: storage.ComplianceRunMetadata_builder{
			StandardId:      "yeet",
			Success:         true,
			RunId:           "Test run ID",
			ClusterId:       clusterID,
			FinishTimestamp: protocompat.TimestampNow(),
			DomainId:        domainID,
		}.Build(),
		ClusterResults: storage.ComplianceRunResults_EntityResults_builder{
			ControlResults: map[string]*storage.ComplianceResultValue{
				"test": storage.ComplianceResultValue_builder{
					Evidence: []*storage.ComplianceResultValue_Evidence{
						storage.ComplianceResultValue_Evidence_builder{
							Message: "test",
						}.Build(),
					},
					OverallState: 0,
				}.Build(),
			},
		}.Build(),
		Domain: domain,
	}.Build()
	return result, domain
}
