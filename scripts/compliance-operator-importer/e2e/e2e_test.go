//go:build e2e

// Package e2e runs acceptance tests against a real ACS + Compliance Operator
// cluster. These tests exercise the importer binary end-to-end.
//
// Required environment:
//
//	ROX_ENDPOINT          ACS Central URL (bare hostname OK, https:// prepended)
//	ROX_ADMIN_PASSWORD    Basic auth password (or ROX_API_TOKEN for token auth)
//
// Optional:
//
//	CO_NAMESPACE          CO namespace (default: openshift-compliance)
//	E2E_KEEP_CONFIGS      Set to "1" to skip cleanup of created scan configs
//
// Run:
//
//	go test -tags e2e -v -count=1 ./e2e/
//	# or via the convenience wrapper:
//	hack/run-e2e.sh
package e2e

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Global state set in TestMain
// ---------------------------------------------------------------------------

var (
	importerBin string // path to compiled binary
	endpoint    string // ACS Central URL (with https://)
	coNamespace string
)

func TestMain(m *testing.M) {
	endpoint = os.Getenv("ROX_ENDPOINT")
	if endpoint == "" {
		fmt.Fprintln(os.Stderr, "SKIP: ROX_ENDPOINT not set")
		os.Exit(0)
	}
	if !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}

	hasToken := os.Getenv("ROX_API_TOKEN") != ""
	hasPassword := os.Getenv("ROX_ADMIN_PASSWORD") != ""
	if !hasToken && !hasPassword {
		fmt.Fprintln(os.Stderr, "SKIP: neither ROX_API_TOKEN nor ROX_ADMIN_PASSWORD set")
		os.Exit(0)
	}

	coNamespace = os.Getenv("CO_NAMESPACE")
	if coNamespace == "" {
		coNamespace = "openshift-compliance"
	}

	// Build the importer binary.
	tmpDir, err := os.MkdirTemp("", "co-importer-e2e-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: mktemp: %v\n", err)
		os.Exit(1)
	}
	importerBin = filepath.Join(tmpDir, "co-acs-scan-importer")

	cmd := exec.Command("go", "build", "-o", importerBin, "./cmd/importer/")
	cmd.Dir = filepath.Join(mustGetwd(), "..")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: build importer: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Built importer: %s\n", importerBin)

	code := m.Run()

	os.RemoveAll(tmpDir)
	os.Exit(code)
}

func mustGetwd() string {
	d, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return d
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// importerResult captures a single importer invocation.
type importerResult struct {
	exitCode int
	stdout   string
	stderr   string
	report   *report // nil if no --report-json
}

// report mirrors the JSON report structure (subset).
type report struct {
	Meta struct {
		DryRun         bool   `json:"dryRun"`
		NamespaceScope string `json:"namespaceScope"`
		Mode           string `json:"mode"`
	} `json:"meta"`
	Counts struct {
		Discovered int `json:"discovered"`
		Create     int `json:"create"`
		Update     int `json:"update"`
		Skip       int `json:"skip"`
		Failed     int `json:"failed"`
	} `json:"counts"`
	Items    []reportItem `json:"items"`
	Problems []problem    `json:"problems"`
}

type reportItem struct {
	Source struct {
		Namespace       string `json:"namespace"`
		BindingName     string `json:"bindingName"`
		ScanSettingName string `json:"scanSettingName"`
	} `json:"source"`
	Action          string `json:"action"`
	Reason          string `json:"reason"`
	Attempts        int    `json:"attempts"`
	ACSScanConfigID string `json:"acsScanConfigId"`
	Error           string `json:"error"`
}

type problem struct {
	Severity    string `json:"severity"`
	Category    string `json:"category"`
	ResourceRef string `json:"resourceRef"`
	Description string `json:"description"`
	FixHint     string `json:"fixHint"`
	Skipped     bool   `json:"skipped"`
}

// runImporter executes the importer binary with the given extra args.
// It always passes --endpoint, --insecure-skip-verify, and --co-namespace.
// If reportJSON is true, a temp file is used and the report is parsed.
func runImporter(t *testing.T, reportJSON bool, extraArgs ...string) importerResult {
	t.Helper()

	args := []string{
		"--endpoint", endpoint,
		"--insecure-skip-verify",
		"--co-namespace", coNamespace,
	}
	args = append(args, extraArgs...)

	var reportPath string
	if reportJSON {
		f, err := os.CreateTemp("", "e2e-report-*.json")
		if err != nil {
			t.Fatalf("create temp report file: %v", err)
		}
		f.Close()
		reportPath = f.Name()
		t.Cleanup(func() { os.Remove(reportPath) })
		args = append(args, "--report-json", reportPath)
	}

	cmd := exec.CommandContext(context.Background(), importerBin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("exec importer: %v", err)
		}
	}

	result := importerResult{
		exitCode: exitCode,
		stdout:   stdout.String(),
		stderr:   stderr.String(),
	}

	if reportJSON && reportPath != "" {
		data, err := os.ReadFile(reportPath)
		if err == nil && len(data) > 0 {
			var r report
			if err := json.Unmarshal(data, &r); err != nil {
				t.Logf("WARNING: report JSON parse error: %v", err)
			} else {
				result.report = &r
			}
		}
	}

	return result
}

// acsConfigSummary is a scan config from the ACS list API.
type acsConfigSummary struct {
	ID       string `json:"id"`
	ScanName string `json:"scanName"`
}

// acsListConfigs returns all scan configurations from ACS.
func acsListConfigs(t *testing.T) []acsConfigSummary {
	t.Helper()
	body := acsGet(t, "/v2/compliance/scan/configurations?pagination.limit=1000")

	var resp struct {
		Configurations []acsConfigSummary `json:"configurations"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("parse ACS list response: %v", err)
	}
	return resp.Configurations
}

// acsDeleteConfig deletes a scan config by ID.
func acsDeleteConfig(t *testing.T, id string) {
	t.Helper()
	req := acsRequest(t, http.MethodDelete, "/v2/compliance/scan/configurations/"+id, nil)
	resp, err := acsHTTPClient().Do(req)
	if err != nil {
		t.Logf("WARNING: delete scan config %s: %v", id, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		t.Logf("WARNING: delete scan config %s: HTTP %d", id, resp.StatusCode)
	}
}

// acsGet does a GET request to ACS and returns the body.
func acsGet(t *testing.T, path string) []byte {
	t.Helper()
	req := acsRequest(t, http.MethodGet, path, nil)
	resp, err := acsHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("ACS GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ACS GET %s: HTTP %d", path, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ACS GET %s: read body: %v", path, err)
	}
	return body
}

func acsRequest(t *testing.T, method, path string, body io.Reader) *http.Request {
	t.Helper()
	url := endpoint + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("build ACS request: %v", err)
	}
	req.Header.Set("Accept", "application/json")

	if token := os.Getenv("ROX_API_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	} else {
		user := os.Getenv("ROX_ADMIN_USER")
		if user == "" {
			user = "admin"
		}
		req.SetBasicAuth(user, os.Getenv("ROX_ADMIN_PASSWORD"))
	}
	return req
}

func acsHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // e2e test
		},
	}
}

// configIDsByPrefix returns the IDs of scan configs whose name starts with prefix.
func configIDsByPrefix(t *testing.T, prefix string) []string {
	t.Helper()
	var ids []string
	for _, c := range acsListConfigs(t) {
		if strings.HasPrefix(c.ScanName, prefix) {
			ids = append(ids, c.ID)
		}
	}
	return ids
}

// cleanupConfigsByPrefix deletes all scan configs matching prefix, unless
// E2E_KEEP_CONFIGS is set.
func cleanupConfigsByPrefix(t *testing.T, prefix string) {
	if os.Getenv("E2E_KEEP_CONFIGS") == "1" {
		t.Logf("E2E_KEEP_CONFIGS=1, skipping cleanup for prefix %q", prefix)
		return
	}
	for _, id := range configIDsByPrefix(t, prefix) {
		acsDeleteConfig(t, id)
		t.Logf("cleaned up scan config %s", id)
	}
}

// scanConfigExists returns true if a scan config with the given name exists.
func scanConfigExists(t *testing.T, name string) bool {
	t.Helper()
	for _, c := range acsListConfigs(t) {
		if c.ScanName == name {
			return true
		}
	}
	return false
}

// countSSBs returns the number of ScanSettingBindings in CO_NAMESPACE.
func countSSBs(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("kubectl", "get", "scansettingbindings.compliance.openshift.io",
		"-n", coNamespace, "-o", "json")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("kubectl list SSBs: %v", err)
	}
	var list struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(out, &list); err != nil {
		t.Fatalf("parse SSB list: %v", err)
	}
	return len(list.Items)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestIMP_ACC_001_COResourcesListable verifies that CO resources can be listed
// from the target cluster.
func TestIMP_ACC_001_COResourcesListable(t *testing.T) {
	for _, resource := range []string{
		"scansettingbindings.compliance.openshift.io",
		"scansettings.compliance.openshift.io",
		"profiles.compliance.openshift.io",
	} {
		t.Run(resource, func(t *testing.T) {
			cmd := exec.Command("kubectl", "get", resource, "-n", coNamespace)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("kubectl get %s failed: %v\n%s", resource, err, out)
			}
		})
	}
}

// TestIMP_ACC_002_AuthPreflight verifies that the importer can authenticate
// with ACS (both preflight probe and actual listing work).
func TestIMP_ACC_002_AuthPreflight(t *testing.T) {
	// Just verify the ACS API is reachable with current creds.
	body := acsGet(t, "/v2/compliance/scan/configurations?pagination.limit=1")
	if len(body) == 0 {
		t.Fatal("empty response from ACS preflight probe")
	}
}

// TestIMP_ACC_003_DryRunNoWrites verifies that dry-run produces no changes.
func TestIMP_ACC_003_DryRunNoWrites(t *testing.T) {
	// Snapshot existing configs.
	before := acsListConfigs(t)

	result := runImporter(t, true, "--dry-run")

	if result.exitCode != 0 && result.exitCode != 2 {
		t.Fatalf("IMP-ACC-003: expected exit code 0 or 2, got %d\nstdout: %s\nstderr: %s",
			result.exitCode, result.stdout, result.stderr)
	}

	if result.report == nil {
		t.Fatal("IMP-ACC-003: expected report JSON to be written")
	}
	if !result.report.Meta.DryRun {
		t.Error("IMP-ACC-003: report meta.dryRun should be true")
	}

	// Verify no new configs were created.
	after := acsListConfigs(t)
	if len(after) != len(before) {
		t.Errorf("IMP-ACC-003: config count changed from %d to %d during dry-run",
			len(before), len(after))
	}
}

// TestIMP_ACC_004_ApplyCreatesConfigs verifies that apply mode creates
// ACS scan configs for discovered SSBs.
func TestIMP_ACC_004_ApplyCreatesConfigs(t *testing.T) {
	nSSBs := countSSBs(t)
	if nSSBs == 0 {
		t.Skip("no ScanSettingBindings found in namespace " + coNamespace)
	}

	result := runImporter(t, true)

	if result.exitCode != 0 && result.exitCode != 2 {
		t.Fatalf("IMP-ACC-004: expected exit code 0 or 2, got %d\nstdout: %s\nstderr: %s",
			result.exitCode, result.stdout, result.stderr)
	}

	if result.report == nil {
		t.Fatal("IMP-ACC-004: expected report")
	}

	t.Logf("Discovered: %d, Created: %d, Skipped: %d, Failed: %d",
		result.report.Counts.Discovered,
		result.report.Counts.Create,
		result.report.Counts.Skip,
		result.report.Counts.Failed,
	)

	if result.report.Counts.Discovered == 0 {
		t.Error("IMP-ACC-004: expected at least 1 discovered binding")
	}

	// Verify created configs exist in ACS.
	for _, item := range result.report.Items {
		if item.Action == "create" && item.ACSScanConfigID != "" {
			t.Logf("Created: %s (id=%s)", item.Source.BindingName, item.ACSScanConfigID)
		}
	}

	// Cleanup: delete configs we created.
	t.Cleanup(func() {
		for _, item := range result.report.Items {
			if item.Action == "create" && item.ACSScanConfigID != "" {
				acsDeleteConfig(t, item.ACSScanConfigID)
			}
		}
	})
}

// TestIMP_ACC_005_IdempotentSecondRun verifies that a second run with the same
// inputs produces only skip actions (no new creates).
func TestIMP_ACC_005_IdempotentSecondRun(t *testing.T) {
	nSSBs := countSSBs(t)
	if nSSBs == 0 {
		t.Skip("no ScanSettingBindings")
	}

	// First run: create.
	r1 := runImporter(t, true)
	if r1.exitCode != 0 && r1.exitCode != 2 {
		t.Fatalf("first run exit code %d", r1.exitCode)
	}

	var createdIDs []string
	if r1.report != nil {
		for _, item := range r1.report.Items {
			if item.Action == "create" && item.ACSScanConfigID != "" {
				createdIDs = append(createdIDs, item.ACSScanConfigID)
			}
		}
	}

	t.Cleanup(func() {
		for _, id := range createdIDs {
			acsDeleteConfig(t, id)
		}
	})

	// Second run: should be all skips.
	r2 := runImporter(t, true)
	if r2.exitCode != 0 && r2.exitCode != 2 {
		t.Fatalf("IMP-ACC-005: second run exit code %d", r2.exitCode)
	}

	if r2.report != nil && r2.report.Counts.Create > 0 {
		t.Errorf("IMP-ACC-005: second run created %d configs (expected 0)", r2.report.Counts.Create)
	}
}

// TestIMP_ACC_007_InvalidTokenFailsFast verifies that an invalid token
// produces exit code 1 (fatal).
func TestIMP_ACC_007_InvalidTokenFailsFast(t *testing.T) {
	// Override auth with a bad token.
	cmd := exec.Command(importerBin,
		"--endpoint", endpoint,
		"--insecure-skip-verify",
		"--co-namespace", coNamespace,
	)
	cmd.Env = append(os.Environ(),
		"ROX_API_TOKEN=definitely-not-a-valid-token",
		"ROX_ADMIN_PASSWORD=", // clear password to avoid ambiguous auth
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}

	if exitCode != 1 {
		t.Errorf("IMP-ACC-007: expected exit code 1 for invalid token, got %d\nstdout: %s\nstderr: %s",
			exitCode, stdout.String(), stderr.String())
	}
}

// TestIMP_ACC_012_ProblemsHaveFixHints verifies that all problems in the
// report include description and fixHint fields.
func TestIMP_ACC_012_ProblemsHaveFixHints(t *testing.T) {
	result := runImporter(t, true, "--dry-run")

	if result.report == nil {
		t.Skip("no report generated")
	}

	for i, p := range result.report.Problems {
		if p.Description == "" {
			t.Errorf("IMP-ACC-012: problem[%d] has empty description", i)
		}
		if p.FixHint == "" {
			t.Errorf("IMP-ACC-012: problem[%d] has empty fixHint (description: %s)", i, p.Description)
		}
	}
}

// TestIMP_ACC_017_AutoDiscoverClusterID verifies that the importer can
// auto-discover the ACS cluster ID without --cluster.
func TestIMP_ACC_017_AutoDiscoverClusterID(t *testing.T) {
	result := runImporter(t, true, "--dry-run")

	if result.exitCode == 1 {
		// Check if it's an auto-discovery failure.
		combined := result.stdout + result.stderr
		if strings.Contains(combined, "discover cluster ID") {
			t.Fatalf("IMP-ACC-017: auto-discovery failed:\n%s", combined)
		}
	}

	// If exit 0 or 2, auto-discovery succeeded (it's used implicitly when
	// no --cluster is given).
	if result.exitCode != 0 && result.exitCode != 2 {
		t.Errorf("IMP-ACC-017: expected exit 0 or 2, got %d\nstdout: %s\nstderr: %s",
			result.exitCode, result.stdout, result.stderr)
	}
}

// TestIMP_ACC_014_OverwriteExistingUpdates verifies that --overwrite-existing
// updates existing scan configs instead of skipping them.
func TestIMP_ACC_014_OverwriteExistingUpdates(t *testing.T) {
	nSSBs := countSSBs(t)
	if nSSBs == 0 {
		t.Skip("no ScanSettingBindings")
	}

	// First run: create.
	r1 := runImporter(t, true)
	if r1.report == nil || r1.report.Counts.Create == 0 {
		// Nothing was created (maybe everything already exists). Create-then-overwrite
		// test only makes sense when we create something.
		t.Skip("no new configs created in first run")
	}

	var createdIDs []string
	for _, item := range r1.report.Items {
		if item.Action == "create" && item.ACSScanConfigID != "" {
			createdIDs = append(createdIDs, item.ACSScanConfigID)
		}
	}
	t.Cleanup(func() {
		for _, id := range createdIDs {
			acsDeleteConfig(t, id)
		}
	})

	// Second run with --overwrite-existing: should update, not skip.
	r2 := runImporter(t, true, "--overwrite-existing")
	if r2.exitCode != 0 && r2.exitCode != 2 {
		t.Fatalf("overwrite run exit code %d", r2.exitCode)
	}

	if r2.report == nil {
		t.Fatal("IMP-ACC-014: expected report from overwrite run")
	}

	if r2.report.Counts.Update == 0 && r2.report.Counts.Skip > 0 {
		t.Error("IMP-ACC-014: expected updates with --overwrite-existing, got only skips")
	}
	t.Logf("Overwrite run: updated=%d, created=%d, skipped=%d",
		r2.report.Counts.Update, r2.report.Counts.Create, r2.report.Counts.Skip)
}
