//go:build compliance

package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func hasProfile(profiles []*v2.ComplianceProfile, name string) bool {
	for _, p := range profiles {
		if p.GetName() == name {
			return true
		}
	}
	return false
}

// TestComplianceV2MultiClusterScanFanout exercises a scenario that has no
// automated coverage today: StackRox's compliance integration across MORE
// THAN ONE secured cluster. Prow only ever runs a single OpenShift cluster
// per job, so there is no existing single-cluster/kind test (and no Prow
// job) that can create one ComplianceScanConfiguration targeting several
// clusters and verify it fans out correctly, with real per-cluster results
// attributed back to the right cluster. This test fills that gap using the
// multi-cluster kind harness in tests/e2e/kind-compliance
// (setup-multicluster-compliance-ci.sh), which stands up Central plus N
// secured clusters, each running a real Compliance Operator at full
// OpenShift fidelity (real ocp4/rhcos4 ProfileBundles + the OpenShift-API
// shim, so ocp4-cis checks produce real PASS/FAIL instead of
// NOT-APPLICABLE).
//
// It proves, end to end, that:
//  1. Central tracks a distinct, HEALTHY Compliance Operator integration for
//     every secured cluster (not just one, and none shared/duplicated).
//  2. A SINGLE scan configuration targeting all clusters fans out its status
//     to every one of them.
//  3. Real OVAL-evaluated PASS and FAIL results round-trip from each cluster
//     and are attributed to the correct cluster in Central.
func TestComplianceV2MultiClusterScanFanout(t *testing.T) {
	const realProfile = "ocp4-cis" // shipped by the real ocp4 ProfileBundle
	ctx := context.Background()

	conn := centralgrpc.GRPCConnectionToCentral(t)
	integrationSvc := v2.NewComplianceIntegrationServiceClient(conn)
	profileSvc := v2.NewComplianceProfileServiceClient(conn)
	scanCfgSvc := v2.NewComplianceScanConfigurationServiceClient(conn)
	resultsSvc := v2.NewComplianceResultsServiceClient(conn)

	// 1. Central sees one HEALTHY Compliance Operator integration per secured
	// cluster, and no two integrations share a cluster id.
	integrations, err := integrationSvc.ListComplianceIntegrations(ctx, &v2.RawQuery{Query: ""})
	require.NoError(t, err)
	require.GreaterOrEqualf(t, len(integrations.GetIntegrations()), 2,
		"multi-cluster scan fan-out needs >=2 compliance integrations, found %d", len(integrations.GetIntegrations()))

	seen := make(map[string]bool, len(integrations.GetIntegrations()))
	var clusterIDs, clusterNames []string
	for _, in := range integrations.GetIntegrations() {
		assert.NotEmpty(t, in.GetClusterId(), "integration missing cluster id")
		assert.NotEmpty(t, in.GetClusterName(), "integration missing cluster name")
		assert.Equalf(t, v2.COStatus_HEALTHY, in.GetStatus(),
			"CO integration for cluster %q is not healthy: %v", in.GetClusterName(), in.GetStatusErrors())
		assert.Falsef(t, seen[in.GetClusterId()], "duplicate cluster id %s across integrations", in.GetClusterId())
		seen[in.GetClusterId()] = true

		// Only target clusters that expose the real profile (also asserts the
		// ocp4 ProfileBundle synced per-cluster).
		profiles, err := profileSvc.ListComplianceProfiles(ctx, &v2.ProfilesForClusterRequest{ClusterId: in.GetClusterId()})
		require.NoError(t, err)
		if hasProfile(profiles.GetProfiles(), realProfile) {
			clusterIDs = append(clusterIDs, in.GetClusterId())
			clusterNames = append(clusterNames, in.GetClusterName())
		}
	}
	require.GreaterOrEqualf(t, len(clusterIDs), 2,
		"need >=2 clusters exposing profile %q, found %d", realProfile, len(clusterIDs))
	t.Logf("targeting %d clusters with profile %q: %v", len(clusterIDs), realProfile, clusterNames)

	// 2. One scan configuration targeting every cluster.
	scanName := fmt.Sprintf("mc-fanout-%s", uuid.NewV4().String())
	req := &v2.ComplianceScanConfiguration{
		ScanName: scanName,
		Clusters: clusterIDs,
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan:  false,
			Profiles:     []string{realProfile},
			Description:  "multi-cluster ocp4-cis scan fan-out",
			ScanSchedule: initialSchedule,
		},
	}
	created, err := scanCfgSvc.CreateComplianceScanConfiguration(ctx, req)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = deleteScanConfig(context.Background(), created.GetId(), scanCfgSvc)
	})

	// The configuration should fan out to every target cluster.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		cfgs, err := scanCfgSvc.ListComplianceScanConfigurations(ctx, &v2.RawQuery{Query: ""})
		require.NoError(c, err)
		var ours *v2.ComplianceScanConfigurationStatus
		for _, cfg := range cfgs.GetConfigurations() {
			if cfg.GetScanName() == scanName {
				ours = cfg
				break
			}
		}
		require.NotNil(c, ours, "scan config %s not found", scanName)
		statusClusters := make(map[string]bool)
		for _, cs := range ours.GetClusterStatus() {
			statusClusters[cs.GetClusterId()] = true
		}
		for i, cid := range clusterIDs {
			assert.Truef(c, statusClusters[cid], "scan config status missing cluster %q (%s)", clusterNames[i], cid)
		}
	}, 2*time.Minute, 10*time.Second)

	// 3. Each target cluster should produce real results (at least one PASS
	// and one FAIL from the real ocp4-cis OVAL evaluation, not
	// NOT-APPLICABLE), attributed to that cluster in Central. ocp4-cis
	// platform checks are real OVAL evaluations against the OpenShift-API
	// shim objects, so give it a generous window.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		results, err := resultsSvc.GetComplianceScanResults(ctx, &v2.RawQuery{Query: ""})
		require.NoError(c, err)
		byCluster := make(map[string]map[v2.ComplianceCheckStatus]bool)
		for _, r := range results.GetScanResults() {
			if r.GetScanName() != scanName {
				continue
			}
			cid := r.GetClusterId()
			if byCluster[cid] == nil {
				byCluster[cid] = make(map[v2.ComplianceCheckStatus]bool)
			}
			byCluster[cid][r.GetResult().GetStatus()] = true
		}
		for i, cid := range clusterIDs {
			require.Containsf(c, byCluster, cid, "no results yet for cluster %q (%s)", clusterNames[i], cid)
			assert.Truef(c, byCluster[cid][v2.ComplianceCheckStatus_PASS], "cluster %q missing a real PASS result (ocp4-cis would be all N/A without the OpenShift-API shim)", clusterNames[i])
			assert.Truef(c, byCluster[cid][v2.ComplianceCheckStatus_FAIL], "cluster %q missing a real FAIL result (ocp4-cis would be all N/A without the OpenShift-API shim)", clusterNames[i])
		}
	}, 15*time.Minute, 30*time.Second)
}
