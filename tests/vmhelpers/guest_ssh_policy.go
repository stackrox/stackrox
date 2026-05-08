package vmhelpers

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SSHReachabilityPolicy defines poll cadence, per-probe timeout, and
// consecutive-failure thresholds for SSH classification.
type SSHReachabilityPolicy struct {
	PollInterval                time.Duration
	ProbeTimeout                time.Duration
	AuthFailureThreshold        int
	BannerTimeoutThreshold      int
	NetworkUnreachableThreshold int
	ProbeTimeoutThreshold       int
}

// DefaultSSHReachabilityPolicy is the package default used by
// WaitForSSHReachable for poll cadence, per-probe timeout, and
// consecutive-failure thresholds that classify stuck or broken SSH.
var DefaultSSHReachabilityPolicy = SSHReachabilityPolicy{
	PollInterval:                sshReachablePollInterval,
	ProbeTimeout:                sshProbeAttemptTimeout,
	AuthFailureThreshold:        sshAuthFailureThreshold,
	BannerTimeoutThreshold:      sshBannerTimeoutThreshold,
	NetworkUnreachableThreshold: sshNetworkUnreachableThreshold,
	ProbeTimeoutThreshold:       sshProbeTimeoutThreshold,
}

// FirstContactSSHPolicy is a lenient variant of DefaultSSHReachabilityPolicy
// for the initial "is SSH up yet?" probe after VM creation, where auth/banner/
// network failures are expected while the guest boots. Thresholds are high so
// the context timeout is the only real bound; callers should set a generous
// context (e.g. 20 min) for fresh VMs.
var FirstContactSSHPolicy = SSHReachabilityPolicy{
	PollInterval:                sshReachablePollInterval,
	ProbeTimeout:                sshProbeAttemptTimeout,
	AuthFailureThreshold:        120,
	BannerTimeoutThreshold:      120,
	NetworkUnreachableThreshold: 120,
	ProbeTimeoutThreshold:       120,
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

// sshFailureCategory describes one class of SSH probe failure with its
// matching predicate, associated counter, threshold, and terminal error.
type sshFailureCategory struct {
	label       string
	matches     func(stderr string, err error) bool
	counter     func(c *sshProbeCounters) *int
	threshold   func(p SSHReachabilityPolicy) int
	sentinelErr error
	terminalMsg func(virt Virtctl, count int, detail string) string
}

var sshFailureCategories = []sshFailureCategory{
	{
		label:       "ssh auth not accepted",
		matches:     func(stderr string, _ error) bool { return isSSHAuthenticationFailure(stderr) },
		counter:     func(c *sshProbeCounters) *int { return &c.authFailures },
		threshold:   func(p SSHReachabilityPolicy) int { return p.AuthFailureThreshold },
		sentinelErr: ErrSSHAuthenticationFailed,
		terminalMsg: func(virt Virtctl, count int, detail string) string {
			return fmt.Sprintf("for ssh user %q after %d consecutive attempts (likely stale/missing authorized key): %s",
				sshUserForDiagnostic(virt), count, detail)
		},
	},
	{
		label:       "ssh banner not received",
		matches:     func(stderr string, _ error) bool { return isSSHBannerTimeoutFailure(stderr) },
		counter:     func(c *sshProbeCounters) *int { return &c.bannerTimeoutFailures },
		threshold:   func(p SSHReachabilityPolicy) int { return p.BannerTimeoutThreshold },
		sentinelErr: ErrSSHConnectivityStalled,
		terminalMsg: func(_ Virtctl, count int, detail string) string {
			return fmt.Sprintf("after %d consecutive attempts (guest may be unhealthy and need recreation): %s", count, detail)
		},
	},
	{
		label:       "ssh network not ready",
		matches:     func(stderr string, _ error) bool { return isSSHNetworkUnreachableFailure(stderr) },
		counter:     func(c *sshProbeCounters) *int { return &c.networkUnreachableFailures },
		threshold:   func(p SSHReachabilityPolicy) int { return p.NetworkUnreachableThreshold },
		sentinelErr: ErrSSHConnectivityStalled,
		terminalMsg: func(_ Virtctl, count int, detail string) string {
			return fmt.Sprintf("after %d consecutive attempts (guest network/sshd may be unhealthy and need recreation): %s", count, detail)
		},
	},
	{
		label:       "ssh probe deadline reached",
		matches:     func(_ string, err error) bool { return isSSHProbeTimeoutFailure(err) },
		counter:     func(c *sshProbeCounters) *int { return &c.probeTimeoutFailures },
		threshold:   func(p SSHReachabilityPolicy) int { return p.ProbeTimeoutThreshold },
		sentinelErr: ErrSSHConnectivityStalled,
		terminalMsg: func(_ Virtctl, count int, detail string) string {
			return fmt.Sprintf("after %d consecutive probe timeouts (ssh command appears stuck): %s", count, detail)
		},
	},
}

// sshProbeDecision is the result of one classifyFailure evaluation: human detail and optional terminal error.
type sshProbeDecision struct {
	detail      string
	terminalErr error
}

// runSSHReachabilityProbe runs a minimal SSH command (`true`) under the policy's per-attempt timeout and returns stderr.
func runSSHReachabilityProbe(ctx context.Context, policy SSHReachabilityPolicy, virt Virtctl, namespace, vm string) (stderr string, err error) {
	probeCtx, cancel := context.WithTimeout(ctx, policy.ProbeTimeout)
	defer cancel()
	_, stderr, err = runSSHCommandWithFramework(probeCtx, virt, namespace, vm, sshCommandRunOptions{
		description:            "ssh reachability probe",
		transportRetryAttempts: 1,
	}, "true")
	return strings.TrimSpace(stderr), err
}

// classifyFailure decides whether to keep retrying while the VM is likely still booting
// or to stop early when repeated failures strongly indicate a broken guest.
func (p SSHReachabilityPolicy) classifyFailure(counters *sshProbeCounters, virt Virtctl, err error, stderr string) sshProbeDecision {
	detail := sshProbeFailureDetail(err, stderr)
	for _, cat := range sshFailureCategories {
		if !cat.matches(stderr, err) {
			continue
		}
		// Increment the matched category's counter while resetting all others,
		// so we track consecutive failures of the same kind only.
		ctr := cat.counter(counters)
		prev := *ctr
		counters.resetAll()
		*ctr = prev + 1
		thresh := cat.threshold(p)
		msg := fmt.Sprintf("%s (%d/%d consecutive): %s", cat.label, *ctr, thresh, detail)
		if *ctr >= thresh {
			return sshProbeDecision{
				detail:      msg,
				terminalErr: fmt.Errorf("%w %s", cat.sentinelErr, cat.terminalMsg(virt, *ctr, truncateWaitDetail(detail))),
			}
		}
		return sshProbeDecision{detail: msg}
	}
	counters.resetAll()
	return sshProbeDecision{detail: fmt.Sprintf("ssh not reachable yet: %s", detail)}
}
