package run_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stackrox/co-acs-importer/internal/cofetch"
	"github.com/stackrox/co-acs-importer/internal/models"
	"github.com/stackrox/co-acs-importer/internal/run"
)

// ---------------------------------------------------------------------------
// Mock: models.ACSClient
// ---------------------------------------------------------------------------

type mockACSClient struct {
	listErr     error
	listResult  []models.ACSConfigSummary
	createErr   error
	createID    string
	createCalls int
}

func (m *mockACSClient) Preflight(_ context.Context) error { return nil }

func (m *mockACSClient) ListScanConfigurations(_ context.Context) ([]models.ACSConfigSummary, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listResult, nil
}

func (m *mockACSClient) CreateScanConfiguration(_ context.Context, _ models.ACSCreatePayload) (string, error) {
	m.createCalls++
	if m.createErr != nil {
		return "", m.createErr
	}
	id := m.createID
	if id == "" {
		id = fmt.Sprintf("new-id-%d", m.createCalls)
	}
	return id, nil
}

func (m *mockACSClient) UpdateScanConfiguration(_ context.Context, _ string, _ models.ACSCreatePayload) error {
	// For now, this is a no-op in run tests since we focus on create-only mode.
	// Update-specific tests are in reconcile_test.go.
	return nil
}

func (m *mockACSClient) ListClusters(_ context.Context) ([]models.ACSClusterInfo, error) {
	// Not used in run tests, return empty list
	return []models.ACSClusterInfo{}, nil
}

// Compile-time check: mockACSClient satisfies models.ACSClient.
var _ models.ACSClient = (*mockACSClient)(nil)

// ---------------------------------------------------------------------------
// Mock: cofetch.COClient
// ---------------------------------------------------------------------------

type mockCOClient struct {
	bindings    []cofetch.ScanSettingBinding
	listErr     error
	scanSetting *cofetch.ScanSetting
	settingErr  error
}

func (m *mockCOClient) ListScanSettingBindings(_ context.Context) ([]cofetch.ScanSettingBinding, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.bindings, nil
}

func (m *mockCOClient) GetScanSetting(_ context.Context, _, _ string) (*cofetch.ScanSetting, error) {
	if m.settingErr != nil {
		return nil, m.settingErr
	}
	return m.scanSetting, nil
}

// Compile-time check: mockCOClient satisfies cofetch.COClient.
var _ cofetch.COClient = (*mockCOClient)(nil)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// httpStatusError lets the reconciler identify transient vs. non-transient codes.
type httpStatusError struct {
	code int
}

func (e *httpStatusError) Error() string   { return fmt.Sprintf("HTTP %d", e.code) }
func (e *httpStatusError) StatusCode() int { return e.code }

// baseConfig returns a valid Config for most tests.
func baseConfig() *models.Config {
	return &models.Config{
		ACSEndpoint:  "https://acs.example.com",
		ACSClusterID: "cluster-a",
		CONamespace:  "openshift-compliance",
		MaxRetries:   1,
	}
}

// goodBinding returns a ScanSettingBinding that maps cleanly to an ACS payload.
func goodBinding(name string) cofetch.ScanSettingBinding {
	return cofetch.ScanSettingBinding{
		Namespace:       "openshift-compliance",
		Name:            name,
		ScanSettingName: "default-auto-apply",
		Profiles: []cofetch.ProfileRef{
			{Name: "ocp4-cis", Kind: "Profile"},
		},
	}
}

// goodScanSetting returns a ScanSetting with a valid daily cron schedule.
func goodScanSetting() *cofetch.ScanSetting {
	return &cofetch.ScanSetting{
		Namespace: "openshift-compliance",
		Name:      "default-auto-apply",
		Schedule:  "0 1 * * *",
	}
}

// runWithCapture executes Run and captures all printed output.
func runWithCapture(t *testing.T, cfg *models.Config, acs models.ACSClient, co cofetch.COClient) (int, string) {
	t.Helper()
	var buf bytes.Buffer
	r := run.NewRunner(cfg, acs, co).WithOutput(&buf)
	code := r.Run(context.Background())
	return code, buf.String()
}

// ---------------------------------------------------------------------------
// Tests: exit codes (IMP-CLI-017, IMP-CLI-018, IMP-CLI-019, IMP-ERR-003)
// ---------------------------------------------------------------------------

// TestIMP_CLI_017_AllSuccessExitZero verifies that when all bindings are
// created successfully the runner returns exit code 0.
// Requirements: IMP-CLI-017, IMP-ERR-003.
func TestIMP_CLI_017_AllSuccessExitZero(t *testing.T) {
	acsClient := &mockACSClient{} // no existing configs, create succeeds
	coClient := &mockCOClient{
		bindings:    []cofetch.ScanSettingBinding{goodBinding("cis-weekly")},
		scanSetting: goodScanSetting(),
	}

	code, _ := runWithCapture(t, baseConfig(), acsClient, coClient)

	if code != run.ExitSuccess {
		t.Errorf("IMP-CLI-017: expected exit code %d (success), got %d", run.ExitSuccess, code)
	}
}

// TestIMP_CLI_017_EmptyBindingListExitZero verifies that an empty binding
// list (nothing to import) also produces exit code 0.
// Requirement: IMP-CLI-017.
func TestIMP_CLI_017_EmptyBindingListExitZero(t *testing.T) {
	acsClient := &mockACSClient{}
	coClient := &mockCOClient{bindings: []cofetch.ScanSettingBinding{}}

	code, _ := runWithCapture(t, baseConfig(), acsClient, coClient)

	if code != run.ExitSuccess {
		t.Errorf("IMP-CLI-017: expected exit code %d for empty run, got %d", run.ExitSuccess, code)
	}
}

// TestIMP_CLI_018_ListACSConfigsFatalExitOne verifies that a fatal failure
// when listing ACS scan configurations returns exit code 1.
// Requirements: IMP-CLI-018, IMP-ERR-003.
func TestIMP_CLI_018_ListACSConfigsFatalExitOne(t *testing.T) {
	acsClient := &mockACSClient{listErr: errors.New("ACS unreachable")}
	coClient := &mockCOClient{}

	code, output := runWithCapture(t, baseConfig(), acsClient, coClient)

	if code != run.ExitFatalError {
		t.Errorf("IMP-CLI-018: expected exit code %d (fatal), got %d", run.ExitFatalError, code)
	}
	if !strings.Contains(output, "✗") {
		t.Errorf("IMP-CLI-018: expected failure marker in output, got: %q", output)
	}
}

// TestIMP_CLI_018_ListBindingsFatalExitOne verifies that a fatal failure
// when listing CO ScanSettingBindings returns exit code 1.
// Requirements: IMP-CLI-018, IMP-ERR-003.
func TestIMP_CLI_018_ListBindingsFatalExitOne(t *testing.T) {
	acsClient := &mockACSClient{}
	coClient := &mockCOClient{listErr: errors.New("k8s unreachable")}

	code, output := runWithCapture(t, baseConfig(), acsClient, coClient)

	if code != run.ExitFatalError {
		t.Errorf("IMP-CLI-018: expected exit code %d (fatal), got %d", run.ExitFatalError, code)
	}
	if !strings.Contains(output, "✗") {
		t.Errorf("IMP-CLI-018: expected failure marker in output, got: %q", output)
	}
}

// TestIMP_CLI_019_SomeFailedExitTwo verifies that when at least one binding
// fails the runner returns exit code 2.
// Requirements: IMP-CLI-019, IMP-ERR-003.
func TestIMP_CLI_019_SomeFailedExitTwo(t *testing.T) {
	// Inject a non-transient 400 error so the binding fails without retry.
	acsClient := &mockACSClient{
		createErr: &httpStatusError{code: 400},
	}
	coClient := &mockCOClient{
		bindings:    []cofetch.ScanSettingBinding{goodBinding("cis-weekly")},
		scanSetting: goodScanSetting(),
	}

	code, _ := runWithCapture(t, baseConfig(), acsClient, coClient)

	if code != run.ExitPartialError {
		t.Errorf("IMP-CLI-019: expected exit code %d (partial), got %d", run.ExitPartialError, code)
	}
}

// TestIMP_CLI_019_MissingScanSettingExitTwo verifies that a missing ScanSetting
// causes a binding-level failure that results in exit code 2.
// Requirements: IMP-CLI-019, IMP-ERR-003.
func TestIMP_CLI_019_MissingScanSettingExitTwo(t *testing.T) {
	acsClient := &mockACSClient{}
	coClient := &mockCOClient{
		bindings:   []cofetch.ScanSettingBinding{goodBinding("broken")},
		settingErr: errors.New("ScanSetting not found"),
	}

	code, _ := runWithCapture(t, baseConfig(), acsClient, coClient)

	if code != run.ExitPartialError {
		t.Errorf("IMP-CLI-019: expected exit code %d (partial), got %d", run.ExitPartialError, code)
	}
}

// TestIMP_ERR_003_ExitCodesMapCorrectly exercises all three exit code paths in
// a single test to confirm the mapping is exact.
// Requirement: IMP-ERR-003.
func TestIMP_ERR_003_ExitCodesMapCorrectly(t *testing.T) {
	tests := []struct {
		name     string
		wantCode int
		acs      *mockACSClient
		co       *mockCOClient
	}{
		{
			name:     "all_successful",
			wantCode: run.ExitSuccess,
			acs:      &mockACSClient{},
			co: &mockCOClient{
				bindings:    []cofetch.ScanSettingBinding{goodBinding("ok")},
				scanSetting: goodScanSetting(),
			},
		},
		{
			name:     "fatal_acs_list",
			wantCode: run.ExitFatalError,
			acs:      &mockACSClient{listErr: errors.New("down")},
			co:       &mockCOClient{},
		},
		{
			name:     "partial_binding_failure",
			wantCode: run.ExitPartialError,
			acs:      &mockACSClient{createErr: &httpStatusError{code: 400}},
			co: &mockCOClient{
				bindings:    []cofetch.ScanSettingBinding{goodBinding("fail")},
				scanSetting: goodScanSetting(),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, _ := runWithCapture(t, baseConfig(), tc.acs, tc.co)
			if code != tc.wantCode {
				t.Errorf("IMP-ERR-003: %s: expected exit code %d, got %d", tc.name, tc.wantCode, code)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tests: console output (IMP-CLI-020)
// ---------------------------------------------------------------------------

// TestIMP_CLI_020_ConsoleSummaryIncludesAllCounters verifies that the console
// summary contains discovered, created, skipped, and failed counts.
// Requirement: IMP-CLI-020.
func TestIMP_CLI_020_ConsoleSummaryIncludesAllCounters(t *testing.T) {
	// Two bindings: one creates, one is skipped because it already exists.
	acsClient := &mockACSClient{
		listResult: []models.ACSConfigSummary{
			{ID: "existing-id", ScanName: "existing-scan"},
		},
	}
	coClient := &mockCOClient{
		bindings: []cofetch.ScanSettingBinding{
			goodBinding("new-scan"),
			goodBinding("existing-scan"), // will be skipped
		},
		scanSetting: goodScanSetting(),
	}

	_, output := runWithCapture(t, baseConfig(), acsClient, coClient)

	requiredPhrases := []string{
		"discovered:",
		"created:",
		"skipped:",
		"failed:",
	}
	for _, phrase := range requiredPhrases {
		if !strings.Contains(output, phrase) {
			t.Errorf("IMP-CLI-020: output missing %q\nGot:\n%s", phrase, output)
		}
	}
}

// TestIMP_CLI_020_DryRunLabelInSummary verifies that the summary includes
// the dry-run indicator.
// Requirement: IMP-CLI-020.
func TestIMP_CLI_020_DryRunLabelInSummary(t *testing.T) {
	cfg := baseConfig()
	cfg.DryRun = true

	coClient := &mockCOClient{
		bindings:    []cofetch.ScanSettingBinding{goodBinding("cis-weekly")},
		scanSetting: goodScanSetting(),
	}

	_, output := runWithCapture(t, cfg, &mockACSClient{}, coClient)

	if !strings.Contains(output, "dry-run") {
		t.Errorf("IMP-CLI-020: expected 'dry-run' in output, got:\n%s", output)
	}
}

// TestIMP_CLI_020_NonDryRunLabelInSummary verifies the non-dry-run label.
// Requirement: IMP-CLI-020.
func TestIMP_CLI_020_NonDryRunLabelInSummary(t *testing.T) {
	cfg := baseConfig()
	cfg.DryRun = false

	coClient := &mockCOClient{
		bindings:    []cofetch.ScanSettingBinding{goodBinding("cis-weekly")},
		scanSetting: goodScanSetting(),
	}

	_, output := runWithCapture(t, cfg, &mockACSClient{}, coClient)

	if !strings.Contains(output, "live") {
		t.Errorf("IMP-CLI-020: expected 'live' in output, got:\n%s", output)
	}
}

// TestIMP_CLI_020_CorrectCountsInSummary verifies that counts reported in the
// console summary are numerically correct.
// Requirement: IMP-CLI-020.
func TestIMP_CLI_020_CorrectCountsInSummary(t *testing.T) {
	// Arrange: 3 bindings discovered, 2 create, 1 skipped (existing).
	acsClient := &mockACSClient{
		listResult: []models.ACSConfigSummary{
			{ID: "id-existing", ScanName: "scan-c"},
		},
	}
	coClient := &mockCOClient{
		bindings: []cofetch.ScanSettingBinding{
			goodBinding("scan-a"),
			goodBinding("scan-b"),
			goodBinding("scan-c"), // exists => skip
		},
		scanSetting: goodScanSetting(),
	}

	_, output := runWithCapture(t, baseConfig(), acsClient, coClient)

	if !strings.Contains(output, "discovered: 3") {
		t.Errorf("IMP-CLI-020: expected 'discovered: 3' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "created: 2") {
		t.Errorf("IMP-CLI-020: expected 'created: 2' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "skipped: 1") {
		t.Errorf("IMP-CLI-020: expected 'skipped: 1' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "failed: 0") {
		t.Errorf("IMP-CLI-020: expected 'failed: 0' in output, got:\n%s", output)
	}
}

// ---------------------------------------------------------------------------
// Tests: API error => problem recorded (IMP-ERR-004)
// ---------------------------------------------------------------------------

// TestIMP_ERR_004_APIErrorRecordedAsProblem verifies that a non-transient API
// error causes the binding to be skipped and recorded as a problem in the report.
// The report's failed count must reflect the failure.
// Requirements: IMP-ERR-004, IMP-CLI-022.
func TestIMP_ERR_004_APIErrorRecordedAsProblem(t *testing.T) {
	acsClient := &mockACSClient{
		createErr: &httpStatusError{code: 400},
	}
	coClient := &mockCOClient{
		bindings:    []cofetch.ScanSettingBinding{goodBinding("bad-scan")},
		scanSetting: goodScanSetting(),
	}
	cfg := baseConfig()
	cfg.MaxRetries = 1

	code, output := runWithCapture(t, cfg, acsClient, coClient)

	// Exit code must be partial failure (IMP-ERR-003).
	if code != run.ExitPartialError {
		t.Errorf("IMP-ERR-004: expected exit code %d (partial), got %d", run.ExitPartialError, code)
	}
	// Console summary must show 1 failed.
	if !strings.Contains(output, "failed: 1") {
		t.Errorf("IMP-ERR-004: expected 'failed: 1' in output, got:\n%s", output)
	}
}

// TestIMP_ERR_004_MissingScanSettingRecordedAsProblem verifies that a missing
// ScanSetting is treated as a binding-level failure and recorded.
// Requirements: IMP-ERR-004, IMP-CLI-022.
func TestIMP_ERR_004_MissingScanSettingRecordedAsProblem(t *testing.T) {
	acsClient := &mockACSClient{}
	coClient := &mockCOClient{
		bindings: []cofetch.ScanSettingBinding{
			goodBinding("broken"),
			goodBinding("ok"),
		},
		scanSetting: goodScanSetting(),
	}

	// Fail GetScanSetting on the first call (for "broken"), succeed on the second ("ok").
	coClient2 := &selectiveCOClientByOrder{
		base:       coClient,
		failAtCall: 1,
		failErr:    errors.New("ScanSetting not found"),
	}

	code, output := runWithCapture(t, baseConfig(), acsClient, coClient2)

	// Partial failure: one succeeded, one failed.
	if code != run.ExitPartialError {
		t.Errorf("IMP-ERR-004: expected exit code %d (partial), got %d", run.ExitPartialError, code)
	}
	if !strings.Contains(output, "failed: 1") {
		t.Errorf("IMP-ERR-004: expected 'failed: 1' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "created: 1") {
		t.Errorf("IMP-ERR-004: expected 'created: 1' in output, got:\n%s", output)
	}
}

// selectiveCOClientByBinding wraps COClient to fail GetScanSetting for a
// specific binding name by inspecting which binding is being processed.
// Since GetScanSetting doesn't receive the binding name, we use a counter-based
// approach: the first call goes to the first binding, etc.
type selectiveCOClientByOrder struct {
	base       *mockCOClient
	failAtCall int // 1-based; call index that should fail
	callCount  int
	failErr    error
}

func (s *selectiveCOClientByOrder) ListScanSettingBindings(ctx context.Context) ([]cofetch.ScanSettingBinding, error) {
	return s.base.ListScanSettingBindings(ctx)
}

func (s *selectiveCOClientByOrder) GetScanSetting(ctx context.Context, namespace, name string) (*cofetch.ScanSetting, error) {
	s.callCount++
	if s.callCount == s.failAtCall {
		return nil, s.failErr
	}
	return s.base.GetScanSetting(ctx, namespace, name)
}

// ---------------------------------------------------------------------------
// Tests: dry-run mode (IMP-IDEM-004..007)
// ---------------------------------------------------------------------------

// TestIMP_CLI_007_DryRunNoCreates verifies that no ACS create calls are made
// in dry-run mode.
// Requirement: IMP-CLI-007.
func TestIMP_CLI_007_DryRunNoCreates(t *testing.T) {
	acsClient := &mockACSClient{}
	coClient := &mockCOClient{
		bindings:    []cofetch.ScanSettingBinding{goodBinding("cis-weekly")},
		scanSetting: goodScanSetting(),
	}
	cfg := baseConfig()
	cfg.DryRun = true

	runWithCapture(t, cfg, acsClient, coClient)

	if acsClient.createCalls != 0 {
		t.Errorf("IMP-CLI-007: expected 0 create calls in dry-run mode, got %d", acsClient.createCalls)
	}
}

// TestIMP_CLI_007_DryRunReportedAsCreate verifies that dry-run planned creates
// appear as "create" actions in the console summary.
// Requirement: IMP-CLI-007.
func TestIMP_CLI_007_DryRunReportedAsCreate(t *testing.T) {
	acsClient := &mockACSClient{}
	coClient := &mockCOClient{
		bindings:    []cofetch.ScanSettingBinding{goodBinding("cis-weekly")},
		scanSetting: goodScanSetting(),
	}
	cfg := baseConfig()
	cfg.DryRun = true

	_, output := runWithCapture(t, cfg, acsClient, coClient)

	if !strings.Contains(output, "created: 1") {
		t.Errorf("IMP-CLI-007: expected 'created: 1' (planned) in dry-run output, got:\n%s", output)
	}
}
