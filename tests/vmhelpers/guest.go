package vmhelpers

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const (
	guestCommandErrorMaxLen = 4096
	vsockReadinessMarker    = "VSOCK_READY"
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
