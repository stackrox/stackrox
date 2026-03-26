// Package adopt patches ScanSettingBinding resources on each cluster to
// reference the ScanSetting that ACS creates when a scan configuration is
// pushed to Sensor.  This completes the "handover" so the SSB is fully
// managed by ACS going forward.
package adopt

import (
	"context"
	"fmt"
	"time"

	"github.com/stackrox/co-acs-importer/internal/cofetch"
)

// DefaultPollInterval is how often to check for the ScanSetting.
const DefaultPollInterval = 3 * time.Second

// DefaultPollTimeout is the maximum time to wait for the ScanSetting to appear.
const DefaultPollTimeout = 60 * time.Second

// Request describes one SSB that should be adopted after an ACS scan config
// was successfully created.
type Request struct {
	SSBName       string // ScanSettingBinding name (= ACS scan config name)
	SSBNamespace  string
	OldSettingRef string           // current settingsRef.name on the SSB
	ClusterLabel  string           // kubeconfig context name, for logging
	COClient      cofetch.COClient // k8s client scoped to this cluster

	// PreExistingScanSettings is the set of ScanSetting names that existed
	// on this cluster before reconciliation.  If the target name is in this
	// set, adoption is skipped to avoid patching the SSB onto a resource
	// that ACS doesn't control.
	PreExistingScanSettings map[string]bool
}

// Result records the outcome for one adoption request.
type Result struct {
	SSBName      string
	ClusterLabel string
	Adopted      bool   // true if the SSB was patched
	Skipped      bool   // true if settingsRef already correct
	TimedOut     bool   // true if the ScanSetting didn't appear in time
	Err          error  // non-nil on unexpected failure
	Message      string // human-readable description of what happened
}

// Adopter runs the adoption step for a batch of requests.
type Adopter struct {
	PollInterval time.Duration
	PollTimeout  time.Duration
}

// New creates an Adopter with default poll settings.
func New() *Adopter {
	return &Adopter{
		PollInterval: DefaultPollInterval,
		PollTimeout:  DefaultPollTimeout,
	}
}

// Adopt processes a list of adoption requests.  Each request is handled
// independently — a failure or timeout on one cluster does not block others.
func (a *Adopter) Adopt(ctx context.Context, requests []Request) []Result {
	results := make([]Result, 0, len(requests))
	for _, req := range requests {
		results = append(results, a.adoptOne(ctx, req))
	}
	return results
}

func (a *Adopter) adoptOne(ctx context.Context, req Request) Result {
	newSettingName := req.SSBName // ACS creates a ScanSetting with the same name as the scan config

	// IMP-ADOPT-003: skip if already pointing to the right ScanSetting.
	if req.OldSettingRef == newSettingName {
		return Result{
			SSBName:      req.SSBName,
			ClusterLabel: req.ClusterLabel,
			Skipped:      true,
			Message:      fmt.Sprintf("SSB %s/%s already references ScanSetting %q, no patch needed", req.SSBNamespace, req.SSBName, newSettingName),
		}
	}

	// IMP-ADOPT-007: if a ScanSetting with the target name already existed
	// on the cluster before reconciliation, it's a pre-existing resource
	// that would conflict with the ACS-managed one.  Skip adoption to
	// avoid patching the SSB onto a ScanSetting that ACS doesn't control.
	if req.PreExistingScanSettings[newSettingName] {
		return Result{
			SSBName:      req.SSBName,
			ClusterLabel: req.ClusterLabel,
			Skipped:      true,
			Message: fmt.Sprintf("ScanSetting %q already exists on cluster %s but SSB %s/%s references %q; skipping adoption to avoid conflict with pre-existing resource",
				newSettingName, req.ClusterLabel, req.SSBNamespace, req.SSBName, req.OldSettingRef),
		}
	}

	// Poll for the ACS-created ScanSetting to appear on the cluster.
	if err := a.waitForScanSetting(ctx, req.COClient, req.SSBNamespace, newSettingName); err != nil {
		// IMP-ADOPT-004, IMP-ADOPT-005, IMP-ADOPT-006: timeout is a warning, not an error.
		return Result{
			SSBName:      req.SSBName,
			ClusterLabel: req.ClusterLabel,
			TimedOut:     true,
			Message: fmt.Sprintf("timed out waiting for ScanSetting %q to appear on cluster %s; SSB %s/%s was NOT patched (settingsRef still %q)",
				newSettingName, req.ClusterLabel, req.SSBNamespace, req.SSBName, req.OldSettingRef),
		}
	}

	// IMP-ADOPT-001: patch the SSB's settingsRef to point to the new ScanSetting.
	if err := req.COClient.PatchSSBSettingsRef(ctx, req.SSBNamespace, req.SSBName, newSettingName); err != nil {
		return Result{
			SSBName:      req.SSBName,
			ClusterLabel: req.ClusterLabel,
			Err:          err,
			Message: fmt.Sprintf("failed to patch SSB %s/%s settingsRef on cluster %s: %v",
				req.SSBNamespace, req.SSBName, req.ClusterLabel, err),
		}
	}

	return Result{
		SSBName:      req.SSBName,
		ClusterLabel: req.ClusterLabel,
		Adopted:      true,
		Message: fmt.Sprintf("adopted SSB %s/%s on cluster %s: settingsRef changed from %q to %q",
			req.SSBNamespace, req.SSBName, req.ClusterLabel, req.OldSettingRef, newSettingName),
	}
}

// waitForScanSetting polls until the named ScanSetting exists or the timeout expires.
func (a *Adopter) waitForScanSetting(ctx context.Context, client cofetch.COClient, namespace, name string) error {
	deadline := time.After(a.PollTimeout)
	ticker := time.NewTicker(a.PollInterval)
	defer ticker.Stop()

	// Check immediately before first tick.
	if _, err := client.GetScanSetting(ctx, namespace, name); err == nil {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("ScanSetting %q not found after %s", name, a.PollTimeout)
		case <-ticker.C:
			if _, err := client.GetScanSetting(ctx, namespace, name); err == nil {
				return nil
			}
		}
	}
}
