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

// sshFailureCategory describes one class of SSH probe failure with its
// matching predicate, associated counter, threshold, and terminal error.
type sshFailureCategory struct {
	label       string
	matches     func(stderr string, err error) bool
	counter     func(c *sshProbeCounters) *int
	threshold   func(p sshReachabilityPolicy) int
	sentinelErr error
	terminalMsg func(virt Virtctl, count int, detail string) string
}

var sshFailureCategories = []sshFailureCategory{
	{
		label:       "ssh authentication failed",
		matches:     func(stderr string, _ error) bool { return isSSHAuthenticationFailure(stderr) },
		counter:     func(c *sshProbeCounters) *int { return &c.authFailures },
		threshold:   func(p sshReachabilityPolicy) int { return p.authFailureThreshold },
		sentinelErr: ErrSSHAuthenticationFailed,
		terminalMsg: func(virt Virtctl, count int, detail string) string {
			return fmt.Sprintf("for ssh user %q after %d consecutive attempts (likely stale/missing authorized key): %s",
				sshUserForDiagnostic(virt), count, detail)
		},
	},
	{
		label:       "ssh banner timeout",
		matches:     func(stderr string, _ error) bool { return isSSHBannerTimeoutFailure(stderr) },
		counter:     func(c *sshProbeCounters) *int { return &c.bannerTimeoutFailures },
		threshold:   func(p sshReachabilityPolicy) int { return p.bannerTimeoutThreshold },
		sentinelErr: ErrSSHConnectivityStalled,
		terminalMsg: func(_ Virtctl, count int, detail string) string {
			return fmt.Sprintf("after %d consecutive attempts (guest may be unhealthy and need recreation): %s", count, detail)
		},
	},
	{
		label:       "ssh network unavailable",
		matches:     func(stderr string, _ error) bool { return isSSHNetworkUnreachableFailure(stderr) },
		counter:     func(c *sshProbeCounters) *int { return &c.networkUnreachableFailures },
		threshold:   func(p sshReachabilityPolicy) int { return p.networkUnreachableThreshold },
		sentinelErr: ErrSSHConnectivityStalled,
		terminalMsg: func(_ Virtctl, count int, detail string) string {
			return fmt.Sprintf("after %d consecutive attempts (guest network/sshd may be unhealthy and need recreation): %s", count, detail)
		},
	},
	{
		label:       "ssh probe timed out",
		matches:     func(_ string, err error) bool { return isSSHProbeTimeoutFailure(err) },
		counter:     func(c *sshProbeCounters) *int { return &c.probeTimeoutFailures },
		threshold:   func(p sshReachabilityPolicy) int { return p.probeTimeoutThreshold },
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
	for _, cat := range sshFailureCategories {
		if !cat.matches(stderr, err) {
			continue
		}
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
