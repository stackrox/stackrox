package vmhelpers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	quadletInstallScriptName  = "install.sh"
	quadletContainerFileName  = "roxagent.container"
	guestQuadletStageTemplate = "/tmp/roxagent-quadlet.XXXXXX"
)

// InstallRoxagentQuadlet copies the customer installer plus a staged
// roxagent.container to the guest and runs install.sh --stage-dir so systemd
// ends up with roxagent.service.
func InstallRoxagentQuadlet(ctx context.Context, virt Virtctl, namespace, vm, image, repo2cpeURL, pullSecretPath string) error {
	if err := requireGuestPodman(ctx, virt, namespace, vm); err != nil {
		return err
	}
	if err := installGuestPodmanAuth(ctx, virt, namespace, vm, pullSecretPath); err != nil {
		return err
	}

	hostStage, err := stageQuadletInstall(image, repo2cpeURL)
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
			return fmt.Errorf("podman not found on guest %s/%s (VM image must ship it; unactivated guests cannot dnf install): %w: %s",
				namespace, vm, err, strings.TrimSpace(stderr))
		}
		return nil
	})
}

func installGuestPodmanAuth(ctx context.Context, virt Virtctl, namespace, vm, pullSecretPath string) error {
	if strings.TrimSpace(pullSecretPath) == "" {
		return nil
	}
	if _, err := os.Stat(pullSecretPath); err != nil {
		return fmt.Errorf("podman pull secret %q: %w", pullSecretPath, err)
	}
	return retryOnSSHTransport(ctx, virt.Logf, "install guest podman auth", func(ctx context.Context) error {
		if err := scpToGuest(ctx, virt, namespace, vm, pullSecretPath, guestPodmanAuthStagingPath); err != nil {
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
	stderr, err := virt.SCPTo(ctx, namespace, vm, src, dst)
	if err != nil {
		return fmt.Errorf("virtctl scp %s -> %s: %w: %s", src, dst, err, strings.TrimSpace(stderr))
	}
	return nil
}

func stageQuadletInstall(image, repo2cpeURL string) (string, error) {
	srcDir := filepath.Join(repoRoot(), "compliance", "virtualmachines", "roxagent", "quadlet")
	srcContainer, err := os.ReadFile(filepath.Join(srcDir, quadletContainerFileName))
	if err != nil {
		return "", fmt.Errorf("read %s: %w", quadletContainerFileName, err)
	}
	overlayed, err := overlayQuadletContainer(srcContainer, image, repo2cpeURL)
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

// overlayQuadletContainer rewrites Image= and Exec= so e2e uses the cluster
// image and a 5m rescan without changing the shipped unit file.
func overlayQuadletContainer(src []byte, image, repo2cpeURL string) ([]byte, error) {
	if strings.TrimSpace(image) == "" {
		return nil, errors.New("overlayQuadletContainer: image is empty")
	}
	if strings.TrimSpace(repo2cpeURL) == "" {
		return nil, errors.New("overlayQuadletContainer: repo2cpeURL is empty")
	}

	var b strings.Builder
	b.Grow(len(src) + len(image) + len(repo2cpeURL))
	sawImage, sawExec := false, false
	first := true
	for line := range strings.SplitSeq(string(src), "\n") {
		if !first {
			b.WriteByte('\n')
		}
		first = false
		switch {
		case strings.HasPrefix(line, "Image="):
			b.WriteString("Image=")
			b.WriteString(image)
			sawImage = true
		case strings.HasPrefix(line, "Exec="):
			b.WriteString("Exec=serve --host-path /host --rescan-interval ")
			b.WriteString(E2ERescanInterval.String())
			b.WriteString(" --repo-cpe-url ")
			b.WriteString(repo2cpeURL)
			sawExec = true
		default:
			b.WriteString(line)
		}
	}
	if !sawImage {
		return nil, errors.New("overlayQuadletContainer: no Image= line")
	}
	if !sawExec {
		return nil, errors.New("overlayQuadletContainer: no Exec= line")
	}
	return []byte(b.String()), nil
}
