package vmhelpers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

// Guest preparation tuning: dnf priming and RHSM precheck intervals, vsock/RHSM markers,
// SSH retry thresholds for prechecks, and dnf lock contention retry budget.
const (
	guestCommandErrorMaxLen = 4096
	// dnfHistoryPrimingPackage is a lightweight package used to force a real dnf
	// transaction without pulling a large dependency graph.
	dnfHistoryPrimingPackage = "bc"
	// dnfHistoryPrimingPollInterval is how often we re-check whether RHSM background
	// uploader activity has finished before starting dnf priming.
	dnfHistoryPrimingPollInterval = 10 * time.Second
	// dnfHistoryPrimingWaitLogEveryAttempts controls periodic logging cadence while
	// waiting for RHSM uploader processes to exit.
	dnfHistoryPrimingWaitLogEveryAttempts = 6
	rhsmUploaderActiveMarker              = "RHSM_UPLOADER_ACTIVE"
	rhsmUploaderProcessPattern            = "/usr/libexec/rhsm-package-profile-uploader"
	vsockReadinessMarker                  = "VSOCK_READY"
	rhsmPrecheckSSHRetryThreshold         = 5
	dnfLockRetryMaxAttempts               = 30
)

// WaitForSSHReachable is a high-level guest preparation step.
// SSH retry and classification internals are implemented in guest_ssh_internal.go.
func WaitForSSHReachable(t testing.TB, ctx context.Context, virt Virtctl, namespace, vm string) error {
	return waitForSSHReachableImpl(t, ctx, virt, namespace, vm)
}

// WaitForCloudInitFinished waits for cloud-init to complete on the guest.
// SSH/sudo error parsing internals are implemented in guest_ssh_internal.go.
func WaitForCloudInitFinished(ctx context.Context, virt Virtctl, namespace, vm string) error {
	return waitForCloudInitFinishedImpl(ctx, virt, namespace, vm)
}

// VerifySudoWorks checks that passwordless sudo works for the SSH user.
// SSH/sudo error parsing internals are implemented in guest_ssh_internal.go.
func VerifySudoWorks(ctx context.Context, virt Virtctl, namespace, vm string) error {
	return verifySudoWorksImpl(ctx, virt, namespace, vm)
}

// probeRHSMUploaderIdle checks whether the RHSM uploader process is running.
// Returns (true, detail, nil) when idle and (false, detail, nil) when active.
// SSH transport errors are propagated for the caller's retry logic.
func probeRHSMUploaderIdle(ctx context.Context, virt Virtctl, namespace, vm string) (idle bool, detail string, err error) {
	stdout, _, err := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
		description:            "pgrep RHSM uploader",
		transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
	}, "pgrep", "-f", rhsmUploaderProcessPattern)
	pids := strings.TrimSpace(stdout)
	if err != nil {
		if errors.Is(err, errSSHTransport) {
			return false, "", err
		}
		// pgrep exits 1 when no processes match.
		return true, "no RHSM uploader processes", nil
	}
	if pids == "" {
		return true, "no RHSM uploader processes", nil
	}
	return false, fmt.Sprintf("%s pids=%s", rhsmUploaderActiveMarker, pids), nil
}

// terminateRHSMUploader finds and kills RHSM uploader processes.
func terminateRHSMUploader(ctx context.Context, virt Virtctl, namespace, vm string) string {
	stdout, _, err := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
		description:            "pgrep RHSM uploader for terminate",
		transportRetryAttempts: 2,
	}, "pgrep", "-f", rhsmUploaderProcessPattern)
	pids := strings.TrimSpace(stdout)
	if err != nil || pids == "" {
		return "no RHSM uploader process"
	}
	killArgs := append([]string{"sudo", "kill"}, strings.Fields(pids)...)
	_, _, _ = runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
		description:            "kill RHSM uploader",
		transportRetryAttempts: 1,
	}, killArgs...)
	time.Sleep(1 * time.Second)
	stdout2, _, _ := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
		description:            "pgrep RHSM uploader after kill",
		transportRetryAttempts: 1,
	}, "pgrep", "-f", rhsmUploaderProcessPattern)
	if remaining := strings.TrimSpace(stdout2); remaining != "" {
		return fmt.Sprintf("RHSM uploader still running pids=%s", remaining)
	}
	return "RHSM uploader terminated"
}

// checkVsockReadiness tests /dev/vsock presence and gathers diagnostics when absent.
func checkVsockReadiness(ctx context.Context, virt Virtctl, namespace, vm string) (ready bool, detail string, err error) {
	_, _, testErr := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
		description:            "vsock device check",
		transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
	}, "test", "-e", "/dev/vsock")
	if testErr == nil {
		return true, vsockReadinessMarker + " /dev/vsock present", nil
	}
	if errors.Is(testErr, errSSHTransport) {
		return false, "", testErr
	}
	// /dev/vsock absent — gather diagnostics via separate calls.
	var diag []string
	lsOut, _, _ := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
		description: "vsock diagnostic ls", transportRetryAttempts: 1,
	}, "ls", "-l", "/dev/vsock")
	if s := strings.TrimSpace(lsOut); s != "" {
		diag = append(diag, "ls: "+s)
	}
	lsmodOut, _, _ := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
		description: "vsock diagnostic lsmod", transportRetryAttempts: 1,
	}, "lsmod")
	for _, line := range strings.Split(lsmodOut, "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "vsock") || strings.Contains(lower, "vhost") {
			diag = append(diag, "module: "+strings.TrimSpace(line))
		}
	}
	diagStr := "no diagnostic info available"
	if len(diag) > 0 {
		diagStr = strings.Join(diag, "; ")
	}
	return false, fmt.Sprintf("VSOCK_MISSING /dev/vsock absent (%s)", diagStr), nil
}

// dnfPrimingArgs returns the sudo dnf arguments for the priming operation.
func dnfPrimingArgs(reinstall bool) []string {
	action := "install"
	if reinstall {
		action = "reinstall"
	}
	return []string{
		"sudo", "dnf", "-y", action,
		"--setopt=install_weak_deps=False",
		"--setopt=exit_on_lock=True",
		dnfHistoryPrimingPackage,
	}
}

// isDnfLockContentionOutput reports whether dnf stderr/stdout indicates another process holds the transaction lock.
func isDnfLockContentionOutput(output string) bool {
	lower := strings.ToLower(strings.TrimSpace(output))
	return strings.Contains(lower, "metadata already locked by") ||
		strings.Contains(lower, "failed to obtain the transaction lock") ||
		strings.Contains(lower, "waiting for process with pid")
}

// dnfHistoryHasTransactionsOutput returns true if dnf history list output contains at least one numeric transaction ID line.
func dnfHistoryHasTransactionsOutput(output string) bool {
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) == 0 {
			continue
		}
		if _, err := strconv.Atoi(fields[0]); err == nil {
			return true
		}
	}
	return false
}

// shouldLogDnfPrimingWaitAttempt limits log noise during RHSM uploader / dnf priming polls (first attempt and periodic).
func shouldLogDnfPrimingWaitAttempt(attempt int) bool {
	if attempt <= 1 {
		return true
	}
	return attempt%dnfHistoryPrimingWaitLogEveryAttempts == 0
}

// logPrimingPrecheck logs one dnf/guest priming precheck iteration with attempt and optional max-attempt context.
func logPrimingPrecheck(logf func(string, ...any), namespace, vm, msg string, attempts, maxAttempts int, maxKnown bool, detail string) {
	if logf == nil {
		return
	}
	if maxKnown {
		logf("dnf priming precheck: %s on %s/%s (attempt %d/%d, retries left: %d): %s",
			msg, namespace, vm, attempts, maxAttempts, max(0, maxAttempts-attempts), detail)
	} else {
		logf("dnf priming precheck: %s on %s/%s (attempt %d): %s",
			msg, namespace, vm, attempts, detail)
	}
}

// waitForRHSMUploaderIdle polls until the rhsmcertd package-profile uploader is not running so dnf does not contend with it.
func waitForRHSMUploaderIdle(ctx context.Context, virt Virtctl, namespace, vm string) error {
	attempts := 0
	maxAttempts, maxKnown := maxGuestPollAttempts(ctx, dnfHistoryPrimingPollInterval)
	lastDetail := ""
	err := wait.PollUntilContextCancel(ctx, dnfHistoryPrimingPollInterval, true, func(ctx context.Context) (bool, error) {
		attempts++
		idle, detail, probeErr := probeRHSMUploaderIdle(ctx, virt, namespace, vm)
		if probeErr != nil {
			lastDetail = probeErr.Error()
			if errors.Is(probeErr, errSSHTransport) {
				if shouldLogDnfPrimingWaitAttempt(attempts) {
					logPrimingPrecheck(virt.Logf, namespace, vm, "SSH transport error - retrying", attempts, maxAttempts, maxKnown, lastDetail)
				}
				return false, nil
			}
			return false, fmt.Errorf("dnf priming precheck on %s/%s: %w", namespace, vm, probeErr)
		}
		lastDetail = detail
		if idle {
			if attempts > 1 && virt.Logf != nil {
				virt.Logf("dnf priming precheck: RHSM uploader idle on %s/%s after %d attempt(s)", namespace, vm, attempts)
			}
			return true, nil
		}
		if shouldLogDnfPrimingWaitAttempt(attempts) {
			logPrimingPrecheck(virt.Logf, namespace, vm, "waiting for RHSM uploader", attempts, maxAttempts, maxKnown, detail)
		}
		return false, nil
	})
	if err != nil {
		if lastDetail != "" {
			return fmt.Errorf("dnf priming precheck on %s/%s failed after %d attempt(s): %w (last detail: %s)",
				namespace, vm, attempts, err, lastDetail)
		}
		return fmt.Errorf("dnf priming precheck on %s/%s failed after %d attempt(s): %w",
			namespace, vm, attempts, err)
	}
	return nil
}

// ensureVsockReady retries the virtio-vsock device check until /dev/vsock exists or SSH transport errors are exhausted.
func ensureVsockReady(ctx context.Context, virt Virtctl, namespace, vm, stage string) error {
	return retryOnSSHTransport(ctx, virt.Logf, fmt.Sprintf("vsock precheck before %s", stage), func(ctx context.Context) error {
		ready, detail, err := checkVsockReadiness(ctx, virt, namespace, vm)
		if err != nil {
			return err
		}
		if !ready {
			return fmt.Errorf("vsock precheck before %s on %s/%s: %s", stage, namespace, vm, detail)
		}
		if virt.Logf != nil {
			virt.Logf("vsock precheck before %s on %s/%s: ready", stage, namespace, vm)
		}
		return nil
	})
}

// isDnfHistoryAlreadyPopulated runs dnf history list on the guest and reports whether any transactions exist yet.
func isDnfHistoryAlreadyPopulated(ctx context.Context, virt Virtctl, namespace, vm string) (bool, error) {
	stdout, stderr, err := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
		description:            "dnf history precheck",
		transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
	}, "sudo", "dnf", "-q", "history", "list", "--reverse")
	combined := strings.TrimSpace(stdout + "\n" + stderr)
	if err != nil {
		return false, fmt.Errorf("dnf history precheck: %w: %s", err, formatGuestCommandOutputForError(combined))
	}
	return dnfHistoryHasTransactionsOutput(combined), nil
}

// primeDnfHistoryWithTinyPackageTransaction records a deterministic dnf transaction by
// installing or reinstalling one tiny package.
// This is not a random package despite the
// public API name PopulateDnfHistoryWithRandomPackage (kept for the suite plan contract).
func primeDnfHistoryWithTinyPackageTransaction(ctx context.Context, virt Virtctl, namespace, vm string) error {
	if err := ensureVsockReady(ctx, virt, namespace, vm, "dnf history priming"); err != nil {
		return err
	}
	populated, precheckErr := isDnfHistoryAlreadyPopulated(ctx, virt, namespace, vm)
	if precheckErr == nil && populated {
		if virt.Logf != nil {
			virt.Logf("dnf history priming skipped on %s/%s: history already has transactions", namespace, vm)
		}
		return nil
	}
	if precheckErr != nil && virt.Logf != nil {
		virt.Logf("dnf history precheck on %s/%s failed; continuing with priming: %v", namespace, vm, precheckErr)
	}

	if err := waitForRHSMUploaderIdle(ctx, virt, namespace, vm); err != nil {
		return err
	}

	// Determine install vs reinstall once; the package status won't change
	// between lock-contention retries.
	_, _, rpmErr := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
		description:            "check priming package installed",
		transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
	}, "rpm", "-q", dnfHistoryPrimingPackage)
	if rpmErr != nil && errors.Is(rpmErr, errSSHTransport) {
		return fmt.Errorf("dnf history priming: %w", rpmErr)
	}
	args := dnfPrimingArgs(rpmErr == nil)

	for attempt := 1; attempt <= dnfLockRetryMaxAttempts; attempt++ {
		stdout, stderr, err := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
			description:            "dnf history priming command",
			transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
		}, args...)
		if err == nil {
			return nil
		}
		combined := strings.TrimSpace(stdout + "\n" + stderr)
		formatted := formatGuestCommandOutputForError(combined)
		if !isDnfLockContentionOutput(combined) {
			return fmt.Errorf("dnf history priming: %w: %s", err, formatted)
		}
		if virt.Logf != nil {
			virt.Logf("dnf history priming lock contention on %s/%s (attempt %d/%d): %s",
				namespace, vm, attempt, dnfLockRetryMaxAttempts, formatted)
		}

		terminateCtx, terminateCancel := context.WithTimeout(ctx, 30*time.Second)
		terminateDetail := terminateRHSMUploader(terminateCtx, virt, namespace, vm)
		terminateCancel()
		if virt.Logf != nil {
			virt.Logf("dnf history priming remediation on %s/%s (attempt %d/%d): %s",
				namespace, vm, attempt, dnfLockRetryMaxAttempts, terminateDetail)
		}

		idleCtx, idleCancel := context.WithTimeout(ctx, 2*time.Minute)
		idleErr := waitForRHSMUploaderIdle(idleCtx, virt, namespace, vm)
		idleCancel()
		if idleErr != nil && virt.Logf != nil {
			virt.Logf("dnf history priming remediation: RHSM uploader still not idle on %s/%s: %v",
				namespace, vm, idleErr)
		}
	}
	return fmt.Errorf("dnf history priming: lock contention persisted after %d attempts", dnfLockRetryMaxAttempts)
}

// formatGuestCommandOutputForError trims guest command output and truncates it for error message inclusion.
func formatGuestCommandOutputForError(output string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return "<no guest stdout/stderr>"
	}
	if len(output) <= guestCommandErrorMaxLen {
		return output
	}
	return output[:guestCommandErrorMaxLen] + fmt.Sprintf(" ... (truncated from %d bytes)", len(output))
}

// overallStatusFromSubscriptionManagerOutput parses the "Overall Status:" line from subscription-manager combined output.
func overallStatusFromSubscriptionManagerOutput(output string) (status string, found bool) {
	const prefix = "overall status:"
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(strings.ToLower(line), prefix) {
			continue
		}
		return strings.TrimSpace(line[len(prefix):]), true
	}
	return "", false
}

// isActivatedOverallStatus reports whether subscription-manager overall status text means the system is subscribed/registered.
func isActivatedOverallStatus(status string) bool {
	status = strings.TrimSpace(status)
	return strings.EqualFold(status, "Current") || strings.EqualFold(status, "Registered")
}

// activationStatusFromCommandOutput interprets subscription-manager status stdout/stderr into activated vs error vs fallback identity parsing.
func activationStatusFromCommandOutput(stdout, stderr string, cmdErr error) (activated bool, details string, err error) {
	combined := strings.TrimSpace(stdout + "\n" + stderr)
	if status, found := overallStatusFromSubscriptionManagerOutput(combined); found {
		return isActivatedOverallStatus(status), combined, nil
	}
	if cmdErr != nil {
		return false, combined, fmt.Errorf("subscription-manager status: %w (output: %s)", cmdErr, formatGuestCommandOutputForError(combined))
	}
	return activationFromSubscriptionManagerOutput(stdout), strings.TrimSpace(stdout), nil
}

// GetActivationStatus runs subscription-manager status and reports whether the guest looks activated.
func GetActivationStatus(ctx context.Context, virt Virtctl, namespace, vm string) (activated bool, details string, err error) {
	stdout, stderr, err := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
		description:            "subscription-manager status",
		transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
	}, "sudo", "subscription-manager", "status")
	activated, details, statusErr := activationStatusFromCommandOutput(stdout, stderr, err)
	if statusErr != nil {
		return false, details, fmt.Errorf("subscription-manager status on %s/%s: %w", namespace, vm, statusErr)
	}
	return activated, details, nil
}

// activationFromSubscriptionManagerOutput returns true when subscription-manager stdout contains an activated overall status.
func activationFromSubscriptionManagerOutput(stdout string) bool {
	status, found := overallStatusFromSubscriptionManagerOutput(stdout)
	return found && isActivatedOverallStatus(status)
}

// isSubscriptionManagerAlreadyRegisteredOutput detects the common "already registered" stderr/stdout from subscription-manager register.
func isSubscriptionManagerAlreadyRegisteredOutput(output string) bool {
	output = strings.ToLower(strings.TrimSpace(output))
	return strings.Contains(output, "this system is already registered")
}

// ActivateWithRHC registers the guest with Red Hat subscription-manager.
func ActivateWithRHC(ctx context.Context, virt Virtctl, namespace, vm, org, activationKey, activationEndpoint string) error {
	if org == "" || activationKey == "" {
		return errors.New("activation org and activation key are required")
	}
	args := []string{"sudo", "subscription-manager", "register", "--org", org, "--activationkey", activationKey}
	if activationEndpoint != "" {
		args = append(args, "--serverurl", activationEndpoint)
	}
	stdout, stderr, err := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
		description:            "subscription-manager register",
		transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
		suppressLog:            true,
	}, args...)
	if err != nil {
		combined := strings.TrimSpace(stdout + "\n" + stderr)
		if isSubscriptionManagerAlreadyRegisteredOutput(combined) {
			return nil
		}
		return fmt.Errorf("subscription-manager register: %w: %s", err, formatGuestCommandOutputForError(combined))
	}
	return nil
}

// VerifyActivationSucceeded re-checks subscription-manager status after registration.
func VerifyActivationSucceeded(ctx context.Context, virt Virtctl, namespace, vm string) error {
	ok, details, err := GetActivationStatus(ctx, virt, namespace, vm)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("guest not activated after register; status output:\n%s", details)
	}
	return nil
}

// PopulateDnfHistoryWithRandomPackage ensures dnf has at least one recent transaction for scanning tests.
// Naming follows the VM-scanning plan; behavior is deterministic—see primeDnfHistoryWithTinyPackageTransaction.
func PopulateDnfHistoryWithRandomPackage(ctx context.Context, virt Virtctl, namespace, vm string) error {
	return primeDnfHistoryWithTinyPackageTransaction(ctx, virt, namespace, vm)
}
