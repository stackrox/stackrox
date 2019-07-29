package fixtures

import "github.com/stackrox/rox/generated/storage"

// GetRisk returns a mock Risk
func GetRisk() *storage.Risk {
	return &storage.Risk{
		Score: 10,
		Subject: &storage.RiskSubject{
			Id:        "FakeID",
			Namespace: "FakeNS",
			ClusterId: "FakeClusterID",
			Type:      storage.RiskSubjectType_DEPLOYMENT,
		},
		Results: []*storage.Risk_Result{},
	}
}
