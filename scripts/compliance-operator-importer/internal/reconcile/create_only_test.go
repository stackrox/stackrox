package reconcile_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/stackrox/co-acs-importer/internal/models"
	"github.com/stackrox/co-acs-importer/internal/reconcile"
)

// ---------------------------------------------------------------------------
// Mock ACS client
// ---------------------------------------------------------------------------

// mockACSClient is a test double that records every call and allows the caller
// to inject per-call responses via the nextResponses queue.
//
// IMP-IDEM-003: The mock only implements POST (via CreateScanConfiguration).
// There is no Put/Update method. If one were added to ACSClient, this struct
// would fail to compile unless the method were added here too, making the
// violation immediately visible.
type mockACSClient struct {
	// createResponses is consumed in order; each entry is either nil (success)
	// or an error. Use statusError to encode HTTP status codes.
	createResponses []error

	// callCount tracks total calls to CreateScanConfiguration.
	callCount atomic.Int32

	// recordedIDCounter is used to return unique IDs on success.
	idCounter atomic.Int32

	// listConfigs is the fixed list returned by ListScanConfigurations.
	listConfigs []models.ACSConfigSummary
}

// statusError wraps an HTTP status code so the reconciler can distinguish
// transient (429/502/503/504) from non-transient (400/401/403/404) failures.
type statusError struct {
	code int
}

func (e *statusError) Error() string   { return fmt.Sprintf("HTTP %d", e.code) }
func (e *statusError) StatusCode() int { return e.code }

func (m *mockACSClient) Preflight(_ context.Context) error { return nil }

func (m *mockACSClient) ListScanConfigurations(_ context.Context) ([]models.ACSConfigSummary, error) {
	return m.listConfigs, nil
}

func (m *mockACSClient) CreateScanConfiguration(_ context.Context, _ models.ACSCreatePayload) (string, error) {
	idx := int(m.callCount.Add(1)) - 1
	if idx < len(m.createResponses) {
		if err := m.createResponses[idx]; err != nil {
			return "", err
		}
	}
	id := fmt.Sprintf("created-id-%d", m.idCounter.Add(1))
	return id, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func defaultSource() models.ReportItemSource {
	return models.ReportItemSource{
		Namespace:       "openshift-compliance",
		BindingName:     "cis-weekly",
		ScanSettingName: "default-auto-apply",
	}
}

func defaultPayload(scanName string) models.ACSCreatePayload {
	return models.ACSCreatePayload{
		ScanName: scanName,
		ScanConfig: models.ACSBaseScanConfig{
			Profiles:    []string{"ocp4-cis"},
			Description: "test",
		},
		Clusters: []string{"cluster-a"},
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// IMP-IDEM-001: non-existing name => POST called, action="create"
func TestApply_IMP_IDEM_001_NewName_CreatesConfig(t *testing.T) {
	mock := &mockACSClient{}
	r := reconcile.NewReconciler(mock, 3, false)

	action := r.Apply(context.Background(), defaultPayload("new-scan"), defaultSource(), map[string]bool{})

	if action.ActionType != "create" {
		t.Errorf("IMP-IDEM-001: expected action 'create', got %q", action.ActionType)
	}
	if action.ACSScanConfigID == "" {
		t.Error("IMP-IDEM-001: expected non-empty ACSScanConfigID after create")
	}
	if action.Err != nil {
		t.Errorf("IMP-IDEM-001: unexpected error: %v", action.Err)
	}
	if mock.callCount.Load() != 1 {
		t.Errorf("IMP-IDEM-001: expected 1 POST call, got %d", mock.callCount.Load())
	}
}

// IMP-IDEM-002: existing name => action="skip", Problem.Category=conflict, FixHint non-empty
func TestApply_IMP_IDEM_002_ExistingName_SkipsWithConflictProblem(t *testing.T) {
	mock := &mockACSClient{}
	r := reconcile.NewReconciler(mock, 3, false)

	existing := map[string]bool{"cis-weekly": true}
	action := r.Apply(context.Background(), defaultPayload("cis-weekly"), defaultSource(), existing)

	if action.ActionType != "skip" {
		t.Errorf("IMP-IDEM-002: expected action 'skip', got %q", action.ActionType)
	}
	if action.Problem == nil {
		t.Fatal("IMP-IDEM-002: expected Problem to be non-nil for skipped-existing")
	}
	if action.Problem.Category != models.CategoryConflict {
		t.Errorf("IMP-IDEM-002: expected Problem.Category 'conflict', got %q", action.Problem.Category)
	}
	if action.Problem.FixHint == "" {
		t.Error("IMP-IDEM-002: expected non-empty Problem.FixHint")
	}
	if action.Reason == "" {
		t.Error("IMP-IDEM-002: expected non-empty Reason")
	}
	// "already exists" must appear in the reason (per spec)
	if !containsSubstring(action.Reason, "already exists") {
		t.Errorf("IMP-IDEM-002: Reason must include 'already exists', got %q", action.Reason)
	}
}

// IMP-IDEM-003: verify no PUT ever called (mock records method; ACSClient has no Put)
func TestApply_IMP_IDEM_003_NeverCallsPUT(t *testing.T) {
	// The mockACSClient deliberately has no Put/Update method.
	// It only satisfies models.ACSClient which defines:
	//   Preflight, ListScanConfigurations, CreateScanConfiguration (POST only).
	// If a PUT-based method existed in the interface, the mock would fail to compile.
	mock := &mockACSClient{}
	r := reconcile.NewReconciler(mock, 3, false)

	// Run multiple scenarios - none should trigger a PUT
	for _, scanName := range []string{"new-scan-1", "new-scan-2"} {
		_ = r.Apply(context.Background(), defaultPayload(scanName), defaultSource(), map[string]bool{})
	}
	// existing name - should skip, not PUT
	_ = r.Apply(context.Background(), defaultPayload("existing"), defaultSource(), map[string]bool{"existing": true})

	// The mock only has CreateScanConfiguration (POST). callCount reflects POST calls only.
	// 2 creates + 1 skip = 2 POST calls total (no PUT possible).
	if mock.callCount.Load() != 2 {
		t.Errorf("IMP-IDEM-003: expected exactly 2 POST calls (2 creates, 0 PUT), got %d", mock.callCount.Load())
	}
}

// IMP-IDEM-004: dryRun=true => no POST
func TestApply_IMP_IDEM_004_DryRun_NoPost(t *testing.T) {
	mock := &mockACSClient{}
	r := reconcile.NewReconciler(mock, 3, true) // dryRun=true

	_ = r.Apply(context.Background(), defaultPayload("new-scan"), defaultSource(), map[string]bool{})

	if mock.callCount.Load() != 0 {
		t.Errorf("IMP-IDEM-004: expected 0 POST calls in dry-run mode, got %d", mock.callCount.Load())
	}
}

// IMP-IDEM-006: dryRun => action="create" still recorded as planned
func TestApply_IMP_IDEM_006_DryRun_PlannedCreateRecorded(t *testing.T) {
	mock := &mockACSClient{}
	r := reconcile.NewReconciler(mock, 3, true) // dryRun=true

	action := r.Apply(context.Background(), defaultPayload("new-scan"), defaultSource(), map[string]bool{})

	if action.ActionType != "create" {
		t.Errorf("IMP-IDEM-006: dry-run planned action should be 'create', got %q", action.ActionType)
	}
}

// IMP-IDEM-007: dryRun => problems still populated for problematic resources
func TestApply_IMP_IDEM_007_DryRun_ProblemsStillPopulated(t *testing.T) {
	mock := &mockACSClient{}
	r := reconcile.NewReconciler(mock, 3, true) // dryRun=true

	existing := map[string]bool{"cis-weekly": true}
	action := r.Apply(context.Background(), defaultPayload("cis-weekly"), defaultSource(), existing)

	if action.Problem == nil {
		t.Fatal("IMP-IDEM-007: expected Problem to be populated even in dry-run mode")
	}
	if action.Problem.Category != models.CategoryConflict {
		t.Errorf("IMP-IDEM-007: expected conflict problem in dry-run, got %q", action.Problem.Category)
	}
}

// IMP-ERR-001: 429 first 2 times then 200 => 3 total attempts
func TestApply_IMP_ERR_001_Retry429_ThenSuccess(t *testing.T) {
	mock := &mockACSClient{
		createResponses: []error{
			&statusError{code: 429},
			&statusError{code: 429},
			nil, // 3rd attempt succeeds
		},
	}
	r := reconcile.NewReconciler(mock, 5, false)

	action := r.Apply(context.Background(), defaultPayload("new-scan"), defaultSource(), map[string]bool{})

	if action.ActionType != "create" {
		t.Errorf("IMP-ERR-001: expected action 'create' after retry success, got %q", action.ActionType)
	}
	if action.Attempts != 3 {
		t.Errorf("IMP-ERR-001: expected 3 total attempts, got %d", action.Attempts)
	}
	if action.Err != nil {
		t.Errorf("IMP-ERR-001: expected nil error after eventual success, got %v", action.Err)
	}
}

// IMP-ERR-001: Retry on transient errors 502, 503, 504
func TestApply_IMP_ERR_001_Retry5xx_ThenSuccess(t *testing.T) {
	for _, code := range []int{502, 503, 504} {
		code := code
		t.Run(fmt.Sprintf("HTTP%d", code), func(t *testing.T) {
			mock := &mockACSClient{
				createResponses: []error{
					&statusError{code: code},
					&statusError{code: code},
					nil, // 3rd succeeds
				},
			}
			r := reconcile.NewReconciler(mock, 5, false)

			action := r.Apply(context.Background(), defaultPayload("new-scan"), defaultSource(), map[string]bool{})

			if action.ActionType != "create" {
				t.Errorf("IMP-ERR-001: HTTP %d - expected 'create', got %q", code, action.ActionType)
			}
			if action.Attempts != 3 {
				t.Errorf("IMP-ERR-001: HTTP %d - expected 3 attempts, got %d", code, action.Attempts)
			}
		})
	}
}

// IMP-ERR-002: 400 => 1 attempt only, action="fail"
func TestApply_IMP_ERR_002_NonTransient400_NoRetry(t *testing.T) {
	for _, code := range []int{400, 401, 403, 404} {
		code := code
		t.Run(fmt.Sprintf("HTTP%d", code), func(t *testing.T) {
			mock := &mockACSClient{
				createResponses: []error{
					&statusError{code: code},
				},
			}
			r := reconcile.NewReconciler(mock, 5, false)

			action := r.Apply(context.Background(), defaultPayload("new-scan"), defaultSource(), map[string]bool{})

			if action.ActionType != "fail" {
				t.Errorf("IMP-ERR-002: HTTP %d - expected action 'fail', got %q", code, action.ActionType)
			}
			if action.Attempts != 1 {
				t.Errorf("IMP-ERR-002: HTTP %d - expected exactly 1 attempt (no retry), got %d", code, action.Attempts)
			}
			if mock.callCount.Load() != 1 {
				t.Errorf("IMP-ERR-002: HTTP %d - expected 1 POST call, got %d", code, mock.callCount.Load())
			}
			if action.Err == nil {
				t.Errorf("IMP-ERR-002: HTTP %d - expected non-nil error", code)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Compile-time check: mockACSClient satisfies models.ACSClient
// This fails to compile if models.ACSClient gains any method not implemented
// by the mock, making interface drift immediately visible.
// ---------------------------------------------------------------------------
var _ models.ACSClient = (*mockACSClient)(nil)

// ---------------------------------------------------------------------------
// Utility
// ---------------------------------------------------------------------------

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}

// Verify containsSubstring works correctly
var _ = func() bool {
	if !containsSubstring("scan already exists in ACS", "already exists") {
		panic("containsSubstring broken")
	}
	return true
}()

// errorIs is a helper for unwrapping statusError from wrapped errors.
func errorIs(err error, code int) bool {
	var se *statusError
	return errors.As(err, &se) && se.code == code
}

// keep errorIs in use
var _ = errorIs
