package mapping

import (
	"encoding/json"
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

// ─── Wire-format tests (IMP-MAP-004a..d) ─────────────────────────────────────
//
// These tests serialize the ACS payload to JSON and verify that field names
// match the proto/api/v2 schema. They would have caught the Weekly vs DaysOfWeek
// bug: the ACS API proto has "daysOfWeek" but no "weekly" field, so a JSON
// containing "weekly" would be silently ignored by the gRPC gateway.

// allowedScheduleKeys are the JSON keys allowed in a serialized ACSSchedule,
// matching proto/api/v2/common.proto message Schedule.
var allowedScheduleKeys = map[string]bool{
	"intervalType": true,
	"hour":         true,
	"minute":       true,
	"daysOfWeek":   true,
	"daysOfMonth":  true,
}

// allowedPayloadKeys are the top-level JSON keys allowed in a serialized
// ACSCreatePayload, matching proto ComplianceScanConfiguration.
var allowedPayloadKeys = map[string]bool{
	"scanName":   true,
	"scanConfig": true,
	"clusters":   true,
}

// allowedScanConfigKeys are the JSON keys allowed in a serialized
// ACSBaseScanConfig, matching proto BaseComplianceScanConfigurationSettings.
var allowedScanConfigKeys = map[string]bool{
	"oneTimeScan":  true,
	"profiles":     true,
	"scanSchedule": true,
	"description":  true,
}

// TestIMP_MAP_004a_PayloadWireFormat_AllScheduleTypes verifies that the full
// ACS payload serializes to JSON with only proto-valid field names for each
// schedule type: DAILY, WEEKLY, MONTHLY.
func TestIMP_MAP_004a_PayloadWireFormat_AllScheduleTypes(t *testing.T) {
	cases := []struct {
		name         string
		cron         string
		wantInterval string
		wantDOW      bool // expect daysOfWeek present
		wantDOM      bool // expect daysOfMonth present
	}{
		{name: "DAILY", cron: "0 2 * * *", wantInterval: "DAILY"},
		{name: "WEEKLY_Sunday", cron: "0 2 * * 0", wantInterval: "WEEKLY", wantDOW: true},
		{name: "WEEKLY_Friday", cron: "30 14 * * 5", wantInterval: "WEEKLY", wantDOW: true},
		{name: "MONTHLY_1st", cron: "0 2 1 * *", wantInterval: "MONTHLY", wantDOM: true},
		{name: "MONTHLY_15th", cron: "0 6 15 * *", wantInterval: "MONTHLY", wantDOM: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			binding := cofetch.ScanSettingBinding{
				Namespace:       "ns",
				Name:            "b",
				ScanSettingName: "ss",
				Profiles:        []cofetch.ProfileRef{{Name: "ocp4-cis", Kind: "Profile"}},
			}
			ss := &cofetch.ScanSetting{Namespace: "ns", Name: "ss", Schedule: tc.cron}
			cfg := &models.Config{ACSClusterID: "cluster-1"}

			result := MapBinding(binding, ss, cfg)
			if result.Problem != nil {
				t.Fatalf("unexpected problem: %+v", result.Problem)
			}

			// Serialize to JSON — this is the wire format sent to the ACS API.
			data, err := json.Marshal(result.Payload)
			if err != nil {
				t.Fatalf("json.Marshal: %v", err)
			}

			// Parse back to a generic map to inspect field names.
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("unmarshal payload: %v", err)
			}

			// IMP-MAP-004d: top-level payload keys must match proto.
			for key := range raw {
				if !allowedPayloadKeys[key] {
					t.Errorf("payload contains unexpected JSON key %q (not in ComplianceScanConfiguration proto)", key)
				}
			}

			// Parse scanConfig.
			var scanConfig map[string]json.RawMessage
			if err := json.Unmarshal(raw["scanConfig"], &scanConfig); err != nil {
				t.Fatalf("unmarshal scanConfig: %v", err)
			}
			for key := range scanConfig {
				if !allowedScanConfigKeys[key] {
					t.Errorf("scanConfig contains unexpected JSON key %q (not in BaseComplianceScanConfigurationSettings proto)", key)
				}
			}

			// Parse scanSchedule.
			schedRaw, ok := scanConfig["scanSchedule"]
			if !ok {
				t.Fatal("scanSchedule missing from JSON")
			}
			var sched map[string]json.RawMessage
			if err := json.Unmarshal(schedRaw, &sched); err != nil {
				t.Fatalf("unmarshal scanSchedule: %v", err)
			}

			// IMP-MAP-004a: schedule keys must only be proto-valid.
			for key := range sched {
				if !allowedScheduleKeys[key] {
					t.Errorf("scanSchedule contains unexpected JSON key %q (not in Schedule proto; would be silently ignored by gRPC gateway)", key)
				}
			}

			// Verify intervalType.
			var intervalType string
			if err := json.Unmarshal(sched["intervalType"], &intervalType); err != nil {
				t.Fatalf("unmarshal intervalType: %v", err)
			}
			if intervalType != tc.wantInterval {
				t.Errorf("intervalType: want %q, got %q", tc.wantInterval, intervalType)
			}

			// IMP-MAP-004b: WEEKLY must have daysOfWeek.
			if tc.wantDOW {
				if _, ok := sched["daysOfWeek"]; !ok {
					t.Error("WEEKLY schedule missing daysOfWeek in JSON (API would have no day-of-week info)")
				}
			}

			// IMP-MAP-004c: MONTHLY must have daysOfMonth.
			if tc.wantDOM {
				if _, ok := sched["daysOfMonth"]; !ok {
					t.Error("MONTHLY schedule missing daysOfMonth in JSON (API would have no day-of-month info)")
				}
			}

			// DAILY should NOT have daysOfWeek or daysOfMonth.
			if !tc.wantDOW && !tc.wantDOM {
				if _, ok := sched["daysOfWeek"]; ok {
					t.Error("DAILY schedule should not have daysOfWeek in JSON")
				}
				if _, ok := sched["daysOfMonth"]; ok {
					t.Error("DAILY schedule should not have daysOfMonth in JSON")
				}
			}
		})
	}
}

// TestIMP_MAP_004b_WeeklyDaysOfWeekValue verifies the daysOfWeek.days array
// contains the correct day-of-week integer for weekly schedules.
func TestIMP_MAP_004b_WeeklyDaysOfWeekValue(t *testing.T) {
	cases := []struct {
		cron    string
		wantDay int32
	}{
		{"0 0 * * 0", 0}, // Sunday
		{"0 0 * * 1", 1}, // Monday
		{"0 0 * * 6", 6}, // Saturday
	}

	for _, tc := range cases {
		t.Run(string(rune('0'+tc.wantDay)), func(t *testing.T) {
			ss := &cofetch.ScanSetting{Namespace: "ns", Name: "s", Schedule: tc.cron}
			binding := cofetch.ScanSettingBinding{
				Namespace: "ns", Name: "b", ScanSettingName: "s",
				Profiles: []cofetch.ProfileRef{{Name: "p", Kind: "Profile"}},
			}
			result := MapBinding(binding, ss, &models.Config{ACSClusterID: "c"})
			if result.Problem != nil {
				t.Fatalf("unexpected problem: %+v", result.Problem)
			}

			data, _ := json.Marshal(result.Payload)
			var parsed struct {
				ScanConfig struct {
					ScanSchedule struct {
						DaysOfWeek struct {
							Days []int32 `json:"days"`
						} `json:"daysOfWeek"`
					} `json:"scanSchedule"`
				} `json:"scanConfig"`
			}
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			days := parsed.ScanConfig.ScanSchedule.DaysOfWeek.Days
			if len(days) != 1 || days[0] != tc.wantDay {
				t.Errorf("daysOfWeek.days: want [%d], got %v", tc.wantDay, days)
			}
		})
	}
}
