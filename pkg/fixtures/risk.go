package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
)

// GetRisk returns a mock Risk
func GetRisk() *storage.Risk {
	return &storage.Risk{
		Score: 10,
		Entity: &storage.RiskEntityMeta{
			Id:        "FakeID",
			Namespace: "FakeNS",
			ClusterId: "FakeClusterID",
			Type:      storage.RiskEntityType_DEPLOYMENT,
		},
		Results: []*storage.Risk_Result{
			{Name: "BLAH"},
			{Name: "BLAH2"},
		},
	}
}
