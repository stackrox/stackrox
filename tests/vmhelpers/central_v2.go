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

// v2ListPageSize matches VirtualMachineV2Service's server-side defaultPageSize.
// Requests above that are clamped, so callers must page.
const (
	v2ListPageSize = 100
	v2ListMaxPages = 200
)

// ListV2VMByNamespaceName returns the first VM matching namespace and name.
// Returns (nil, nil) when no match is found.
func ListV2VMByNamespaceName(ctx context.Context, client v2.VirtualMachineV2ServiceClient, namespace, name string) (*v2.VMListItem, error) {
	resp, err := client.ListVMs(ctx, &v2.ListVMsRequest{
		Query: &v2.RawQuery{
			Query: rawListQueryNamespaceAndName(namespace, name),
		},
	})
	if err != nil {
		return nil, err
	}
	if vms := resp.GetVms(); len(vms) > 0 {
		return vms[0], nil
	}
	return nil, nil
}

// WaitForV2VMPresentInCentral polls ListVMs until a VM matches namespace and name.
func WaitForV2VMPresentInCentral(ctx context.Context, client v2.VirtualMachineV2ServiceClient, opts WaitOptions, namespace, name string) (*v2.VMListItem, error) {
	var found *v2.VMListItem
	err := pollUntil(ctx, opts, fmt.Sprintf("V2 VM present in Central (namespace=%q name=%q)", namespace, name), func(ctx context.Context) (bool, string, error) {
		vm, err := ListV2VMByNamespaceName(ctx, client, namespace, name)
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

type vmV2ConditionCheck func(vm *v2.VMDetail) (done bool, detail string)

func waitForV2VMCondition(ctx context.Context, client v2.VirtualMachineV2ServiceClient, opts WaitOptions, id, desc string, check vmV2ConditionCheck) (*v2.VMDetail, error) {
	var vm *v2.VMDetail
	err := pollUntil(ctx, opts, desc, func(ctx context.Context) (bool, string, error) {
		cur, err := client.GetVM(ctx, &v2.GetVMRequest{Id: id})
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

// WaitForV2VMIdentityFields polls GetVM until id maps to the expected namespace and name.
func WaitForV2VMIdentityFields(ctx context.Context, client v2.VirtualMachineV2ServiceClient, opts WaitOptions, id, expectedNamespace, expectedName string) (*v2.VMDetail, error) {
	return waitForV2VMCondition(ctx, client, opts, id, fmt.Sprintf("V2 VM identity fields (id=%q)", id), func(vm *v2.VMDetail) (bool, string) {
		detail := fmt.Sprintf("namespace=%q name=%q", vm.GetNamespace(), vm.GetName())
		return vm.GetNamespace() == expectedNamespace && vm.GetName() == expectedName, detail
	})
}

// WaitForV2VMRunningInCentral polls until the VM state is VM_STATE_RUNNING.
func WaitForV2VMRunningInCentral(ctx context.Context, client v2.VirtualMachineV2ServiceClient, opts WaitOptions, id string) (*v2.VMDetail, error) {
	return waitForV2VMCondition(ctx, client, opts, id, fmt.Sprintf("V2 VM RUNNING (id=%q)", id), func(vm *v2.VMDetail) (bool, string) {
		st := vm.GetState()
		return st == v2.VirtualMachineV2State_VM_STATE_RUNNING, fmt.Sprintf("state=%s", st)
	})
}

// WaitForV2VMLatestScan polls until latest_scan is present with scan_time set.
func WaitForV2VMLatestScan(ctx context.Context, client v2.VirtualMachineV2ServiceClient, opts WaitOptions, id string) (*v2.VMDetail, error) {
	return waitForV2VMCondition(ctx, client, opts, id, fmt.Sprintf("V2 VM latest_scan (id=%q)", id), func(vm *v2.VMDetail) (bool, string) {
		scan := vm.GetLatestScan()
		if scan == nil {
			return false, "latest_scan is nil"
		}
		if scan.GetScanTime() == nil {
			return false, "latest_scan.scan_time is nil"
		}
		return true, "latest_scan present"
	})
}

// WaitForV2ScanReady polls until the VM has scanned components and none are NOT_SCANNED.
func WaitForV2ScanReady(ctx context.Context, client v2.VirtualMachineV2ServiceClient, opts WaitOptions, id string) (*v2.VMDetail, error) {
	var detail *v2.VMDetail
	err := pollUntil(ctx, opts, fmt.Sprintf("V2 scan ready (id=%q)", id), func(ctx context.Context) (bool, string, error) {
		vm, err := client.GetVM(ctx, &v2.GetVMRequest{Id: id})
		if err != nil {
			return false, "", err
		}
		scan := vm.GetLatestScan()
		if scan == nil {
			return false, "latest_scan is nil", nil
		}

		comps, total, err := ListAllVMComponents(ctx, client, id)
		if err != nil {
			return false, "", err
		}

		var ready, pending []string
		if total > 0 {
			ready = append(ready, fmt.Sprintf("components=%d", total))
		} else {
			pending = append(pending, "components")
		}

		if total > 0 && !slices.ContainsFunc(comps, func(c *v2.VMComponentRow) bool {
			return c.GetScanStatus() == v2.ScanStatus_NOT_SCANNED || c.GetScanStatus() == v2.ScanStatus_SCAN_PENDING
		}) {
			ready = append(ready, "all-scanned")
		} else {
			pending = append(pending, "all-scanned")
		}

		if os := scan.GetScanOs(); os != "" {
			ready = append(ready, fmt.Sprintf("os=%q", os))
		} else if guestOS := vm.GetGuestOs(); guestOS != "" {
			ready = append(ready, fmt.Sprintf("guest_os=%q", guestOS))
		} else if len(pending) == 0 {
			ready = append(ready, "os=<not reported>")
		} else {
			pending = append(pending, "os")
		}

		pollDetail := fmt.Sprintf("ready:[%s] waiting:[%s]", strings.Join(ready, ","), strings.Join(pending, ","))
		if len(pending) == 0 {
			detail = vm
		}
		return len(pending) == 0, pollDetail, nil
	})
	if err != nil {
		return nil, err
	}
	return detail, nil
}

// ListAllVMComponents pages through ListVMComponents until every component is collected.
func ListAllVMComponents(ctx context.Context, client v2.VirtualMachineV2ServiceClient, vmID string) ([]*v2.VMComponentRow, int32, error) {
	var all []*v2.VMComponentRow
	var total int32
	for page := range v2ListMaxPages {
		offset := int32(page) * v2ListPageSize
		resp, err := client.ListVMComponents(ctx, &v2.ListVMComponentsRequest{
			VmId: vmID,
			Query: &v2.RawQuery{
				Pagination: &v2.Pagination{
					Limit:      v2ListPageSize,
					Offset:     offset,
					SortOption: &v2.SortOption{Field: search.Component.String()},
				},
			},
		})
		if err != nil {
			return nil, 0, err
		}
		total = resp.GetTotalCount()
		all = append(all, resp.GetComponents()...)
		if int32(len(all)) >= total || len(resp.GetComponents()) == 0 {
			return all, total, nil
		}
	}
	return nil, 0, fmt.Errorf("ListVMComponents: exceeded %d pages for vm %s (collected %d, total_count=%d)", v2ListMaxPages, vmID, len(all), total)
}

// ListAllVMCVEsByVM pages through ListVMCVEsByVM until every CVE row is collected.
func ListAllVMCVEsByVM(ctx context.Context, client v2.VirtualMachineV2ServiceClient, vmID string) ([]*v2.VMCVERow, int32, error) {
	var all []*v2.VMCVERow
	var total int32
	for page := range v2ListMaxPages {
		offset := int32(page) * v2ListPageSize
		resp, err := client.ListVMCVEsByVM(ctx, &v2.ListVMCVEsByVMRequest{
			VmId: vmID,
			Query: &v2.RawQuery{
				Pagination: &v2.Pagination{
					Limit:      v2ListPageSize,
					Offset:     offset,
					SortOption: &v2.SortOption{Field: search.CVE.String()},
				},
			},
		})
		if err != nil {
			return nil, 0, err
		}
		total = resp.GetTotalCount()
		all = append(all, resp.GetCves()...)
		if int32(len(all)) >= total || len(resp.GetCves()) == 0 {
			return all, total, nil
		}
	}
	return nil, 0, fmt.Errorf("ListVMCVEsByVM: exceeded %d pages for vm %s (collected %d, total_count=%d)", v2ListMaxPages, vmID, len(all), total)
}

// VulnCountBySeverityTotal sums the per-severity total fields.
func VulnCountBySeverityTotal(c *v2.VulnCountBySeverity) int32 {
	if c == nil {
		return 0
	}
	var n int32
	for _, sev := range []*v2.VulnFixableCount{
		c.GetCritical(),
		c.GetImportant(),
		c.GetModerate(),
		c.GetLow(),
		c.GetUnknown(),
	} {
		n += sev.GetTotal()
	}
	return n
}

// WaitForV2ScanMissingComponent polls until packageName is absent from a scan
// whose latest_scan.scan_time has advanced at least minScanAdvances times past after.
func WaitForV2ScanMissingComponent(
	ctx context.Context,
	client v2.VirtualMachineV2ServiceClient,
	opts WaitOptions,
	id, packageName string,
	after time.Time,
	minScanAdvances int,
) ([]*v2.VMComponentRow, error) {
	if minScanAdvances < 1 {
		minScanAdvances = 1
	}
	advances := 0
	lastSeen := after
	var components []*v2.VMComponentRow
	err := pollUntil(ctx, opts,
		fmt.Sprintf("V2 scan missing package %q after %d scan_time advance(s) past %s (id=%q)",
			packageName, minScanAdvances, after.UTC().Format(time.RFC3339), id),
		func(ctx context.Context) (bool, string, error) {
			vm, err := client.GetVM(ctx, &v2.GetVMRequest{Id: id})
			if err != nil {
				return false, "", err
			}
			scan := vm.GetLatestScan()
			if scan == nil {
				return false, "latest_scan is nil", nil
			}
			st := scan.GetScanTime()
			if st == nil {
				return false, "latest_scan.scan_time is nil", nil
			}
			scanTime := st.AsTime()
			if scanTime.After(lastSeen) {
				advances++
				lastSeen = scanTime
			}
			if advances < minScanAdvances {
				return false, fmt.Sprintf("scan_time advances=%d/%d last=%s",
					advances, minScanAdvances, scanTime.UTC().Format(time.RFC3339)), nil
			}

			filtered, err := client.ListVMComponents(ctx, &v2.ListVMComponentsRequest{
				VmId: id,
				Query: &v2.RawQuery{
					Query: fmt.Sprintf("%s:%q", search.Component, packageName),
				},
			})
			if err != nil {
				return false, "", err
			}
			if filtered.GetTotalCount() > 0 {
				return false, fmt.Sprintf("scan_time advances=%d but package %q still present (matches=%d)",
					advances, packageName, filtered.GetTotalCount()), nil
			}

			comps, total, err := ListAllVMComponents(ctx, client, id)
			if err != nil {
				return false, "", err
			}
			components = comps
			return true, fmt.Sprintf("scan_time advances=%d package %q gone (components=%d)",
				advances, packageName, total), nil
		})
	if err != nil {
		return nil, err
	}
	return components, nil
}
