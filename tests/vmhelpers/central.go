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

// DefaultCentralWaitTimeout and DefaultCentralPollInterval are the fixed defaults for plan-signature
// wait helpers (no WaitOptions parameter). Tests and tight loops should use *WithOptions instead of
// mutating package state.
const (
	DefaultCentralWaitTimeout  = 30 * time.Minute
	DefaultCentralPollInterval = 5 * time.Second
)

// defaultCentralWaitOptions returns WaitOptions matching DefaultCentralWaitTimeout and DefaultCentralPollInterval.
func defaultCentralWaitOptions() WaitOptions {
	return WaitOptions{
		Timeout:      DefaultCentralWaitTimeout,
		PollInterval: DefaultCentralPollInterval,
	}
}

// WaitForVMPresentInCentral polls ListVirtualMachines until a VM matches namespace and name.
func WaitForVMPresentInCentral(ctx context.Context, client v2.VirtualMachineServiceClient, namespace, name string) (*v2.VirtualMachine, error) {
	return WaitForVMPresentInCentralWithOptions(ctx, client, defaultCentralWaitOptions(), namespace, name)
}

// WaitForVMPresentInCentralWithOptions is like WaitForVMPresentInCentral with explicit poll/timeouts.
func WaitForVMPresentInCentralWithOptions(ctx context.Context, client v2.VirtualMachineServiceClient, opts WaitOptions, namespace, name string) (*v2.VirtualMachine, error) {
	var found *v2.VirtualMachine
	err := pollUntil(ctx, opts, fmt.Sprintf("VM present in Central (namespace=%q name=%q)", namespace, name), func(ctx context.Context) (bool, string, error) {
		vm, err := listVirtualMachineByNamespaceName(ctx, client, namespace, name)
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

// WaitForVMIdentityFields polls GetVirtualMachine until id maps to the expected namespace and name.
func WaitForVMIdentityFields(ctx context.Context, client v2.VirtualMachineServiceClient, id, expectedNamespace, expectedName string) (*v2.VirtualMachine, error) {
	return WaitForVMIdentityFieldsWithOptions(ctx, client, defaultCentralWaitOptions(), id, expectedNamespace, expectedName)
}

// WaitForVMIdentityFieldsWithOptions is like WaitForVMIdentityFields with explicit poll/timeouts.
func WaitForVMIdentityFieldsWithOptions(ctx context.Context, client v2.VirtualMachineServiceClient, opts WaitOptions, id, expectedNamespace, expectedName string) (*v2.VirtualMachine, error) {
	var vm *v2.VirtualMachine
	err := pollUntil(ctx, opts, fmt.Sprintf("VM identity fields (id=%q)", id), func(ctx context.Context) (bool, string, error) {
		cur, err := client.GetVirtualMachine(ctx, &v2.GetVirtualMachineRequest{Id: id})
		if err != nil {
			return false, "", err
		}
		detail := fmt.Sprintf("namespace=%q name=%q", cur.GetNamespace(), cur.GetName())
		if cur.GetNamespace() != expectedNamespace || cur.GetName() != expectedName {
			return false, detail, nil
		}
		vm = cur
		return true, detail, nil
	})
	if err != nil {
		return nil, err
	}
	return vm, nil
}

// WaitForVMRunningInCentral polls until the VM state is RUNNING.
func WaitForVMRunningInCentral(ctx context.Context, client v2.VirtualMachineServiceClient, id string) (*v2.VirtualMachine, error) {
	return WaitForVMRunningInCentralWithOptions(ctx, client, defaultCentralWaitOptions(), id)
}

// WaitForVMRunningInCentralWithOptions is like WaitForVMRunningInCentral with explicit poll/timeouts.
func WaitForVMRunningInCentralWithOptions(ctx context.Context, client v2.VirtualMachineServiceClient, opts WaitOptions, id string) (*v2.VirtualMachine, error) {
	var vm *v2.VirtualMachine
	err := pollUntil(ctx, opts, fmt.Sprintf("VM RUNNING in Central (id=%q)", id), func(ctx context.Context) (bool, string, error) {
		cur, err := client.GetVirtualMachine(ctx, &v2.GetVirtualMachineRequest{Id: id})
		if err != nil {
			return false, "", err
		}
		st := cur.GetState()
		detail := fmt.Sprintf("state=%s", st.String())
		if st != v2.VirtualMachine_RUNNING {
			return false, detail, nil
		}
		vm = cur
		return true, detail, nil
	})
	if err != nil {
		return nil, err
	}
	return vm, nil
}

// WaitForVMScanNonNil polls until VirtualMachine.scan is non-nil.
func WaitForVMScanNonNil(ctx context.Context, client v2.VirtualMachineServiceClient, id string) (*v2.VirtualMachine, error) {
	return WaitForVMScanNonNilWithOptions(ctx, client, defaultCentralWaitOptions(), id)
}

// WaitForVMScanNonNilWithOptions is like WaitForVMScanNonNil with explicit poll/timeouts.
func WaitForVMScanNonNilWithOptions(ctx context.Context, client v2.VirtualMachineServiceClient, opts WaitOptions, id string) (*v2.VirtualMachine, error) {
	var vm *v2.VirtualMachine
	err := pollUntil(ctx, opts, fmt.Sprintf("VM scan non-nil (id=%q)", id), func(ctx context.Context) (bool, string, error) {
		cur, err := client.GetVirtualMachine(ctx, &v2.GetVirtualMachineRequest{Id: id})
		if err != nil {
			return false, "", err
		}
		if cur.GetScan() == nil {
			return false, "scan is nil", nil
		}
		vm = cur
		return true, "scan present", nil
	})
	if err != nil {
		return nil, err
	}
	return vm, nil
}

// WaitForVMScanTimestamp polls until scan_time is set.
func WaitForVMScanTimestamp(ctx context.Context, client v2.VirtualMachineServiceClient, id string) (*v2.VirtualMachine, error) {
	return WaitForVMScanTimestampWithOptions(ctx, client, defaultCentralWaitOptions(), id)
}

// WaitForVMScanTimestampWithOptions is like WaitForVMScanTimestamp with explicit poll/timeouts.
func WaitForVMScanTimestampWithOptions(ctx context.Context, client v2.VirtualMachineServiceClient, opts WaitOptions, id string) (*v2.VirtualMachine, error) {
	var vm *v2.VirtualMachine
	err := pollUntil(ctx, opts, fmt.Sprintf("VM scan timestamp (id=%q)", id), func(ctx context.Context) (bool, string, error) {
		cur, err := client.GetVirtualMachine(ctx, &v2.GetVirtualMachineRequest{Id: id})
		if err != nil {
			return false, "", err
		}
		sc := cur.GetScan()
		if sc == nil {
			return false, "scan is nil", nil
		}
		if sc.GetScanTime() == nil {
			return false, "scan_time is nil", nil
		}
		vm = cur
		return true, "scan_time set", nil
	})
	if err != nil {
		return nil, err
	}
	return vm, nil
}

// WaitForScanTimestampAfter polls until the VM's scan_time is strictly after the given
// threshold. Use this to wait for a rescan to be reflected in Central.
func WaitForScanTimestampAfter(ctx context.Context, client v2.VirtualMachineServiceClient, id string, after time.Time) (*v2.VirtualMachine, error) {
	return WaitForScanTimestampAfterWithOptions(ctx, client, defaultCentralWaitOptions(), id, after)
}

// WaitForScanTimestampAfterWithOptions is like WaitForScanTimestampAfter with explicit options.
func WaitForScanTimestampAfterWithOptions(ctx context.Context, client v2.VirtualMachineServiceClient, opts WaitOptions, id string, after time.Time) (*v2.VirtualMachine, error) {
	var vm *v2.VirtualMachine
	err := pollUntil(ctx, opts, fmt.Sprintf("scan timestamp after %v (id=%q)", after.Format(time.RFC3339), id), func(ctx context.Context) (bool, string, error) {
		cur, err := client.GetVirtualMachine(ctx, &v2.GetVirtualMachineRequest{Id: id})
		if err != nil {
			return false, "", err
		}
		sc := cur.GetScan()
		if sc == nil || sc.GetScanTime() == nil {
			return false, "scan_time not set yet", nil
		}
		ts := sc.GetScanTime().AsTime()
		if !ts.After(after) {
			return false, fmt.Sprintf("scan_time=%v not yet after %v", ts.Format(time.RFC3339Nano), after.Format(time.RFC3339Nano)), nil
		}
		vm = cur
		return true, fmt.Sprintf("scan_time=%v", ts.Format(time.RFC3339Nano)), nil
	})
	if err != nil {
		return nil, err
	}
	return vm, nil
}

// WaitForVMComponentsReported polls until at least one scan component exists.
func WaitForVMComponentsReported(ctx context.Context, client v2.VirtualMachineServiceClient, id string) (*v2.VirtualMachine, error) {
	return WaitForVMComponentsReportedWithOptions(ctx, client, defaultCentralWaitOptions(), id)
}

// WaitForVMComponentsReportedWithOptions is like WaitForVMComponentsReported with explicit poll/timeouts.
func WaitForVMComponentsReportedWithOptions(ctx context.Context, client v2.VirtualMachineServiceClient, opts WaitOptions, id string) (*v2.VirtualMachine, error) {
	var vm *v2.VirtualMachine
	err := pollUntil(ctx, opts, fmt.Sprintf("VM components reported (id=%q)", id), func(ctx context.Context) (bool, string, error) {
		cur, err := client.GetVirtualMachine(ctx, &v2.GetVirtualMachineRequest{Id: id})
		if err != nil {
			return false, "", err
		}
		if !hasReportedComponents(cur) {
			n := 0
			if cur.GetScan() != nil {
				n = len(cur.GetScan().GetComponents())
			}
			return false, fmt.Sprintf("component count=%d", n), nil
		}
		vm = cur
		return true, "components present", nil
	})
	if err != nil {
		return nil, err
	}
	return vm, nil
}

// WaitForAllVMComponentsScanned polls until every scan component lacks the UNSCANNED note.
func WaitForAllVMComponentsScanned(ctx context.Context, client v2.VirtualMachineServiceClient, id string) (*v2.VirtualMachine, error) {
	return WaitForAllVMComponentsScannedWithOptions(ctx, client, defaultCentralWaitOptions(), id)
}

// WaitForAllVMComponentsScannedWithOptions is like WaitForAllVMComponentsScanned with explicit poll/timeouts.
func WaitForAllVMComponentsScannedWithOptions(ctx context.Context, client v2.VirtualMachineServiceClient, opts WaitOptions, id string) (*v2.VirtualMachine, error) {
	var vm *v2.VirtualMachine
	err := pollUntil(ctx, opts, fmt.Sprintf("all VM components scanned (id=%q)", id), func(ctx context.Context) (bool, string, error) {
		cur, err := client.GetVirtualMachine(ctx, &v2.GetVirtualMachineRequest{Id: id})
		if err != nil {
			return false, "", err
		}
		if !allComponentsScanned(cur) {
			return false, "pending UNSCANNED components or empty component list", nil
		}
		vm = cur
		return true, "all components scanned", nil
	})
	if err != nil {
		return nil, err
	}
	return vm, nil
}

// WaitForScanReady polls GetVirtualMachine in a single loop until every field
// requested in conds is populated. Each poll iteration logs which conditions
// are already met and which are still pending, so partial progress is visible.
func WaitForScanReady(ctx context.Context, client v2.VirtualMachineServiceClient, id string, conds ScanReadiness) (*v2.VirtualMachine, error) {
	return WaitForScanReadyWithOptions(ctx, client, defaultCentralWaitOptions(), id, conds)
}

// WaitForScanReadyWithOptions is like WaitForScanReady with explicit poll/timeouts.
func WaitForScanReadyWithOptions(ctx context.Context, client v2.VirtualMachineServiceClient, opts WaitOptions, id string, conds ScanReadiness) (*v2.VirtualMachine, error) {
	var vm *v2.VirtualMachine
	err := pollUntil(ctx, opts, fmt.Sprintf("scan ready (id=%q)", id), func(ctx context.Context) (bool, string, error) {
		cur, err := client.GetVirtualMachine(ctx, &v2.GetVirtualMachineRequest{Id: id})
		if err != nil {
			return false, "", err
		}
		scan := cur.GetScan()
		if scan == nil {
			return false, "scan is nil", nil
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
		if len(pending) > 0 {
			return false, detail, nil
		}
		vm = cur
		return true, detail, nil
	})
	if err != nil {
		return nil, err
	}
	return vm, nil
}

// listVMPageSize is the page size used when iterating ListVirtualMachines pages.
const listVMPageSize = int32(1000)

// rawListQueryNamespaceAndName builds a Central raw search query matching namespace and VM name.
func rawListQueryNamespaceAndName(namespace, name string) string {
	return fmt.Sprintf("%s:%s+%s:%s", search.Namespace, namespace, search.VirtualMachineName, name)
}

// ListVMByNamespaceName returns the first VirtualMachine in Central whose namespace and name
// exactly match the given values, paging through all results.  Returns (nil, nil) when no
// match is found.
func ListVMByNamespaceName(ctx context.Context, client v2.VirtualMachineServiceClient, namespace, name string) (*v2.VirtualMachine, error) {
	return listVirtualMachineByNamespaceName(ctx, client, namespace, name)
}

// listVirtualMachineByNamespaceName pages ListVirtualMachines until it finds an exact namespace/name match.
func listVirtualMachineByNamespaceName(ctx context.Context, client v2.VirtualMachineServiceClient, namespace, name string) (*v2.VirtualMachine, error) {
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
