package vmhelpers

import (
	"context"
	"fmt"
	"strings"
	"time"

	v2 "github.com/stackrox/rox/generated/api/v2"
)

// VMImageProbePackage is always installed into StackRox VM scanning
// container-disk images (stackrox/vm-images) specifically so tests can remove
// a known, non-critical package without depending on base-image contents.
const VMImageProbePackage = "bc"

// RemoveGuestRPMPackage erases pkg on the guest with rpm -e --nodeps so
// unactivated VMs (no dnf repos) can still change the live RPM database.
func RemoveGuestRPMPackage(ctx context.Context, virt Virtctl, namespace, vm, pkg string) error {
	return retryOnSSHTransport(ctx, virt.Logf, "remove guest RPM package "+pkg, func(ctx context.Context) error {
		_, stderr, err := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
			description:            "rpm -e --nodeps " + pkg,
			transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
		}, "sudo", "rpm", "-e", "--nodeps", pkg)
		if err != nil {
			return fmt.Errorf("rpm -e --nodeps %s: %w: %s", pkg, err, strings.TrimSpace(stderr))
		}
		_, _, qErr := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
			description:            "rpm -q after erase " + pkg,
			transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
		}, "rpm", "-q", pkg)
		if qErr == nil {
			return fmt.Errorf("package %s still installed after rpm -e", pkg)
		}
		virt.Logf("removed guest RPM %s on %s/%s", pkg, namespace, vm)
		return nil
	})
}

// WaitForScanMissingComponent polls Central until the named package is absent
// from a scan whose scan_time has advanced at least minScanAdvances times past
// after.
//
// minScanAdvances should be at least 2 when waiting on a periodic rescan: the
// first post-removal agent cycle can still observe the pre-removal RPM DB if
// it overlapped the erase, and Sensor needs a scrape after each agent cycle
// before Central's scan_time moves.
func WaitForScanMissingComponent(
	ctx context.Context,
	client v2.VirtualMachineServiceClient,
	opts WaitOptions,
	id, packageName string,
	after time.Time,
	minScanAdvances int,
) (*v2.VirtualMachine, error) {
	if minScanAdvances < 1 {
		minScanAdvances = 1
	}
	advances := 0
	lastSeen := after
	return waitForVMCondition(ctx, client, opts, id,
		fmt.Sprintf("scan missing package %q after %d scan_time advance(s) past %s (id=%q)",
			packageName, minScanAdvances, after.UTC().Format(time.RFC3339), id),
		func(vm *v2.VirtualMachine) (bool, string) {
			scan := vm.GetScan()
			if scan == nil {
				return false, "scan is nil"
			}
			st := scan.GetScanTime()
			if st == nil {
				return false, "scan_time is nil"
			}
			scanTime := st.AsTime()
			if scanTime.After(lastSeen) {
				advances++
				lastSeen = scanTime
			}
			if advances < minScanAdvances {
				return false, fmt.Sprintf("scan_time advances=%d/%d last=%s (components=%d)",
					advances, minScanAdvances, scanTime.UTC().Format(time.RFC3339), len(scan.GetComponents()))
			}
			for _, c := range scan.GetComponents() {
				if c.GetName() == packageName {
					return false, fmt.Sprintf("scan_time advances=%d but package %q still present (components=%d)",
						advances, packageName, len(scan.GetComponents()))
				}
			}
			return true, fmt.Sprintf("scan_time advances=%d package %q gone (components=%d)",
				advances, packageName, len(scan.GetComponents()))
		})
}
