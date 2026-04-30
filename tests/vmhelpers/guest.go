package vmhelpers

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const (
	guestCommandErrorMaxLen       = 4096
	vsockReadinessMarker          = "VSOCK_READY"
	rhsmPrecheckSSHRetryThreshold = 5
)

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
	diagStr := collectVsockDiagnostics(ctx, virt, namespace, vm)
	return false, fmt.Sprintf("VSOCK_MISSING /dev/vsock absent (%s)", diagStr), nil
}

// collectVsockDiagnostics gathers /dev/vsock and kernel module info from the guest for troubleshooting.
func collectVsockDiagnostics(ctx context.Context, virt Virtctl, namespace, vm string) string {
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
	if len(diag) == 0 {
		return "no diagnostic info available"
	}
	return strings.Join(diag, "; ")
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
// Activation is not strictly required for the guest to be scanned, but knowing the status helps with debugging.
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
