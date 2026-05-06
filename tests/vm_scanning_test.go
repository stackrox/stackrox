//go:build test_e2e || test_e2e_vm

package tests

import (
	"context"
	"testing"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/tests/vmhelpers"
	"github.com/stretchr/testify/require"
)

func (s *VMScanningSuite) TestScanPipeline() {
	for i := range s.persistentVMs {
		vm := &s.persistentVMs[i]
		s.T().Run(vm.Name, func(t *testing.T) {
			var result *vmhelpers.RoxagentRunResult
			var first *v2.VirtualMachine

			t.Run("RunRoxagent", func(t *testing.T) {
				t.Logf("running roxagent: sudo env ROXAGENT_REPO2CPE_URL=%s %s --verbose",
					s.cfg.Repo2CPEPrimaryURL, vmhelpers.DefaultRoxagentInstallPath)
				var err error
				result, err = s.ensureCanonicalScan(s.ctx, vm)
				require.NoError(t, err)
				require.NotNil(t, result)
				if result.UsedFallback {
					t.Logf("repo2cpe fallback was used")
				}
			})
			if result == nil {
				return
			}

			t.Run("WaitForScan", func(t *testing.T) {
				var err error
				first, err = s.waitForScan(s.ctx, vm)
				require.NoError(t, err)
				vm.ID = first.GetId()
			})
			if first == nil {
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

			beforeTime := s.mustGetScanTimestamp(first.GetId())
			var rescan *v2.VirtualMachine

			t.Run("Rescan", func(t *testing.T) {
				_, err := s.ensureCanonicalScan(s.ctx, vm)
				require.NoError(t, err)

				waitCtx, cancel := context.WithTimeout(s.ctx, s.cfg.ScanTimeout)
				defer cancel()
				rescan, err = vmhelpers.WaitForScanTimestampAfter(
					waitCtx, s.vmClient,
					vmhelpers.WaitOptions{
						Timeout:      s.cfg.ScanTimeout,
						PollInterval: s.cfg.ScanPollInterval,
						Logf:         s.logf,
					},
					vm.ID, beforeTime.AsTime(),
				)
				require.NoError(t, err, "rescan should produce a newer scan_time than %v", beforeTime.AsTime())
			})
			if rescan == nil {
				return
			}

			t.Run("ConsistencyCheck", func(t *testing.T) {
				fetched := s.mustGetVM(rescan.GetId())
				require.Equal(t, first.GetId(), fetched.GetId(),
					"VM ID should remain stable across rescans")
			})
		})
	}
}
