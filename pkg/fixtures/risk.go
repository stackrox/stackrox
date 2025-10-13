package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
)

// GetRisk returns a mock Risk.
func GetRisk() *storage.Risk {
	return &storage.Risk{
		Id:    fixtureconsts.Deployment1,
		Score: 10,
		Subject: &storage.RiskSubject{
			Id:        fixtureconsts.Deployment1,
			Namespace: fixtureconsts.Namespace1,
			ClusterId: fixtureconsts.Cluster1,
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
