package adopt

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stackrox/co-acs-importer/internal/cofetch"
)

// mockCOClient is a test double for cofetch.COClient that supports:
// - controllable ScanSetting existence (via scanSettings map)
// - tracking PatchSSBSettingsRef calls
// - simulating errors
type mockCOClient struct {
	mu           sync.Mutex
	scanSettings map[string]*cofetch.ScanSetting // key: "namespace/name"
	patches      []patchCall                     // recorded PatchSSBSettingsRef calls
	patchErr     error                           // if non-nil, PatchSSBSettingsRef returns this
}

type patchCall struct {
	Namespace          string
	SSBName            string
	NewSettingsRefName string
}

func newMockCOClient() *mockCOClient {
	return &mockCOClient{
		scanSettings: make(map[string]*cofetch.ScanSetting),
	}
}

func (m *mockCOClient) addScanSetting(namespace, name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scanSettings[namespace+"/"+name] = &cofetch.ScanSetting{
		Namespace: namespace,
		Name:      name,
	}
}

func (m *mockCOClient) ListScanSettingBindings(_ context.Context) ([]cofetch.ScanSettingBinding, error) {
	return nil, nil
}

func (m *mockCOClient) GetScanSetting(_ context.Context, namespace, name string) (*cofetch.ScanSetting, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ss, ok := m.scanSettings[namespace+"/"+name]
	if !ok {
		return nil, fmt.Errorf("ScanSetting %q not found in namespace %q", name, namespace)
	}
	return ss, nil
}

func (m *mockCOClient) PatchSSBSettingsRef(_ context.Context, namespace, ssbName, newSettingsRefName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.patchErr != nil {
		return m.patchErr
	}
	m.patches = append(m.patches, patchCall{
		Namespace:          namespace,
		SSBName:            ssbName,
		NewSettingsRefName: newSettingsRefName,
	})
	return nil
}

// Compile-time check.
var _ cofetch.COClient = (*mockCOClient)(nil)

// TestIMP_ADOPT_001_PatchSettingsRef verifies that the SSB's settingsRef is
// patched to the scan config name after ACS creates the ScanSetting.
func TestIMP_ADOPT_001_PatchSettingsRef(t *testing.T) {
	client := newMockCOClient()
	client.addScanSetting("openshift-compliance", "cis-weekly")

	adopter := &Adopter{PollInterval: 10 * time.Millisecond, PollTimeout: 1 * time.Second}
	results := adopter.Adopt(context.Background(), []Request{{
		SSBName:       "cis-weekly",
		SSBNamespace:  "openshift-compliance",
		OldSettingRef: "my-old-setting",
		ClusterLabel:  "ctx-a",
		COClient:      client,
	}})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if !r.Adopted {
		t.Errorf("expected Adopted=true, got false; message: %s", r.Message)
	}
	if len(client.patches) != 1 {
		t.Fatalf("expected 1 patch call, got %d", len(client.patches))
	}
	p := client.patches[0]
	if p.NewSettingsRefName != "cis-weekly" {
		t.Errorf("patch newSettingsRefName: want %q, got %q", "cis-weekly", p.NewSettingsRefName)
	}
	if p.SSBName != "cis-weekly" {
		t.Errorf("patch SSBName: want %q, got %q", "cis-weekly", p.SSBName)
	}
}

// TestIMP_ADOPT_002_LogMessage verifies the result message mentions the adoption.
func TestIMP_ADOPT_002_LogMessage(t *testing.T) {
	client := newMockCOClient()
	client.addScanSetting("openshift-compliance", "cis-weekly")

	adopter := &Adopter{PollInterval: 10 * time.Millisecond, PollTimeout: 1 * time.Second}
	results := adopter.Adopt(context.Background(), []Request{{
		SSBName:       "cis-weekly",
		SSBNamespace:  "openshift-compliance",
		OldSettingRef: "my-old-setting",
		ClusterLabel:  "ctx-a",
		COClient:      client,
	}})

	r := results[0]
	if r.Message == "" {
		t.Error("expected non-empty message for adopted SSB")
	}
	// Message should mention old and new setting names.
	for _, want := range []string{"my-old-setting", "cis-weekly", "adopted"} {
		if !containsStr(r.Message, want) {
			t.Errorf("message should contain %q, got %q", want, r.Message)
		}
	}
}

// TestIMP_ADOPT_003_SkipAlreadyAdopted verifies that no patch is issued when
// the SSB already references the correct ScanSetting.
func TestIMP_ADOPT_003_SkipAlreadyAdopted(t *testing.T) {
	client := newMockCOClient()
	// ScanSetting doesn't even need to exist — we skip before polling.

	adopter := &Adopter{PollInterval: 10 * time.Millisecond, PollTimeout: 1 * time.Second}
	results := adopter.Adopt(context.Background(), []Request{{
		SSBName:       "cis-weekly",
		SSBNamespace:  "openshift-compliance",
		OldSettingRef: "cis-weekly", // already correct!
		ClusterLabel:  "ctx-a",
		COClient:      client,
	}})

	r := results[0]
	if !r.Skipped {
		t.Error("expected Skipped=true when settingsRef already matches")
	}
	if r.Adopted {
		t.Error("expected Adopted=false when skipped")
	}
	if len(client.patches) != 0 {
		t.Errorf("expected 0 patch calls, got %d", len(client.patches))
	}
}

// TestIMP_ADOPT_004_005_006_Timeout verifies that a timeout waiting for the
// ScanSetting results in a warning (not an error), and no patch.
func TestIMP_ADOPT_004_005_006_Timeout(t *testing.T) {
	client := newMockCOClient()
	// Don't add the ScanSetting — it never appears.

	adopter := &Adopter{PollInterval: 10 * time.Millisecond, PollTimeout: 50 * time.Millisecond}
	results := adopter.Adopt(context.Background(), []Request{{
		SSBName:       "cis-weekly",
		SSBNamespace:  "openshift-compliance",
		OldSettingRef: "my-old-setting",
		ClusterLabel:  "ctx-a",
		COClient:      client,
	}})

	r := results[0]
	if !r.TimedOut {
		t.Error("expected TimedOut=true")
	}
	if r.Adopted {
		t.Error("expected Adopted=false on timeout")
	}
	if r.Err != nil {
		t.Errorf("expected Err=nil on timeout (warning, not error), got %v", r.Err)
	}
	if len(client.patches) != 0 {
		t.Errorf("expected 0 patch calls on timeout, got %d", len(client.patches))
	}
}

// TestIMP_ADOPT_007_MultiClusterIndependent verifies that adoption patches
// SSBs on each cluster independently.
func TestIMP_ADOPT_007_MultiClusterIndependent(t *testing.T) {
	clientA := newMockCOClient()
	clientA.addScanSetting("openshift-compliance", "cis-weekly")

	clientB := newMockCOClient()
	clientB.addScanSetting("openshift-compliance", "cis-weekly")

	adopter := &Adopter{PollInterval: 10 * time.Millisecond, PollTimeout: 1 * time.Second}
	results := adopter.Adopt(context.Background(), []Request{
		{
			SSBName:       "cis-weekly",
			SSBNamespace:  "openshift-compliance",
			OldSettingRef: "setting-a",
			ClusterLabel:  "ctx-a",
			COClient:      clientA,
		},
		{
			SSBName:       "cis-weekly",
			SSBNamespace:  "openshift-compliance",
			OldSettingRef: "setting-b",
			ClusterLabel:  "ctx-b",
			COClient:      clientB,
		},
	})

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for i, r := range results {
		if !r.Adopted {
			t.Errorf("results[%d]: expected Adopted=true, got false; message: %s", i, r.Message)
		}
	}
	if len(clientA.patches) != 1 {
		t.Errorf("clientA: expected 1 patch, got %d", len(clientA.patches))
	}
	if len(clientB.patches) != 1 {
		t.Errorf("clientB: expected 1 patch, got %d", len(clientB.patches))
	}
}

// TestIMP_ADOPT_008_PartialSuccess verifies that a timeout on one cluster
// does not block adoption on another.
func TestIMP_ADOPT_008_PartialSuccess(t *testing.T) {
	clientA := newMockCOClient()
	clientA.addScanSetting("openshift-compliance", "cis-weekly")

	clientB := newMockCOClient()
	// Don't add ScanSetting on B — it times out.

	adopter := &Adopter{PollInterval: 10 * time.Millisecond, PollTimeout: 50 * time.Millisecond}
	results := adopter.Adopt(context.Background(), []Request{
		{
			SSBName:       "cis-weekly",
			SSBNamespace:  "openshift-compliance",
			OldSettingRef: "setting-a",
			ClusterLabel:  "ctx-a",
			COClient:      clientA,
		},
		{
			SSBName:       "cis-weekly",
			SSBNamespace:  "openshift-compliance",
			OldSettingRef: "setting-b",
			ClusterLabel:  "ctx-b",
			COClient:      clientB,
		},
	})

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// ctx-a should succeed.
	if !results[0].Adopted {
		t.Errorf("ctx-a: expected Adopted=true; message: %s", results[0].Message)
	}
	// ctx-b should time out without error.
	if !results[1].TimedOut {
		t.Errorf("ctx-b: expected TimedOut=true; message: %s", results[1].Message)
	}
	if results[1].Err != nil {
		t.Errorf("ctx-b: expected Err=nil on timeout, got %v", results[1].Err)
	}
}

// TestIMP_ADOPT_PatchError verifies that a patch failure is recorded as an error.
func TestIMP_ADOPT_PatchError(t *testing.T) {
	client := newMockCOClient()
	client.addScanSetting("openshift-compliance", "cis-weekly")
	client.patchErr = errors.New("permission denied")

	adopter := &Adopter{PollInterval: 10 * time.Millisecond, PollTimeout: 1 * time.Second}
	results := adopter.Adopt(context.Background(), []Request{{
		SSBName:       "cis-weekly",
		SSBNamespace:  "openshift-compliance",
		OldSettingRef: "my-old-setting",
		ClusterLabel:  "ctx-a",
		COClient:      client,
	}})

	r := results[0]
	if r.Adopted {
		t.Error("expected Adopted=false on patch error")
	}
	if r.Err == nil {
		t.Error("expected non-nil Err on patch failure")
	}
}

// TestIMP_ADOPT_DelayedScanSetting verifies that the adopter polls and
// succeeds when the ScanSetting appears after a delay.
func TestIMP_ADOPT_DelayedScanSetting(t *testing.T) {
	client := newMockCOClient()

	// Add the ScanSetting after a short delay.
	go func() {
		time.Sleep(30 * time.Millisecond)
		client.addScanSetting("openshift-compliance", "cis-weekly")
	}()

	adopter := &Adopter{PollInterval: 10 * time.Millisecond, PollTimeout: 1 * time.Second}
	results := adopter.Adopt(context.Background(), []Request{{
		SSBName:       "cis-weekly",
		SSBNamespace:  "openshift-compliance",
		OldSettingRef: "my-old-setting",
		ClusterLabel:  "ctx-a",
		COClient:      client,
	}})

	r := results[0]
	if !r.Adopted {
		t.Errorf("expected Adopted=true after delayed ScanSetting; message: %s", r.Message)
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
