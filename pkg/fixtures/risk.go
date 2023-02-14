package fixtures

import "github.com/stackrox/rox/generated/storage"

// GetRisk returns a mock Risk.
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

// GetScopedRisk returns a mock Risk belonging to the input scope.
func GetScopedRisk(id string, clusterID string, namespace string) *storage.Risk {
	return &storage.Risk{
		Id: id,
		Subject: &storage.RiskSubject{
			Id:        id,
			Namespace: namespace,
			ClusterId: clusterID,
			Type:      storage.RiskSubjectType_DEPLOYMENT,
		},
	}
}
