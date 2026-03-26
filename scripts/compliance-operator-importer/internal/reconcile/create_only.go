// Package reconcile implements the reconciliation loop that can either create-only
// or create-or-update scan configurations based on the overwriteExisting setting.
package reconcile

import (
	"context"
	"fmt"
	"time"

	"github.com/stackrox/co-acs-importer/internal/models"
)

// transientStatusCodes is the set of HTTP status codes that should trigger a retry.
// Non-transient codes (400, 401, 403, 404) are NOT in this set and cause immediate failure.
//
// Implements IMP-ERR-001 (retry) and IMP-ERR-002 (no retry).
var transientStatusCodes = map[int]bool{
	429: true,
	502: true,
	503: true,
	504: true,
}

// statusCoder is the interface satisfied by acs.HTTPError (and the test statusError).
// It lets the reconciler inspect the HTTP status without importing the acs package,
// avoiding an import cycle.
type statusCoder interface {
	StatusCode() int
}

// Action records the outcome of a single Apply call.
type Action struct {
	Source          models.ReportItemSource
	ActionType      string // "create" | "skip" | "fail"
	Reason          string
	Attempts        int
	ACSScanConfigID string
	Err             error
	Problem         *models.Problem
}

// Reconciler implements the reconciliation loop.
// When overwriteExisting=false, existing scan names are skipped with a conflict problem.
// When overwriteExisting=true, existing scan names are updated via PUT.
type Reconciler struct {
	client            models.ACSClient
	maxRetries        int
	dryRun            bool
	overwriteExisting bool
}

// NewReconciler creates a Reconciler.
//
//   - client:            ACS API client supporting both POST and PUT operations
//   - maxRetries:        maximum total attempts for a single create/update (must be >= 1)
//   - dryRun:            when true, no POST/PUT is issued; planned actions are still recorded
//   - overwriteExisting: when true, existing configs are updated via PUT instead of skipped
func NewReconciler(client models.ACSClient, maxRetries int, dryRun bool, overwriteExisting bool) *Reconciler {
	if maxRetries < 1 {
		maxRetries = 1
	}
	return &Reconciler{
		client:            client,
		maxRetries:        maxRetries,
		dryRun:            dryRun,
		overwriteExisting: overwriteExisting,
	}
}

// Apply tries to create or update the scan config based on whether scanName exists in existingNames.
//
// Behaviour:
//   - If dryRun=true: records planned action, no POST/PUT is issued.     (IMP-IDEM-004, IMP-IDEM-006)
//   - If scanName exists and overwriteExisting=false: skip + conflict problem. (IMP-IDEM-002, IMP-IDEM-003)
//   - If scanName exists and overwriteExisting=true: update via PUT. (IMP-IDEM-008)
//   - If scanName not exists: create via POST regardless of overwriteExisting. (IMP-IDEM-009)
//   - Transient failures (429,502,503,504): retry with exponential backoff. (IMP-ERR-001)
//   - Non-transient failures (400,401,403,404): record as fail immediately. (IMP-ERR-002)
//
// Exponential backoff: base=500ms, doubles each retry; up to maxRetries total attempts.
// Attempts count is always recorded in the returned Action.
//
// existingNames maps scanName -> configID so we know the ID for PUT operations.
func (r *Reconciler) Apply(
	ctx context.Context,
	payload models.ACSCreatePayload,
	source models.ReportItemSource,
	existingNames map[string]string,
) Action {
	action := Action{Source: source}

	existingID, nameExists := existingNames[payload.ScanName]

	// Handle existing name based on overwriteExisting setting
	if nameExists {
		if !r.overwriteExisting {
			// IMP-IDEM-002: existing name and overwriteExisting=false => skip with conflict problem
			// IMP-IDEM-003: no PUT is attempted when overwriteExisting=false
			problem := &models.Problem{
				Severity:    models.SeverityWarning,
				Category:    models.CategoryConflict,
				ResourceRef: resourceRef(source),
				Description: fmt.Sprintf("scan configuration %q already exists in ACS and will not be updated (create-only mode)", payload.ScanName),
				FixHint:     fmt.Sprintf("Remove the existing ACS scan configuration named %q before re-running, or use --overwrite-existing flag, or rename the ScanSettingBinding to use a different name.", payload.ScanName),
				Skipped:     true,
			}
			action.ActionType = "skip"
			action.Reason = fmt.Sprintf("scan configuration %q already exists in ACS", payload.ScanName)
			action.Problem = problem
			return action
		}

		// IMP-IDEM-008: overwriteExisting=true and name exists => update via PUT
		if r.dryRun {
			action.ActionType = "update"
			action.ACSScanConfigID = existingID
			action.Reason = "dry-run: would PUT /v2/compliance/scan/configurations/" + existingID
			action.Attempts = 0
			return action
		}

		// Perform update with retry logic
		var (
			lastErr error
			delay   = 500 * time.Millisecond
		)

		for attempt := 1; attempt <= r.maxRetries; attempt++ {
			action.Attempts = attempt

			lastErr = r.client.UpdateScanConfiguration(ctx, existingID, payload)
			if lastErr == nil {
				action.ActionType = "update"
				action.ACSScanConfigID = existingID
				action.Reason = "scan configuration updated successfully"
				return action
			}

			// Check if the error is transient (eligible for retry)
			if sc, ok := asStatusCoder(lastErr); ok {
				code := sc.StatusCode()
				if !transientStatusCodes[code] {
					// Non-transient: fail immediately, no more attempts
					action.ActionType = "fail"
					action.Reason = fmt.Sprintf("non-transient HTTP %d error updating scan configuration: %v", code, lastErr)
					action.Err = lastErr
					return action
				}
			}

			// Do not sleep after the last attempt
			if attempt < r.maxRetries {
				select {
				case <-ctx.Done():
					action.ActionType = "fail"
					action.Reason = "context cancelled during retry backoff"
					action.Err = ctx.Err()
					return action
				case <-time.After(delay):
				}
				delay *= 2
			}
		}

		// Exhausted all retries for update
		action.ActionType = "fail"
		action.Reason = fmt.Sprintf("failed to update after %d attempt(s): %v", action.Attempts, lastErr)
		action.Err = lastErr
		return action
	}

	// IMP-IDEM-009: name not exists => create via POST regardless of overwriteExisting flag
	// IMP-IDEM-004: dry-run => record planned action, do not POST
	// IMP-IDEM-006: planned action "create" is still recorded
	if r.dryRun {
		action.ActionType = "create"
		action.Reason = "dry-run: would POST /v2/compliance/scan/configurations"
		action.Attempts = 0
		return action
	}

	// IMP-IDEM-001: POST /v2/compliance/scan/configurations when name not found
	// IMP-ERR-001: retry on transient errors with exponential backoff
	// IMP-ERR-002: no retry on non-transient errors
	var (
		lastErr error
		id      string
		delay   = 500 * time.Millisecond
	)

	for attempt := 1; attempt <= r.maxRetries; attempt++ {
		action.Attempts = attempt

		id, lastErr = r.client.CreateScanConfiguration(ctx, payload)
		if lastErr == nil {
			action.ActionType = "create"
			action.ACSScanConfigID = id
			action.Reason = "scan configuration created successfully"
			return action
		}

		// Check if the error is transient (eligible for retry)
		if sc, ok := asStatusCoder(lastErr); ok {
			code := sc.StatusCode()
			if !transientStatusCodes[code] {
				// Non-transient: fail immediately, no more attempts
				action.ActionType = "fail"
				action.Reason = fmt.Sprintf("non-transient HTTP %d error creating scan configuration: %v", code, lastErr)
				action.Err = lastErr
				return action
			}
		} else {
			// Unknown error type (e.g. network error): treat as transient and retry
		}

		// Do not sleep after the last attempt
		if attempt < r.maxRetries {
			select {
			case <-ctx.Done():
				action.ActionType = "fail"
				action.Reason = "context cancelled during retry backoff"
				action.Err = ctx.Err()
				return action
			case <-time.After(delay):
			}
			delay *= 2
		}
	}

	// Exhausted all retries
	action.ActionType = "fail"
	action.Reason = fmt.Sprintf("failed after %d attempt(s): %v", action.Attempts, lastErr)
	action.Err = lastErr
	return action
}

// resourceRef formats the source as "namespace/bindingName" for use in Problem.ResourceRef.
func resourceRef(source models.ReportItemSource) string {
	if source.Namespace == "" {
		return source.BindingName
	}
	return source.Namespace + "/" + source.BindingName
}

// asStatusCoder attempts to extract a statusCoder from err using errors.As-style
// type assertion. It handles both direct and wrapped errors.
func asStatusCoder(err error) (statusCoder, bool) {
	// Direct type assertion first (most common path)
	if sc, ok := err.(statusCoder); ok {
		return sc, true
	}
	// Unwrap chain
	type unwrapper interface{ Unwrap() error }
	for err != nil {
		if sc, ok := err.(statusCoder); ok {
			return sc, true
		}
		uw, ok := err.(unwrapper)
		if !ok {
			break
		}
		err = uw.Unwrap()
	}
	return nil, false
}
