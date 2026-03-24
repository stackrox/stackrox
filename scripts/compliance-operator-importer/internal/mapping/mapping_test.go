package mapping

import (
	"strings"
	"testing"

	"github.com/stackrox/co-acs-importer/internal/cofetch"
	"github.com/stackrox/co-acs-importer/internal/models"
)

// baseBinding returns a minimal valid ScanSettingBinding for tests.
func baseBinding() cofetch.ScanSettingBinding {
	return cofetch.ScanSettingBinding{
		Namespace:       "openshift-compliance",
		Name:            "cis-weekly",
		ScanSettingName: "default-auto-apply",
		Profiles: []cofetch.ProfileRef{
			{Name: "ocp4-cis-node", Kind: "Profile"},
			{Name: "ocp4-cis-master", Kind: "Profile"},
			{Name: "my-tailored-profile", Kind: "TailoredProfile"},
		},
	}
}

// baseScanSetting returns a minimal valid ScanSetting for tests.
func baseScanSetting() *cofetch.ScanSetting {
	return &cofetch.ScanSetting{
		Namespace: "openshift-compliance",
		Name:      "default-auto-apply",
		Schedule:  "0 0 * * *",
	}
}

// baseConfig returns a minimal Config for tests.
func baseConfig() *models.Config {
	return &models.Config{
		ACSClusterID: "cluster-a",
	}
}

// TestIMP_MAP_001_ScanName verifies the ACS payload scanName equals the binding name.
func TestIMP_MAP_001_ScanName(t *testing.T) {
	result := MapBinding(baseBinding(), baseScanSetting(), baseConfig())
	if result.Problem != nil {
		t.Fatalf("unexpected problem: %+v", result.Problem)
	}
	if result.Payload == nil {
		t.Fatal("expected non-nil payload")
	}
	if result.Payload.ScanName != "cis-weekly" {
		t.Errorf("ScanName: want %q, got %q", "cis-weekly", result.Payload.ScanName)
	}
}

// TestIMP_MAP_001_ProfilesSortedDeduped verifies profiles are sorted and deduplicated.
func TestIMP_MAP_001_ProfilesSortedDeduped(t *testing.T) {
	binding := baseBinding()
	// Add a duplicate entry.
	binding.Profiles = append(binding.Profiles, cofetch.ProfileRef{Name: "ocp4-cis-node", Kind: "Profile"})

	result := MapBinding(binding, baseScanSetting(), baseConfig())
	if result.Problem != nil {
		t.Fatalf("unexpected problem: %+v", result.Problem)
	}
	want := []string{"my-tailored-profile", "ocp4-cis-master", "ocp4-cis-node"}
	got := result.Payload.ScanConfig.Profiles
	if len(got) != len(want) {
		t.Fatalf("Profiles len: want %d, got %d: %v", len(want), len(got), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("Profiles[%d]: want %q, got %q", i, w, got[i])
		}
	}
}

// TestIMP_MAP_002_MissingKindDefaultsToProfile verifies that a ProfileRef with empty
// Kind is accepted and the profile name is included in ACS profiles (IMP-MAP-002).
// The kind=="" semantics mean "treat as Profile" — no lookup difference for the importer.
func TestIMP_MAP_002_MissingKindDefaultsToProfile(t *testing.T) {
	binding := baseBinding()
	binding.Profiles = []cofetch.ProfileRef{
		{Name: "custom-x"}, // Kind is empty => defaults to "Profile"
	}

	result := MapBinding(binding, baseScanSetting(), baseConfig())
	if result.Problem != nil {
		t.Fatalf("unexpected problem: %+v", result.Problem)
	}
	if len(result.Payload.ScanConfig.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %v", result.Payload.ScanConfig.Profiles)
	}
	if result.Payload.ScanConfig.Profiles[0] != "custom-x" {
		t.Errorf("profile name: want %q, got %q", "custom-x", result.Payload.ScanConfig.Profiles[0])
	}
}

// TestIMP_MAP_003_OneTimeScanFalseWhenScheduleSet verifies oneTimeScan is false
// when the ScanSetting has a cron schedule.
func TestIMP_MAP_003_OneTimeScanFalseWhenScheduleSet(t *testing.T) {
	result := MapBinding(baseBinding(), baseScanSetting(), baseConfig())
	if result.Problem != nil {
		t.Fatalf("unexpected problem: %+v", result.Problem)
	}
	if result.Payload.ScanConfig.OneTimeScan {
		t.Error("OneTimeScan: want false when schedule is set")
	}
}

// TestIMP_MAP_004_ScanSchedulePresentWhenScheduleSet verifies scanSchedule is non-nil
// when the ScanSetting has a cron schedule.
func TestIMP_MAP_004_ScanSchedulePresentWhenScheduleSet(t *testing.T) {
	result := MapBinding(baseBinding(), baseScanSetting(), baseConfig())
	if result.Problem != nil {
		t.Fatalf("unexpected problem: %+v", result.Problem)
	}
	if result.Payload.ScanConfig.ScanSchedule == nil {
		t.Error("ScanSchedule: want non-nil when schedule is set")
	}
}

// TestIMP_MAP_005_DescriptionContainsBindingRef verifies the description contains
// "Imported from CO ScanSettingBinding <namespace>/<name>".
func TestIMP_MAP_005_DescriptionContainsBindingRef(t *testing.T) {
	result := MapBinding(baseBinding(), baseScanSetting(), baseConfig())
	if result.Problem != nil {
		t.Fatalf("unexpected problem: %+v", result.Problem)
	}
	want := "Imported from CO ScanSettingBinding openshift-compliance/cis-weekly"
	if !strings.Contains(result.Payload.ScanConfig.Description, want) {
		t.Errorf("Description: want it to contain %q, got %q", want, result.Payload.ScanConfig.Description)
	}
}

// TestIMP_MAP_006_DescriptionIncludesScanSettingName verifies the description
// includes a reference to the ScanSetting name.
func TestIMP_MAP_006_DescriptionIncludesScanSettingName(t *testing.T) {
	result := MapBinding(baseBinding(), baseScanSetting(), baseConfig())
	if result.Problem != nil {
		t.Fatalf("unexpected problem: %+v", result.Problem)
	}
	if !strings.Contains(result.Payload.ScanConfig.Description, "default-auto-apply") {
		t.Errorf("Description: want ScanSetting name %q included, got %q",
			"default-auto-apply", result.Payload.ScanConfig.Description)
	}
}

// TestIMP_MAP_007_ClustersContainsACSClusterID verifies clusters contains the
// configured ACS cluster ID.
func TestIMP_MAP_007_ClustersContainsACSClusterID(t *testing.T) {
	result := MapBinding(baseBinding(), baseScanSetting(), baseConfig())
	if result.Problem != nil {
		t.Fatalf("unexpected problem: %+v", result.Problem)
	}
	if len(result.Payload.Clusters) != 1 {
		t.Fatalf("Clusters: want 1 entry, got %v", result.Payload.Clusters)
	}
	if result.Payload.Clusters[0] != "cluster-a" {
		t.Errorf("Clusters[0]: want %q, got %q", "cluster-a", result.Payload.Clusters[0])
	}
}

// TestIMP_MAP_008_MissingScanSettingSkipsBinding verifies that a nil ScanSetting
// results in a MappingResult with nil Payload and non-nil Problem (IMP-MAP-008).
func TestIMP_MAP_008_MissingScanSettingSkipsBinding(t *testing.T) {
	result := MapBinding(baseBinding(), nil, baseConfig())
	if result.Payload != nil {
		t.Errorf("Payload: want nil when ScanSetting is missing, got %+v", result.Payload)
	}
	if result.Problem == nil {
		t.Fatal("Problem: want non-nil when ScanSetting is missing")
	}
}

// TestIMP_MAP_008_MissingScanSettingCategoryMapping verifies the problem category
// is "mapping" for a missing ScanSetting (IMP-MAP-008).
func TestIMP_MAP_008_MissingScanSettingCategoryMapping(t *testing.T) {
	result := MapBinding(baseBinding(), nil, baseConfig())
	if result.Problem == nil {
		t.Fatal("Problem: want non-nil")
	}
	if result.Problem.Category != models.CategoryMapping {
		t.Errorf("Problem.Category: want %q, got %q", models.CategoryMapping, result.Problem.Category)
	}
}

// TestIMP_MAP_009_MissingScanSettingProblemsEntry verifies the problem entry has
// a populated ResourceRef (IMP-MAP-009).
func TestIMP_MAP_009_MissingScanSettingProblemsEntry(t *testing.T) {
	result := MapBinding(baseBinding(), nil, baseConfig())
	if result.Problem == nil {
		t.Fatal("Problem: want non-nil")
	}
	if result.Problem.ResourceRef == "" {
		t.Error("Problem.ResourceRef: want non-empty")
	}
}

// TestIMP_MAP_010_MissingScanSettingFixHint verifies the problem entry has a
// non-empty fix hint (IMP-MAP-010).
func TestIMP_MAP_010_MissingScanSettingFixHint(t *testing.T) {
	result := MapBinding(baseBinding(), nil, baseConfig())
	if result.Problem == nil {
		t.Fatal("Problem: want non-nil")
	}
	if result.Problem.FixHint == "" {
		t.Error("Problem.FixHint: want non-empty fix hint for missing ScanSetting")
	}
}

// TestIMP_MAP_011_OtherValidBindingsStillProcessed verifies that a missing ScanSetting
// only affects that binding; independent MapBinding calls for valid bindings succeed (IMP-MAP-011).
func TestIMP_MAP_011_OtherValidBindingsStillProcessed(t *testing.T) {
	// Broken binding (nil ScanSetting).
	broken := MapBinding(baseBinding(), nil, baseConfig())
	if broken.Problem == nil {
		t.Fatal("broken binding: want Problem set")
	}

	// Valid binding processed independently and must succeed.
	validBinding := cofetch.ScanSettingBinding{
		Namespace:       "openshift-compliance",
		Name:            "another-binding",
		ScanSettingName: "default-auto-apply",
		Profiles:        []cofetch.ProfileRef{{Name: "ocp4-cis", Kind: "Profile"}},
	}
	valid := MapBinding(validBinding, baseScanSetting(), baseConfig())
	if valid.Problem != nil {
		t.Fatalf("valid binding: unexpected problem: %+v", valid.Problem)
	}
	if valid.Payload == nil {
		t.Fatal("valid binding: expected non-nil payload")
	}
}

// TestIMP_MAP_012_InvalidCronSkipsBinding verifies that an invalid cron expression
// causes the binding to be skipped (Payload=nil, Problem set, Skipped=true) (IMP-MAP-012).
func TestIMP_MAP_012_InvalidCronSkipsBinding(t *testing.T) {
	ss := &cofetch.ScanSetting{
		Namespace: "openshift-compliance",
		Name:      "bad-schedule",
		Schedule:  "every day at noon",
	}
	result := MapBinding(baseBinding(), ss, baseConfig())
	if result.Payload != nil {
		t.Errorf("Payload: want nil for invalid cron, got %+v", result.Payload)
	}
	if result.Problem == nil {
		t.Fatal("Problem: want non-nil for invalid cron")
	}
	if !result.Problem.Skipped {
		t.Error("Problem.Skipped: want true")
	}
}

// TestIMP_MAP_013_InvalidCronProblemCategoryMapping verifies the problem category
// is "mapping" for an invalid schedule (IMP-MAP-013).
func TestIMP_MAP_013_InvalidCronProblemCategoryMapping(t *testing.T) {
	ss := &cofetch.ScanSetting{
		Namespace: "openshift-compliance",
		Name:      "bad-schedule",
		Schedule:  "every day at noon",
	}
	result := MapBinding(baseBinding(), ss, baseConfig())
	if result.Problem == nil {
		t.Fatal("Problem: want non-nil")
	}
	if result.Problem.Category != models.CategoryMapping {
		t.Errorf("Problem.Category: want %q, got %q", models.CategoryMapping, result.Problem.Category)
	}
}

// TestIMP_MAP_014_InvalidCronDescriptionMentionsSchedule verifies the problem
// description mentions schedule conversion failure (IMP-MAP-014).
func TestIMP_MAP_014_InvalidCronDescriptionMentionsSchedule(t *testing.T) {
	ss := &cofetch.ScanSetting{
		Namespace: "openshift-compliance",
		Name:      "bad-schedule",
		Schedule:  "every day at noon",
	}
	result := MapBinding(baseBinding(), ss, baseConfig())
	if result.Problem == nil {
		t.Fatal("Problem: want non-nil")
	}
	desc := strings.ToLower(result.Problem.Description)
	if !strings.Contains(desc, "schedule") {
		t.Errorf("Problem.Description: want it to mention %q, got %q", "schedule", result.Problem.Description)
	}
}

// TestIMP_MAP_015_InvalidCronFixHintMentionsCron verifies the problem fix hint
// suggests using a valid cron expression (IMP-MAP-015).
func TestIMP_MAP_015_InvalidCronFixHintMentionsCron(t *testing.T) {
	ss := &cofetch.ScanSetting{
		Namespace: "openshift-compliance",
		Name:      "bad-schedule",
		Schedule:  "every day at noon",
	}
	result := MapBinding(baseBinding(), ss, baseConfig())
	if result.Problem == nil {
		t.Fatal("Problem: want non-nil")
	}
	hint := strings.ToLower(result.Problem.FixHint)
	if !strings.Contains(hint, "cron") {
		t.Errorf("Problem.FixHint: want it to mention %q, got %q", "cron", result.Problem.FixHint)
	}
}
