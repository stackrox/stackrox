//go:build compliance

package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Multi-cluster compliance tests.
//
// These exercise StackRox's compliance integration across MORE THAN ONE secured
// cluster, which single-cluster tests cannot cover: per-cluster CO integration
// tracking, fan-out of one scan configuration to several clusters, and
// per-cluster attribution of results in Central.
//
// They are Central-API-only (no per-cluster kube access): they reference a CEL
// TailoredProfile that the environment is expected to have created on every
// secured cluster. On OpenShift/RHCOS, built-in profiles work; on kind the
// kind-compliance CI harness (tests/e2e/kind-compliance) creates a CEL profile
// named "kind-cel-tp" that yields real PASS/FAIL results on any Kubernetes.
// Override the profile name with COMPLIANCE_MC_PROFILE.
//
// Both tests skip (not fail) when fewer than two suitable clusters are present,
// so they are safe to include in single-cluster runs.

func multiClusterProfile() string {
	if p := os.Getenv("COMPLIANCE_MC_PROFILE"); p != "" {
		return p
	}
	return "kind-cel-tp"
}

func hasProfile(profiles []*v2.ComplianceProfile, name string) bool {
	for _, p := range profiles {
		if p.GetName() == name {
			return true
		}
	}
	return false
}

// TestComplianceV2MultiClusterIntegrations verifies Central tracks a Compliance
// Operator integration independently for each secured cluster, and that they are
// distinct and healthy.
func TestComplianceV2MultiClusterIntegrations(t *testing.T) {
	ctx := context.Background()
	conn := centralgrpc.GRPCConnectionToCentral(t)
	integrationSvc := v2.NewComplianceIntegrationServiceClient(conn)

	resp, err := integrationSvc.ListComplianceIntegrations(ctx, &v2.RawQuery{Query: ""})
	require.NoError(t, err)

	integrations := resp.GetIntegrations()
	if len(integrations) < 2 {
		t.Skipf("multi-cluster test needs >=2 compliance integrations, found %d", len(integrations))
	}

	seen := make(map[string]bool, len(integrations))
	for _, in := range integrations {
		assert.NotEmpty(t, in.GetClusterId(), "integration missing cluster id")
		assert.NotEmpty(t, in.GetClusterName(), "integration missing cluster name")
		assert.Equalf(t, v2.COStatus_HEALTHY, in.GetStatus(),
			"CO integration for cluster %q is not healthy: %v", in.GetClusterName(), in.GetStatusErrors())
		assert.Falsef(t, seen[in.GetClusterId()], "duplicate cluster id %s across integrations", in.GetClusterId())
		seen[in.GetClusterId()] = true
	}
	assert.GreaterOrEqual(t, len(seen), 2, "expected integrations for at least two distinct clusters")
}

// TestComplianceV2MultiClusterScanConfigResults creates a SINGLE scan
// configuration that targets every cluster carrying the shared CEL profile, then
// verifies the configuration fans out to all of them and that real PASS and FAIL
// results come back attributed to each cluster.
func TestComplianceV2MultiClusterScanConfigResults(t *testing.T) {
	ctx := context.Background()
	profileName := multiClusterProfile()

	conn := centralgrpc.GRPCConnectionToCentral(t)
	integrationSvc := v2.NewComplianceIntegrationServiceClient(conn)
	profileSvc := v2.NewComplianceProfileServiceClient(conn)
	scanCfgSvc := v2.NewComplianceScanConfigurationServiceClient(conn)
	resultsSvc := v2.NewComplianceResultsServiceClient(conn)

	integrations, err := integrationSvc.ListComplianceIntegrations(ctx, &v2.RawQuery{Query: ""})
	require.NoError(t, err)

	// Keep only clusters that expose the shared profile (so the scan can run and
	// produce results on each). This also asserts the profile synced per-cluster.
	var clusterIDs, clusterNames []string
	for _, in := range integrations.GetIntegrations() {
		cid := in.GetClusterId()
		profiles, err := profileSvc.ListComplianceProfiles(ctx, &v2.ProfilesForClusterRequest{ClusterId: cid})
		require.NoError(t, err)
		if hasProfile(profiles.GetProfiles(), profileName) {
			clusterIDs = append(clusterIDs, cid)
			clusterNames = append(clusterNames, in.GetClusterName())
		}
	}
	if len(clusterIDs) < 2 {
		t.Skipf("multi-cluster results test needs >=2 clusters exposing profile %q, found %d", profileName, len(clusterIDs))
	}
	t.Logf("targeting %d clusters with profile %q: %v", len(clusterIDs), profileName, clusterNames)

	scanName := fmt.Sprintf("mc-cel-%s", uuid.NewV4().String())
	req := &v2.ComplianceScanConfiguration{
		ScanName: scanName,
		Clusters: clusterIDs,
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan:  false,
			Profiles:     []string{profileName},
			Description:  "multi-cluster CEL scan (PoC)",
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

	// Each target cluster should produce real results (at least one PASS and one
	// FAIL from the CEL profile), attributed to that cluster in Central.
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
			assert.Truef(c, byCluster[cid][v2.ComplianceCheckStatus_PASS], "cluster %q missing a PASS result", clusterNames[i])
			assert.Truef(c, byCluster[cid][v2.ComplianceCheckStatus_FAIL], "cluster %q missing a FAIL result", clusterNames[i])
		}
	}, 10*time.Minute, 30*time.Second)
}

// TestComplianceV2MultiClusterRealProfiles verifies that, when the environment
// installs the real ocp4/rhcos4 ProfileBundles (the kind-compliance harness run
// with FIDELITY=content, or any real OpenShift cluster), the full profile+rule
// catalog syncs to Central independently for every secured cluster. It auto-skips
// when only the lean CEL profile is present (FIDELITY=cel), so it is safe in the
// default PR run.
func TestComplianceV2MultiClusterRealProfiles(t *testing.T) {
	const realProfile = "ocp4-cis" // shipped by the real ocp4 ProfileBundle
	ctx := context.Background()

	conn := centralgrpc.GRPCConnectionToCentral(t)
	integrationSvc := v2.NewComplianceIntegrationServiceClient(conn)
	profileSvc := v2.NewComplianceProfileServiceClient(conn)

	integrations, err := integrationSvc.ListComplianceIntegrations(ctx, &v2.RawQuery{Query: ""})
	require.NoError(t, err)
	if len(integrations.GetIntegrations()) < 2 {
		t.Skipf("multi-cluster test needs >=2 compliance integrations, found %d", len(integrations.GetIntegrations()))
	}

	// Content-fidelity detection: skip unless the real profile synced for cluster 1.
	first := integrations.GetIntegrations()[0]
	firstProfiles, err := profileSvc.ListComplianceProfiles(ctx, &v2.ProfilesForClusterRequest{ClusterId: first.GetClusterId()})
	require.NoError(t, err)
	if !hasProfile(firstProfiles.GetProfiles(), realProfile) {
		t.Skipf("real ProfileBundles not installed (no %q profile); run the harness with FIDELITY=content", realProfile)
	}

	for _, in := range integrations.GetIntegrations() {
		resp, err := profileSvc.ListComplianceProfiles(ctx, &v2.ProfilesForClusterRequest{ClusterId: in.GetClusterId()})
		require.NoError(t, err)
		profiles := resp.GetProfiles()

		assert.Truef(t, hasProfile(profiles, realProfile), "cluster %q missing real profile %q", in.GetClusterName(), realProfile)
		// The real ocp4+rhcos4 bundles bring dozens of profiles; assert the catalog
		// actually synced per cluster (not just our single CEL profile).
		assert.GreaterOrEqualf(t, len(profiles), 10, "cluster %q has only %d profiles; expected the full ocp4/rhcos4 catalog", in.GetClusterName(), len(profiles))

		var realP *v2.ComplianceProfile
		for _, p := range profiles {
			if p.GetName() == realProfile {
				realP = p
				break
			}
		}
		require.NotNilf(t, realP, "cluster %q: %q not found after presence check", in.GetClusterName(), realProfile)
		full, err := profileSvc.GetComplianceProfile(ctx, &v2.ResourceByID{Id: realP.GetId()})
		require.NoError(t, err)
		assert.Greaterf(t, len(full.GetRules()), 0, "profile %q on cluster %q has no rules synced", realProfile, in.GetClusterName())
		t.Logf("cluster %q: %d profiles synced, %q has %d rules", in.GetClusterName(), len(profiles), realProfile, len(full.GetRules()))
	}
}

// TestComplianceV2MultiClusterPlatformResults verifies that, when the
// environment installs the OpenShift-API shim (the kind-compliance harness run
// with FIDELITY=platform), the real ocp4-cis OVAL checks actually evaluate on
// kind: creating ONE scan configuration for profile "ocp4-cis" across every
// secured cluster must produce real PASS and FAIL results (not NOT-APPLICABLE
// everywhere, which is what ocp4-cis yields WITHOUT the shim). Gated on
// COMPLIANCE_MC_PLATFORM=true (set by the harness/workflow in platform mode)
// so it never runs unguarded: ocp4-cis without the shim is all N/A and would
// just time out waiting for a PASS/FAIL that never comes.
func TestComplianceV2MultiClusterPlatformResults(t *testing.T) {
	if os.Getenv("COMPLIANCE_MC_PLATFORM") != "true" {
		t.Skip("set COMPLIANCE_MC_PLATFORM=true to run (requires the kind-compliance harness FIDELITY=platform OpenShift-API shim)")
	}
	const realProfile = "ocp4-cis"
	ctx := context.Background()

	conn := centralgrpc.GRPCConnectionToCentral(t)
	integrationSvc := v2.NewComplianceIntegrationServiceClient(conn)
	profileSvc := v2.NewComplianceProfileServiceClient(conn)
	scanCfgSvc := v2.NewComplianceScanConfigurationServiceClient(conn)
	resultsSvc := v2.NewComplianceResultsServiceClient(conn)

	integrations, err := integrationSvc.ListComplianceIntegrations(ctx, &v2.RawQuery{Query: ""})
	require.NoError(t, err)

	var clusterIDs, clusterNames []string
	for _, in := range integrations.GetIntegrations() {
		cid := in.GetClusterId()
		profiles, err := profileSvc.ListComplianceProfiles(ctx, &v2.ProfilesForClusterRequest{ClusterId: cid})
		require.NoError(t, err)
		if hasProfile(profiles.GetProfiles(), realProfile) {
			clusterIDs = append(clusterIDs, cid)
			clusterNames = append(clusterNames, in.GetClusterName())
		}
	}
	require.NotEmptyf(t, clusterIDs, "no cluster exposes profile %q; run the harness with FIDELITY=platform", realProfile)
	t.Logf("targeting %d cluster(s) with profile %q: %v", len(clusterIDs), realProfile, clusterNames)

	scanName := fmt.Sprintf("mc-platform-%s", uuid.NewV4().String())
	req := &v2.ComplianceScanConfiguration{
		ScanName: scanName,
		Clusters: clusterIDs,
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan:  false,
			Profiles:     []string{realProfile},
			Description:  "multi-cluster ocp4-cis platform scan (OpenShift-API shim)",
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

	// ocp4-cis platform checks are real OVAL evaluations against the
	// OpenShift-API shim objects, so an actual scan+OVAL run takes longer than
	// the CEL in-process path; give it the same generous window the real-profile
	// ocp4/rhcos4 CEL-equivalent (CreateGetScanConfigurations) uses upstream.
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
