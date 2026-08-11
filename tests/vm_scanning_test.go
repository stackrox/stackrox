//go:build test_e2e_vm

package tests

import (
	"testing"
	"time"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/tests/vmhelpers"
	"github.com/stretchr/testify/require"
)

func (s *VMScanningSuite) TestScanPipeline() {
	for i := range s.vms {
		vm := &s.vms[i]
		virt := s.virtctlForVM(*vm)

		if err := vmhelpers.EnsureVsockReady(s.ctx, virt, vm.Namespace, vm.Name, "scan pipeline"); err != nil {
			s.T().Fatalf("VSOCK unavailable on %s/%s — cluster does not support vsock, cannot continue: %v",
				vm.Namespace, vm.Name, err)
		}

		s.T().Run(vm.Name, func(t *testing.T) {
			var first *v2.VirtualMachine
			roxagentOK := false

			t.Run("EnsureRoxagentServing", func(t *testing.T) {
				t.Logf("ensuring pull-mode agent: sudo %s serve --port 818 --host-path / --rescan-interval 5m --repo-cpe-url %s",
					vmhelpers.DefaultRoxagentInstallPath, s.cfg.Repo2CPEURL)
				err := s.ensureRoxagentServing(s.ctx, vm)
				require.NoError(t, err)
				roxagentOK = true
			})
			if !roxagentOK {
				t.Log("skipping remaining subtests: roxagent serve failed to become ready")
				return
			}

			t.Run("WaitForScan", func(t *testing.T) {
				var err error
				first, err = s.waitForScan(s.ctx, vm)
				require.NoError(t, err)
				vm.ID = first.GetId()
			})
			if first == nil {
				t.Log("skipping remaining subtests: scan did not appear in Central")
				return
			}

			t.Run("CentralVMMetadata", func(t *testing.T) {
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
				for _, component := range first.GetScan().GetComponents() {
					require.NotContains(t, component.GetNotes(), v2.ScanComponent_UNSCANNED)
				}
				require.NotEmpty(t, first.GetScan().GetComponents())
			})

			t.Run("CentralScanOperatingSystem", func(t *testing.T) {
				os := first.GetScan().GetOperatingSystem()
				require.NotEmpty(t, os,
					"scan.operating_system should be populated via Sensor DiscoveredData")
			})

			t.Run("ConsistencyCheck", func(t *testing.T) {
				fetched := s.mustGetVM(first.GetId())
				require.Equal(t, first.GetId(), fetched.GetId(),
					"VM ID should remain stable after pull-mode scan")
			})

			// Regression test for ROX-36273: after a change to the RPM database (package removal), the agent should
			// detect the change on a later periodic rescan without being restarted.
			t.Run("Changes to RPM DB are detected by periodic rescan", func(t *testing.T) {
				require.NotNil(t, first.GetScan().GetScanTime())
				baselineScanTime := first.GetScan().GetScanTime().AsTime()
				baselineCount := len(first.GetScan().GetComponents())
				// bc is installed by stackrox/vm-images into every VM
				// container-disk so this test has a known removable probe package.
				removed := vmhelpers.VMImageProbePackage
				require.Contains(t, scanComponentNames(first), removed,
					"baseline scan must include %q from the VM image build", removed)

				beforeInvocationID, err := vmhelpers.RoxagentServeInvocationID(s.ctx, virt, vm.Namespace, vm.Name)
				require.NoError(t, err)

				require.NoError(t, vmhelpers.RemoveGuestRPMPackage(s.ctx, virt, vm.Namespace, vm.Name, removed))
				// Two Central scan_time advances past the baseline: the first
				// post-removal agent cycle can still race the erase, and each
				// agent cycle also needs a Sensor scrape (1m in VM e2e) before
				// Central moves scan_time.
				const minScanAdvances = 2
				waitTimeout := max(s.cfg.ScanTimeout, 2*vmhelpers.E2ERescanInterval+2*vmhelpers.E2EScraperPollInterval+3*time.Minute)
				t.Logf("removed package %q; waiting for %d scan_time advances (rescan=%s scraper=%s timeout=%s; baseline components=%d scan_time=%s)",
					removed, minScanAdvances, vmhelpers.E2ERescanInterval, vmhelpers.E2EScraperPollInterval, waitTimeout,
					baselineCount, baselineScanTime.UTC().Format(time.RFC3339))

				updated, err := vmhelpers.WaitForScanMissingComponent(
					s.ctx, s.vmClient, vmhelpers.WaitOptions{
						Timeout:      waitTimeout,
						PollInterval: s.cfg.ScanPollInterval,
						Logf:         s.logf,
					}, first.GetId(), removed, baselineScanTime, minScanAdvances)
				require.NoError(t, err)
				require.Equal(t, baselineCount-1, len(updated.GetScan().GetComponents()),
					"exactly one component should disappear after removing %q", removed)
				require.NotContains(t, scanComponentNames(updated), removed,
					"removed package %q must be absent from the rescanned inventory", removed)

				require.NoError(t, vmhelpers.RoxagentServeDidNotRestart(
					s.ctx, virt, vm.Namespace, vm.Name, beforeInvocationID),
					"roxagent should not restart while waiting for the periodic rescan")
			})
		})
	}
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
