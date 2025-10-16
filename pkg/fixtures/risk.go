package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
)

// GetRisk returns a mock Risk.
func GetRisk() *storage.Risk {
	rs := &storage.RiskSubject{}
	rs.SetId(fixtureconsts.Deployment1)
	rs.SetNamespace(fixtureconsts.Namespace1)
	rs.SetClusterId(fixtureconsts.Cluster1)
	rs.SetType(storage.RiskSubjectType_DEPLOYMENT)
	risk := &storage.Risk{}
	risk.SetId(fixtureconsts.Deployment1)
	risk.SetScore(10)
	risk.SetSubject(rs)
	risk.SetResults([]*storage.Risk_Result{})
	return risk
}

// GetScopedRisk returns a mock Risk belonging to the input scope.
func GetScopedRisk(id string, clusterID string, namespace string) *storage.Risk {
	rs := &storage.RiskSubject{}
	rs.SetId(id)
	rs.SetNamespace(namespace)
	rs.SetClusterId(clusterID)
	rs.SetType(storage.RiskSubjectType_DEPLOYMENT)
	risk := &storage.Risk{}
	risk.SetId(id)
	risk.SetSubject(rs)
	return risk
}
