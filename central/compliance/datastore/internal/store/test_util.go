package store

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"google.golang.org/protobuf/proto"
)

// GetMockResult returns a mock ComplianceRunResults object
func GetMockResult() (*storage.ComplianceRunResults, *storage.ComplianceDomain) {
	clusterID := "Test cluster ID"
	domainID := "a very good domain ID"

	domain := storage.ComplianceDomain_builder{
		Id: &domainID,
		Cluster: storage.ComplianceDomain_Cluster_builder{
			Id: &clusterID,
		}.Build(),
	}.Build()

	message := "test"
	result := storage.ComplianceRunResults_builder{
		RunMetadata: storage.ComplianceRunMetadata_builder{
			StandardId:      proto.String("yeet"),
			Success:         proto.Bool(true),
			RunId:           proto.String("Test run ID"),
			ClusterId:       &clusterID,
			FinishTimestamp: protocompat.TimestampNow(),
			DomainId:        &domainID,
		}.Build(),
		ClusterResults: storage.ComplianceRunResults_EntityResults_builder{
			ControlResults: map[string]*storage.ComplianceResultValue{
				"test": storage.ComplianceResultValue_builder{
					Evidence: []*storage.ComplianceResultValue_Evidence{
						storage.ComplianceResultValue_Evidence_builder{
							Message: &message,
						}.Build(),
					},
					OverallState: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS.Enum(),
				}.Build(),
			},
		}.Build(),
		Domain: domain,
	}.Build()
	return result, domain
}
