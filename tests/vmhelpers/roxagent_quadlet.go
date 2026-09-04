package vmhelpers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	quadletInstallScriptName  = "install.sh"
	quadletContainerFileName  = "roxagent.container"
	guestQuadletStageTemplate = "/tmp/roxagent-quadlet.XXXXXX"
)

// InstallRoxagentQuadlet copies the customer installer plus a staged
// roxagent.container to the guest and runs install.sh --stage-dir so systemd
// ends up with roxagent.service.
func InstallRoxagentQuadlet(ctx context.Context, virt Virtctl, namespace, vm, image, repo2cpeURL, podmanAuthPath string) error {
	if err := requireGuestPodman(ctx, virt, namespace, vm); err != nil {
		return err
	}
	if err := installGuestPodmanAuth(ctx, virt, namespace, vm, podmanAuthPath); err != nil {
		return err
	}

	hostStage, err := stageQuadletInstall(image, repo2cpeURL, podmanAuthPath)
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(hostStage) }()

	return retryOnSSHTransport(ctx, virt.Logf, "install roxagent quadlet", func(ctx context.Context) error {
		guestStage, stderr, err := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
			description:            "mktemp quadlet stage dir",
			transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
		}, "mktemp", "-d", guestQuadletStageTemplate)
		if err != nil {
			return fmt.Errorf("mktemp quadlet stage dir: %w: %s", err, strings.TrimSpace(stderr))
		}
		guestStage = strings.TrimSpace(guestStage)
		if guestStage == "" {
			return errors.New("mktemp quadlet stage dir: empty path")
		}

		for _, name := range []string{quadletInstallScriptName, quadletContainerFileName} {
			if scpErr := scpToGuest(ctx, virt, namespace, vm, filepath.Join(hostStage, name), guestStage+"/"+name); scpErr != nil {
				return scpErr
			}
		}

		prevState := "inactive"
		if s, stateErr := roxagentServeState(ctx, virt, namespace, vm); stateErr == nil {
			prevState = s
		}
		if prevState == "failed" {
			_, _, _ = runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
				description:            "reset-failed roxagent.service",
				transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
			}, "sudo", "systemctl", "reset-failed", roxagentUnit)
		}

		_, stderr, err = runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
			description:            "install.sh --stage-dir",
			transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
		}, "bash", guestStage+"/"+quadletInstallScriptName, "--stage-dir", guestStage)
		if err != nil {
			return wrapRoxagentUnitCmdError(ctx, virt, namespace, vm, "install.sh --stage-dir", err, stderr)
		}

		// install.sh start is a no-op when the unit is already running, so
		// restart only on re-runs to apply a new Image=/Exec= overlay.
		if prevState != "active" && prevState != "activating" {
			return nil
		}
		_, stderr, err = runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
			description:            "systemctl restart roxagent.service",
			transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
		}, "sudo", "systemctl", "restart", roxagentUnit)
		if err != nil {
			return wrapRoxagentUnitCmdError(ctx, virt, namespace, vm, "systemctl restart "+roxagentUnit, err, stderr)
		}
		return nil
	})
}

func requireGuestPodman(ctx context.Context, virt Virtctl, namespace, vm string) error {
	return retryOnSSHTransport(ctx, virt.Logf, "check guest podman", func(ctx context.Context) error {
		_, stderr, err := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
			description:            "podman --version",
			transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
		}, "podman", "--version")
		if err != nil {
			return fmt.Errorf("%w on guest %s/%s (VM image must ship it; unactivated guests cannot dnf install): %v: %s",
				ErrPodmanNotFound, namespace, vm, err, strings.TrimSpace(stderr))
		}
		return nil
	})
}

func installGuestPodmanAuth(ctx context.Context, virt Virtctl, namespace, vm, podmanAuthPath string) error {
	if strings.TrimSpace(podmanAuthPath) == "" {
		return nil
	}
	if _, err := os.Stat(podmanAuthPath); err != nil {
		return fmt.Errorf("podman auth file %q: %w", podmanAuthPath, err)
	}
	return retryOnSSHTransport(ctx, virt.Logf, "install guest podman auth", func(ctx context.Context) error {
		if err := scpToGuest(ctx, virt, namespace, vm, podmanAuthPath, guestPodmanAuthStagingPath); err != nil {
			return err
		}
		defer func() {
			_, _, _ = runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
				description:            "remove staged podman auth file",
				transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
			}, "rm", "-f", guestPodmanAuthStagingPath)
		}()
		_, stderr, err := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
			description:            "install /etc/containers/auth.json",
			transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
		}, "sudo", "install", "-D", "-m", "0600", guestPodmanAuthStagingPath, guestPodmanAuthPath)
		if err != nil {
			return fmt.Errorf("install %s: %w: %s", guestPodmanAuthPath, err, strings.TrimSpace(stderr))
		}
		return nil
	})
}

func scpToGuest(ctx context.Context, virt Virtctl, namespace, vm, src, dst string) error {
	return retryCopyToGuest(ctx, virt.Logf, src, dst, defaultSSHTransportRetryAttempts, defaultSSHTransportRetryInterval, func() (string, error) {
		return virt.SCPTo(ctx, namespace, vm, src, dst)
	})
}

// wrapSCPError attaches errSSHTransport only for retryable SSH failures so
// retryOnSSHTransport does not spin on terminal auth errors.
func wrapSCPError(src, dst, stderr string, err error) error {
	trimmed := strings.TrimSpace(stderr)
	isSSH, retryable, category := classifySSHFailure(stderr, err)
	if !isSSH {
		return fmt.Errorf("virtctl scp %s -> %s: %w: %s", src, dst, err, trimmed)
	}
	wrapped := err
	if isSSHBannerTimeoutFailure(stderr) {
		wrapped = errors.Join(err, ErrSSHConnectivityStalled)
	}
	if isSSHAuthenticationFailure(stderr) {
		wrapped = errors.Join(wrapped, ErrSSHAuthenticationFailed)
	}
	if retryable {
		return fmt.Errorf("%w: virtctl scp %s -> %s: retryable SSH %s failure: %w: %s",
			errSSHTransport, src, dst, category, wrapped, trimmed)
	}
	return fmt.Errorf("virtctl scp %s -> %s: terminal SSH %s failure: %w: %s",
		src, dst, category, wrapped, trimmed)
}

func retryCopyToGuest(ctx context.Context, logf func(string, ...any), src, dst string, attempts int, interval time.Duration, copyFn func() (string, error)) error {
	if attempts <= 0 {
		attempts = defaultSSHTransportRetryAttempts
	}
	if interval <= 0 {
		interval = defaultSSHTransportRetryInterval
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		stderr, err := copyFn()
		if err == nil {
			return nil
		}
		lastErr = wrapSCPError(src, dst, stderr, err)
		isSSH, retryable, category := classifySSHFailure(stderr, err)
		if !isSSH || !retryable || attempt >= attempts {
			return lastErr
		}
		if logf != nil {
			logf("virtctl scp %s -> %s: retryable SSH %s (attempt %d/%d)", src, dst, category, attempt, attempts)
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("%w: virtctl scp %s -> %s: context done during retry backoff: %w",
				errSSHTransport, src, dst, ctx.Err())
		case <-timer.C:
		}
	}
	return lastErr
}

func stageQuadletInstall(image, repo2cpeURL, podmanAuthPath string) (string, error) {
	srcDir := filepath.Join(repoRoot(), "compliance", "virtualmachines", "roxagent", "quadlet")
	srcContainer, err := os.ReadFile(filepath.Join(srcDir, quadletContainerFileName))
	if err != nil {
		return "", fmt.Errorf("read %s: %w", quadletContainerFileName, err)
	}
	authFile := ""
	if strings.TrimSpace(podmanAuthPath) != "" {
		authFile = guestPodmanAuthPath
	}
	overlayed, err := overlayQuadletContainer(srcContainer, image, repo2cpeURL, authFile)
	if err != nil {
		return "", err
	}
	installSrc, err := os.ReadFile(filepath.Join(srcDir, quadletInstallScriptName))
	if err != nil {
		return "", fmt.Errorf("read %s: %w", quadletInstallScriptName, err)
	}

	dir, err := os.MkdirTemp("", "roxagent-quadlet-e2e-")
	if err != nil {
		return "", fmt.Errorf("mktemp host quadlet stage: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, quadletContainerFileName), overlayed, 0o644); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, quadletInstallScriptName), installSrc, 0o755); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}

// overlayQuadletContainer rewrites Image= and Exec= for the cluster image and
// a 5m rescan. A non-empty authFile sets REGISTRY_AUTH_FILE on [Service]
// because RHEL 8 Quadlet rejects AuthFile=.
func overlayQuadletContainer(src []byte, image, repo2cpeURL, authFile string) ([]byte, error) {
	if strings.TrimSpace(image) == "" {
		return nil, errors.New("overlayQuadletContainer: image is empty")
	}

	sawImage, sawExec := false, false
	srcLines := strings.Split(string(src), "\n")
	out := make([]string, 0, len(srcLines)+1)
	for _, line := range srcLines {
		switch {
		case strings.HasPrefix(line, "Image="):
			out = append(out, "Image="+image)
			sawImage = true
		case strings.HasPrefix(line, "AuthFile="):
			continue
		case line == "[Service]":
			out = append(out, "[Service]")
			if authFile != "" {
				out = append(out, "Environment=REGISTRY_AUTH_FILE="+authFile)
			}
		case strings.HasPrefix(line, "Exec="):
			out = append(out, overlayExecLine(repo2cpeURL))
			sawExec = true
		default:
			out = append(out, line)
		}
	}
	if !sawImage {
		return nil, errors.New("overlayQuadletContainer: no Image= line")
	}
	if !sawExec {
		return nil, errors.New("overlayQuadletContainer: no Exec= line")
	}
	return []byte(strings.Join(out, "\n")), nil
}

// overlayExecLine is the Quadlet Exec= line. --repo-cpe-url is omitted when
// empty so systemd cannot pass a blank argument to serve.
func overlayExecLine(repo2cpeURL string) string {
	line := "Exec=serve --host-path /host --rescan-interval " + E2ERescanInterval.String()
	if u := strings.TrimSpace(repo2cpeURL); u != "" {
		line += " --repo-cpe-url " + u
	}
	return line
}
