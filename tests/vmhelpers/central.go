package vmhelpers

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/search"
)

// ScanReadiness configures which scan fields must be present before
// WaitForScanReady considers the scan complete.
type ScanReadiness struct {
	Components bool // at least one component reported
	AllScanned bool // no UNSCANNED notes on any component
}

// WaitForVMPresentInCentral polls ListVirtualMachines until a VM matches namespace and name.
func WaitForVMPresentInCentral(ctx context.Context, client v2.VirtualMachineServiceClient, opts WaitOptions, namespace, name string) (*v2.VirtualMachine, error) {
	var found *v2.VirtualMachine
	err := pollUntil(ctx, opts, fmt.Sprintf("VM present in Central (namespace=%q name=%q)", namespace, name), func(ctx context.Context) (bool, string, error) {
		vm, err := ListVMByNamespaceName(ctx, client, namespace, name)
		if err != nil {
			return false, "", err
		}
		if vm == nil {
			return false, "list returned no matching virtual machine", nil
		}
		found = vm
		return true, fmt.Sprintf("id=%s", vm.GetId()), nil
	})
	if err != nil {
		return nil, err
	}
	return found, nil
}

// vmConditionCheck inspects a freshly-fetched VirtualMachine and reports whether
// the desired condition is met. detail is included in poll logs and timeout errors.
type vmConditionCheck func(vm *v2.VirtualMachine) (done bool, detail string)

// waitForVMCondition polls GetVirtualMachine until check returns done==true.
func waitForVMCondition(ctx context.Context, client v2.VirtualMachineServiceClient, opts WaitOptions, id, desc string, check vmConditionCheck) (*v2.VirtualMachine, error) {
	var vm *v2.VirtualMachine
	err := pollUntil(ctx, opts, desc, func(ctx context.Context) (bool, string, error) {
		cur, err := client.GetVirtualMachine(ctx, &v2.GetVirtualMachineRequest{Id: id})
		if err != nil {
			return false, "", err
		}
		done, detail := check(cur)
		if done {
			vm = cur
		}
		return done, detail, nil
	})
	if err != nil {
		return nil, err
	}
	return vm, nil
}

// WaitForVMIdentityFields polls GetVirtualMachine until id maps to the expected namespace and name.
func WaitForVMIdentityFields(ctx context.Context, client v2.VirtualMachineServiceClient, opts WaitOptions, id, expectedNamespace, expectedName string) (*v2.VirtualMachine, error) {
	return waitForVMCondition(ctx, client, opts, id, fmt.Sprintf("VM identity fields (id=%q)", id), func(vm *v2.VirtualMachine) (bool, string) {
		detail := fmt.Sprintf("namespace=%q name=%q", vm.GetNamespace(), vm.GetName())
		return vm.GetNamespace() == expectedNamespace && vm.GetName() == expectedName, detail
	})
}

// WaitForVMRunningInCentral polls until the VM state is RUNNING.
func WaitForVMRunningInCentral(ctx context.Context, client v2.VirtualMachineServiceClient, opts WaitOptions, id string) (*v2.VirtualMachine, error) {
	return waitForVMCondition(ctx, client, opts, id, fmt.Sprintf("VM RUNNING (id=%q)", id), func(vm *v2.VirtualMachine) (bool, string) {
		st := vm.GetState()
		return st == v2.VirtualMachine_RUNNING, fmt.Sprintf("state=%s", st)
	})
}

// WaitForVMScanNonNil polls until VirtualMachine.scan is non-nil.
func WaitForVMScanNonNil(ctx context.Context, client v2.VirtualMachineServiceClient, opts WaitOptions, id string) (*v2.VirtualMachine, error) {
	return waitForVMCondition(ctx, client, opts, id, fmt.Sprintf("VM scan non-nil (id=%q)", id), func(vm *v2.VirtualMachine) (bool, string) {
		if vm.GetScan() == nil {
			return false, "scan is nil"
		}
		return true, "scan present"
	})
}

// WaitForVMScanTimestamp polls until scan_time is set.
func WaitForVMScanTimestamp(ctx context.Context, client v2.VirtualMachineServiceClient, opts WaitOptions, id string) (*v2.VirtualMachine, error) {
	return waitForVMCondition(ctx, client, opts, id, fmt.Sprintf("VM scan timestamp (id=%q)", id), func(vm *v2.VirtualMachine) (bool, string) {
		sc := vm.GetScan()
		if sc == nil {
			return false, "scan is nil"
		}
		if sc.GetScanTime() == nil {
			return false, "scan_time is nil"
		}
		return true, "scan_time set"
	})
}

// WaitForScanTimestampAfter polls until the VM's scan_time is strictly after the given
// threshold. Use this to wait for a rescan to be reflected in Central.
func WaitForScanTimestampAfter(ctx context.Context, client v2.VirtualMachineServiceClient, opts WaitOptions, id string, after time.Time) (*v2.VirtualMachine, error) {
	return waitForVMCondition(ctx, client, opts, id, fmt.Sprintf("scan timestamp after %v (id=%q)", after.Format(time.RFC3339), id), func(vm *v2.VirtualMachine) (bool, string) {
		sc := vm.GetScan()
		if sc == nil || sc.GetScanTime() == nil {
			return false, "scan_time not set yet"
		}
		ts := sc.GetScanTime().AsTime()
		if !ts.After(after) {
			return false, fmt.Sprintf("scan_time=%v not yet after %v", ts.Format(time.RFC3339Nano), after.Format(time.RFC3339Nano))
		}
		return true, fmt.Sprintf("scan_time=%v", ts.Format(time.RFC3339Nano))
	})
}

// WaitForVMComponentsReported polls until at least one scan component exists.
func WaitForVMComponentsReported(ctx context.Context, client v2.VirtualMachineServiceClient, opts WaitOptions, id string) (*v2.VirtualMachine, error) {
	return waitForVMCondition(ctx, client, opts, id, fmt.Sprintf("VM components reported (id=%q)", id), func(vm *v2.VirtualMachine) (bool, string) {
		if hasReportedComponents(vm) {
			return true, "components present"
		}
		n := 0
		if vm.GetScan() != nil {
			n = len(vm.GetScan().GetComponents())
		}
		return false, fmt.Sprintf("component count=%d", n)
	})
}

// WaitForAllVMComponentsScanned polls until every scan component lacks the UNSCANNED note.
func WaitForAllVMComponentsScanned(ctx context.Context, client v2.VirtualMachineServiceClient, opts WaitOptions, id string) (*v2.VirtualMachine, error) {
	return waitForVMCondition(ctx, client, opts, id, fmt.Sprintf("all VM components scanned (id=%q)", id), func(vm *v2.VirtualMachine) (bool, string) {
		if allComponentsScanned(vm) {
			return true, "all components scanned"
		}
		return false, "pending UNSCANNED components or empty component list"
	})
}

// WaitForScanReady polls GetVirtualMachine in a single loop until every field
// requested in conds is populated. Each poll iteration logs which conditions
// are already met and which are still pending, so partial progress is visible.
func WaitForScanReady(ctx context.Context, client v2.VirtualMachineServiceClient, opts WaitOptions, id string, conds ScanReadiness) (*v2.VirtualMachine, error) {
	return waitForVMCondition(ctx, client, opts, id, fmt.Sprintf("scan ready (id=%q)", id), func(vm *v2.VirtualMachine) (bool, string) {
		scan := vm.GetScan()
		if scan == nil {
			return false, "scan is nil"
		}

		var ready, pending []string
		comps := scan.GetComponents()

		if conds.Components {
			if n := len(comps); n > 0 {
				ready = append(ready, fmt.Sprintf("components=%d", n))
			} else {
				pending = append(pending, "components")
			}
		}
		if conds.AllScanned {
			if len(comps) > 0 && !slices.ContainsFunc(comps, func(c *v2.ScanComponent) bool {
				return slices.Contains(c.GetNotes(), v2.ScanComponent_UNSCANNED)
			}) {
				ready = append(ready, "all-scanned")
			} else {
				pending = append(pending, "all-scanned")
			}
		}

		// OS is informational: it arrives in the same message as components,
		// so if components are present and fully scanned, the OS won't appear
		// in a later update. Report it but never block on it.
		if os := scan.GetOperatingSystem(); os != "" {
			ready = append(ready, fmt.Sprintf("os=%q", os))
		} else if len(pending) == 0 {
			ready = append(ready, "os=<not reported>")
		}

		detail := fmt.Sprintf("ready:[%s] waiting:[%s]", strings.Join(ready, ","), strings.Join(pending, ","))
		return len(pending) == 0, detail
	})
}

// listVMPageSize is the page size used when iterating ListVirtualMachines pages.
const listVMPageSize = int32(1000)

// rawListQueryNamespaceAndName builds a Central raw search query matching namespace and VM name.
func rawListQueryNamespaceAndName(namespace, name string) string {
	return fmt.Sprintf("%s:%s+%s:%s", search.Namespace, namespace, search.VirtualMachineName, name)
}

// ListVMByNamespaceName returns the first VirtualMachine in Central whose namespace and name
// exactly match the given values, paging through all results. Returns (nil, nil) when no
// match is found.
func ListVMByNamespaceName(ctx context.Context, client v2.VirtualMachineServiceClient, namespace, name string) (*v2.VirtualMachine, error) {
	baseQuery := rawListQueryNamespaceAndName(namespace, name)
	for offset := int32(0); ; offset += listVMPageSize {
		resp, err := client.ListVirtualMachines(ctx, &v2.ListVirtualMachinesRequest{
			Query: &v2.RawQuery{
				Query: baseQuery,
				Pagination: &v2.Pagination{
					Limit:  listVMPageSize,
					Offset: offset,
				},
			},
		})
		if err != nil {
			return nil, err
		}
		for _, vm := range resp.GetVirtualMachines() {
			if vm.GetNamespace() == namespace && vm.GetName() == name {
				return vm, nil
			}
		}
		if int32(len(resp.GetVirtualMachines())) < listVMPageSize {
			return nil, nil
		}
	}
}

// hasReportedComponents reports whether the VM scan lists at least one component.
func hasReportedComponents(vm *v2.VirtualMachine) bool {
	if vm == nil || vm.GetScan() == nil {
		return false
	}
	return len(vm.GetScan().GetComponents()) > 0
}

// allComponentsScanned reports whether every scan component lacks the UNSCANNED note.
func allComponentsScanned(vm *v2.VirtualMachine) bool {
	if vm == nil || vm.GetScan() == nil {
		return false
	}
	comps := vm.GetScan().GetComponents()
	if len(comps) == 0 {
		return false
	}
	for _, c := range comps {
		if slices.Contains(c.GetNotes(), v2.ScanComponent_UNSCANNED) {
			return false
		}
	}
	return true
}
