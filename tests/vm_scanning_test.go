//go:build test_e2e_vm

package tests

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/tests/logmatchers"
	"github.com/stackrox/rox/tests/vmhelpers"
	"github.com/stretchr/testify/require"
)

func (s *VMScanningSuite) TestScanPipeline() {
	for i := range s.vms {
		vm := &s.vms[i]
		if vm.SkipReason != "" {
			s.T().Run(vm.Name, func(t *testing.T) {
				t.Skip(vm.SkipReason)
			})
			continue
		}
		virt := s.virtctlForVM(*vm)

		if err := vmhelpers.EnsureVsockReady(s.ctx, virt, vm.Namespace, vm.Name, "scan pipeline"); err != nil {
			s.T().Fatalf("VSOCK unavailable on %s/%s — cluster does not support vsock, cannot continue: %v",
				vm.Namespace, vm.Name, err)
		}

		s.T().Run(vm.Name, func(t *testing.T) {
			var snapshot *centralScanSnapshot
			roxagentOK := false

			t.Run("EnsureRoxagentServing", func(t *testing.T) {
				t.Logf("ensuring Quadlet roxagent.service is active (image=%s rescan=%s)",
					s.cfg.RoxagentImage, vmhelpers.E2ERescanInterval)
				err := s.ensureRoxagentServing(s.ctx, vm)
				require.NoError(t, err)
				roxagentOK = true
			})
			if !roxagentOK {
				t.Log("skipping remaining subtests: roxagent serve failed to become ready")
				return
			}

			t.Run("WaitForSensorPushedMapping", func(t *testing.T) {
				s.waitForSensorPushedMapping(vm)
			})

			t.Run("WaitForScan", func(t *testing.T) {
				var err error
				snapshot, err = s.waitForScan(s.ctx, vm)
				require.NoError(t, err)
				require.NotEmpty(t, snapshot.ID)
				vm.ID = snapshot.ID
			})
			if snapshot == nil {
				t.Log("skipping remaining subtests: scan did not appear in Central")
				return
			}

			t.Run("CentralVMMetadata", func(t *testing.T) {
				s.skipUnlessLegacyVMAPI(t)
				first := snapshot.Legacy
				require.NotNil(t, first, "WaitForScan returned no VirtualMachine")
				listed := s.mustListVMByNamespaceAndName(vm.Namespace, vm.Name)
				require.Equal(t, listed.GetId(), first.GetId())
				require.Equal(t, vm.Name, first.GetName())
				require.Equal(t, vm.Namespace, first.GetNamespace())
				require.NotEmpty(t, first.GetClusterId())
				require.NotEmpty(t, first.GetClusterName())
				require.Equal(t, v2.VirtualMachine_RUNNING, first.GetState())
				require.NotNil(t, first.GetScan())
				require.NotNil(t, first.GetScan().GetScanTime())
			})

			t.Run("CentralScanComponents", func(t *testing.T) {
				s.skipUnlessLegacyVMAPI(t)
				first := snapshot.Legacy
				require.NotNil(t, first)
				for _, component := range first.GetScan().GetComponents() {
					require.NotContains(t, component.GetNotes(), v2.ScanComponent_UNSCANNED)
				}
				require.NotEmpty(t, first.GetScan().GetComponents())
			})

			t.Run("CentralScanOperatingSystem", func(t *testing.T) {
				s.skipUnlessLegacyVMAPI(t)
				os := snapshot.Legacy.GetScan().GetOperatingSystem()
				require.NotEmpty(t, os,
					"scan.operating_system should be populated via Sensor DiscoveredData")
			})

			t.Run("ConsistencyCheck", func(t *testing.T) {
				s.skipUnlessLegacyVMAPI(t)
				fetched := s.mustGetVM(snapshot.ID)
				require.Equal(t, snapshot.ID, fetched.GetId(),
					"VM ID should remain stable after pull-mode scan")
			})

			t.Run("VirtualMachineV2GetVM", func(t *testing.T) {
				s.skipUnlessV2VMAPI(t)
				detail := s.mustGetVMV2(snapshot.ID)
				require.Equal(t, snapshot.ID, detail.GetId())
				require.Equal(t, vm.Name, detail.GetName())
				require.Equal(t, vm.Namespace, detail.GetNamespace())
				require.NotEmpty(t, detail.GetClusterId())
				require.NotEmpty(t, detail.GetClusterName())
				require.Equal(t, v2.VirtualMachineV2State_VM_STATE_RUNNING, detail.GetState())
				require.NotNil(t, detail.GetLatestScan())
				require.NotNil(t, detail.GetLatestScan().GetScanTime())
			})

			t.Run("VirtualMachineV2ListVMs", func(t *testing.T) {
				s.skipUnlessV2VMAPI(t)
				listed := s.mustListV2VMByNamespaceAndName(vm.Namespace, vm.Name)
				require.Equal(t, snapshot.ID, listed.GetId())
				require.Equal(t, vm.Name, listed.GetName())
				require.Equal(t, vm.Namespace, listed.GetNamespace())
				require.NotEmpty(t, listed.GetClusterId())
				require.NotEmpty(t, listed.GetClusterName())
				require.Equal(t, v2.VirtualMachineV2State_VM_STATE_RUNNING, listed.GetState())

				cves, total, err := vmhelpers.ListAllVMCVEsByVM(s.ctx, s.vmV2Client, snapshot.ID)
				require.NoError(t, err)
				require.Greater(t, total, int32(0), "scanned RHEL guest should report CVEs via ListVMCVEsByVM")
				require.Equal(t, int(total), len(cves), "paginated CVE rows should cover total_count")
				for _, row := range cves {
					require.NotEmpty(t, row.GetCve())
				}

				distinct := distinctCVEIDs(cves)
				require.Equal(t, int32(len(distinct)), vmhelpers.VulnCountBySeverityTotal(listed.GetCveSeverityCounts()),
					"ListVMs.cveSeverityCounts totals must match distinct CVEs from ListVMCVEsByVM")

				summary, err := s.vmV2Client.GetVMVulnSummary(s.ctx, &v2.GetVMVulnSummaryRequest{Id: snapshot.ID})
				require.NoError(t, err)
				require.Equal(t, int32(len(distinct)), vmhelpers.VulnCountBySeverityTotal(summary.GetSeverityCounts()),
					"GetVMVulnSummary severity totals must match distinct CVEs from ListVMCVEsByVM")
			})

			t.Run("VirtualMachineV2ListVMCVEsByVM", func(t *testing.T) {
				s.skipUnlessV2VMAPI(t)
				cves, total, err := vmhelpers.ListAllVMCVEsByVM(s.ctx, s.vmV2Client, snapshot.ID)
				require.NoError(t, err)
				require.Greater(t, total, int32(0))
				require.NotEmpty(t, cves)
				for _, row := range cves {
					require.NotEmpty(t, row.GetCve())
				}
				require.Equal(t, len(cves), len(distinctCVEIDs(cves)), "ListVMCVEsByVM CVE IDs must be unique")

				var found bool
				for _, row := range cves {
					comps, err := s.vmV2Client.GetVMCVEComponents(s.ctx, &v2.GetVMCVEComponentsRequest{
						VmId:  snapshot.ID,
						CveId: row.GetCve(),
					})
					require.NoError(t, err)
					if len(comps.GetComponents()) == 0 {
						continue
					}
					require.GreaterOrEqual(t, row.GetAffectedComponentCount(), int32(1),
						"CVE %q has %d components from GetVMCVEComponents but affected_component_count=%d",
						row.GetCve(), len(comps.GetComponents()), row.GetAffectedComponentCount())
					found = true
					break
				}
				require.True(t, found, "at least one ListVMCVEsByVM row should have components from GetVMCVEComponents")
			})

			t.Run("VirtualMachineV2ListVMComponents", func(t *testing.T) {
				s.skipUnlessV2VMAPI(t)
				comps, total, err := vmhelpers.ListAllVMComponents(s.ctx, s.vmV2Client, snapshot.ID)
				require.NoError(t, err)
				require.Greater(t, total, int32(0), "completed scan should list components")
				require.NotEmpty(t, comps)
				require.Equal(t, int(total), len(comps), "paginated component rows should cover total_count")
				for _, c := range comps {
					require.NotEqual(t, v2.ScanStatus_NOT_SCANNED, c.GetScanStatus())
					require.NotEqual(t, v2.ScanStatus_SCAN_PENDING, c.GetScanStatus())
				}
			})

			// Regression test for ROX-36273: after a change to the RPM database (package removal), the agent should
			// detect the change on a later periodic rescan without being restarted.
			t.Run("Changes to RPM DB are detected by periodic rescan", func(t *testing.T) {
				removed := vmhelpers.VMImageProbePackage
				baselineCount := s.requireProbePackagePresent(t, snapshot, removed)

				beforeInvocationID, err := vmhelpers.RoxagentServeInvocationID(s.ctx, virt, vm.Namespace, vm.Name)
				require.NoError(t, err)

				require.NoError(t, vmhelpers.RemoveGuestRPMPackage(s.ctx, virt, vm.Namespace, vm.Name, removed))
				// Two Central scan_time advances past the baseline: the first
				// post-removal agent cycle can still race the erase, and each
				// agent cycle also needs a Sensor scrape (1m in VM e2e) before
				// Central moves scan_time.
				const minScanAdvances = 2
				waitTimeout := max(s.cfg.ScanTimeout, 2*vmhelpers.E2ERescanInterval+2*vmhelpers.E2EScraperPollInterval+3*time.Minute)
				t.Logf("removed package %q; waiting for %d scan_time advances (rescan=%s scraper=%s timeout=%s; baseline components=%d)",
					removed, minScanAdvances, vmhelpers.E2ERescanInterval, vmhelpers.E2EScraperPollInterval, waitTimeout,
					baselineCount)

				opts := vmhelpers.WaitOptions{
					Timeout:      waitTimeout,
					PollInterval: s.cfg.ScanPollInterval,
					Logf:         s.logf,
				}
				if s.enhancedVMModel {
					require.NotNil(t, snapshot.Detail.GetLatestScan().GetScanTime())
					baselineScanTime := snapshot.Detail.GetLatestScan().GetScanTime().AsTime()
					updated, err := vmhelpers.WaitForV2ScanMissingComponent(
						s.ctx, s.vmV2Client, opts, snapshot.ID, removed, baselineScanTime, minScanAdvances)
					require.NoError(t, err)
					require.Equal(t, baselineCount-1, len(updated),
						"exactly one component should disappear after removing %q", removed)
				} else {
					require.NotNil(t, snapshot.Legacy.GetScan().GetScanTime())
					baselineScanTime := snapshot.Legacy.GetScan().GetScanTime().AsTime()
					updated, err := vmhelpers.WaitForScanMissingComponent(
						s.ctx, s.vmClient, opts, snapshot.ID, removed, baselineScanTime, minScanAdvances)
					require.NoError(t, err)
					require.Equal(t, baselineCount-1, len(updated.GetScan().GetComponents()),
						"exactly one component should disappear after removing %q", removed)
				}

				require.NoError(t, vmhelpers.RoxagentServeDidNotRestart(
					s.ctx, virt, vm.Namespace, vm.Name, beforeInvocationID),
					"roxagent should not restart while waiting for the periodic rescan")
			})
		})
	}
}

// waitForSensorPushedMapping waits until Sensor logs a successful repo-to-CPE
// mapping push for vm. Setup installs without --repo-cpe-url, so a scan cannot
// complete until this push happens.
func (s *VMScanningSuite) waitForSensorPushedMapping(vm *VMHandle) {
	waitCtx, cancel := context.WithTimeout(s.ctx, s.cfg.ScanTimeout)
	defer cancel()
	re := regexp.MustCompile(regexp.QuoteMeta(
		fmt.Sprintf(`VMScraper: synced repo-to-CPE mapping to "%s/%s"`, vm.Namespace, vm.Name)))
	s.waitUntilLog(waitCtx, namespaces.StackRox, sensorPodLabels, sensorContainer,
		"contain Sensor-pushed repo-to-CPE mapping sync",
		logmatchers.ContainsLineMatching(re))
}

func (s *VMScanningSuite) requireProbePackagePresent(t *testing.T, snapshot *centralScanSnapshot, pkg string) int {
	t.Helper()
	if s.enhancedVMModel {
		comps, _, err := vmhelpers.ListAllVMComponents(s.ctx, s.vmV2Client, snapshot.ID)
		require.NoError(t, err)
		require.Contains(t, componentRowNames(comps), pkg,
			"baseline v2 component list must include %q from the VM image build", pkg)
		return len(comps)
	}
	require.NotNil(t, snapshot.Legacy)
	require.Contains(t, scanComponentNames(snapshot.Legacy), pkg,
		"baseline scan must include %q from the VM image build", pkg)
	return len(snapshot.Legacy.GetScan().GetComponents())
}

func scanComponentNames(vm *v2.VirtualMachine) []string {
	comps := vm.GetScan().GetComponents()
	names := make([]string, 0, len(comps))
	for _, c := range comps {
		if name := c.GetName(); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func componentRowNames(comps []*v2.VMComponentRow) []string {
	names := make([]string, 0, len(comps))
	for _, c := range comps {
		if name := c.GetName(); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func distinctCVEIDs(cves []*v2.VMCVERow) []string {
	seen := make(map[string]struct{}, len(cves))
	ids := make([]string, 0, len(cves))
	for _, row := range cves {
		id := row.GetCve()
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}
