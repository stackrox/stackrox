package vmhelpers

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// sshReachabilityPolicy defines poll cadence, per-probe timeout, and consecutive-failure thresholds for SSH classification.
type sshReachabilityPolicy struct {
	pollInterval                time.Duration
	probeTimeout                time.Duration
	authFailureThreshold        int
	bannerTimeoutThreshold      int
	networkUnreachableThreshold int
	probeTimeoutThreshold       int
}

// defaultSSHReachabilityPolicy is the production policy used by waitForSSHReachableImpl to detect stuck or broken SSH.
var defaultSSHReachabilityPolicy = sshReachabilityPolicy{
	pollInterval:                sshReachablePollInterval,
	probeTimeout:                sshProbeAttemptTimeout,
	authFailureThreshold:        sshAuthFailureThreshold,
	bannerTimeoutThreshold:      sshBannerTimeoutThreshold,
	networkUnreachableThreshold: sshNetworkUnreachableThreshold,
	probeTimeoutThreshold:       sshProbeTimeoutThreshold,
}

// sshProbeCounters holds consecutive failure counts for each SSH failure category between successful probes.
type sshProbeCounters struct {
	authFailures               int
	bannerTimeoutFailures      int
	networkUnreachableFailures int
	probeTimeoutFailures       int
}

// resetAll clears all category counters after a successful SSH probe.
func (c *sshProbeCounters) resetAll() {
	c.authFailures = 0
	c.bannerTimeoutFailures = 0
	c.networkUnreachableFailures = 0
	c.probeTimeoutFailures = 0
}

// sshProbeDecision is the result of one classifyFailure evaluation: human detail and optional terminal error.
type sshProbeDecision struct {
	detail      string
	terminalErr error
}

// runSSHReachabilityProbe runs a minimal SSH command (`true`) under the policy's per-attempt timeout and returns stderr.
func runSSHReachabilityProbe(ctx context.Context, policy sshReachabilityPolicy, virt Virtctl, namespace, vm string) (stderr string, err error) {
	probeCtx, cancel := context.WithTimeout(ctx, policy.probeTimeout)
	defer cancel()
	_, stderr, err = runSSHCommandWithFramework(probeCtx, virt, namespace, vm, sshCommandRunOptions{
		description:            "ssh reachability probe",
		transportRetryAttempts: 1,
	}, "true")
	return strings.TrimSpace(stderr), err
}

// classifyFailure decides whether to keep retrying while the VM is likely still booting
// or to stop early when repeated failures strongly indicate a broken guest.
func (p sshReachabilityPolicy) classifyFailure(counters *sshProbeCounters, virt Virtctl, err error, stderr string) sshProbeDecision {
	detail := sshProbeFailureDetail(err, stderr)
	switch {
	case isSSHAuthenticationFailure(stderr):
		counters.authFailures++
		counters.bannerTimeoutFailures = 0
		counters.networkUnreachableFailures = 0
		counters.probeTimeoutFailures = 0
		msg := fmt.Sprintf("ssh authentication failed (%d/%d consecutive): %s",
			counters.authFailures, p.authFailureThreshold, detail)
		if counters.authFailures >= p.authFailureThreshold {
			return sshProbeDecision{
				detail: msg,
				terminalErr: fmt.Errorf("%w for ssh user %q after %d consecutive attempts (likely stale/missing authorized key): %s",
					ErrSSHAuthenticationFailed, sshUserForDiagnostic(virt), counters.authFailures, truncateGuestWaitDetail(detail)),
			}
		}
		return sshProbeDecision{detail: msg}
	case isSSHBannerTimeoutFailure(stderr):
		counters.authFailures = 0
		counters.bannerTimeoutFailures++
		counters.networkUnreachableFailures = 0
		counters.probeTimeoutFailures = 0
		msg := fmt.Sprintf("ssh banner timeout (%d/%d consecutive): %s",
			counters.bannerTimeoutFailures, p.bannerTimeoutThreshold, detail)
		if counters.bannerTimeoutFailures >= p.bannerTimeoutThreshold {
			return sshProbeDecision{
				detail: msg,
				terminalErr: fmt.Errorf("%w after %d consecutive attempts (guest may be unhealthy and need recreation): %s",
					ErrSSHConnectivityStalled, counters.bannerTimeoutFailures, truncateGuestWaitDetail(detail)),
			}
		}
		return sshProbeDecision{detail: msg}
	case isSSHNetworkUnreachableFailure(stderr):
		counters.authFailures = 0
		counters.bannerTimeoutFailures = 0
		counters.networkUnreachableFailures++
		counters.probeTimeoutFailures = 0
		msg := fmt.Sprintf("ssh network unavailable (%d/%d consecutive): %s",
			counters.networkUnreachableFailures, p.networkUnreachableThreshold, detail)
		if counters.networkUnreachableFailures >= p.networkUnreachableThreshold {
			return sshProbeDecision{
				detail: msg,
				terminalErr: fmt.Errorf("%w after %d consecutive attempts (guest network/sshd may be unhealthy and need recreation): %s",
					ErrSSHConnectivityStalled, counters.networkUnreachableFailures, truncateGuestWaitDetail(detail)),
			}
		}
		return sshProbeDecision{detail: msg}
	case isSSHProbeTimeoutFailure(err):
		counters.authFailures = 0
		counters.bannerTimeoutFailures = 0
		counters.networkUnreachableFailures = 0
		counters.probeTimeoutFailures++
		msg := fmt.Sprintf("ssh probe timed out (%d/%d consecutive): %s",
			counters.probeTimeoutFailures, p.probeTimeoutThreshold, detail)
		if counters.probeTimeoutFailures >= p.probeTimeoutThreshold {
			return sshProbeDecision{
				detail: msg,
				terminalErr: fmt.Errorf("%w after %d consecutive probe timeouts (ssh command appears stuck): %s",
					ErrSSHConnectivityStalled, counters.probeTimeoutFailures, truncateGuestWaitDetail(detail)),
			}
		}
		return sshProbeDecision{detail: msg}
	default:
		counters.resetAll()
		return sshProbeDecision{detail: fmt.Sprintf("ssh not reachable yet: %s", detail)}
	}
}
