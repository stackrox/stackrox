package merge

import (
	"slices"
	"testing"

	"github.com/stackrox/co-acs-importer/internal/models"
)

// TestIMP_MAP_019_MergeSameProfilesSameSchedule verifies that SSBs with the
// same name, same profiles, and same schedule are merged across clusters.
func TestIMP_MAP_019_MergeSameProfilesSameSchedule(t *testing.T) {
	input := map[string][]MappedSSB{
		"cluster-1": {
			{
				Name:     "cis-benchmark",
				Profiles: []string{"ocp4-cis", "ocp4-cis-node"},
				Payload: models.ACSCreatePayload{
					ScanName: "cis-benchmark",
					ScanConfig: models.ACSBaseScanConfig{
						Profiles: []string{"ocp4-cis", "ocp4-cis-node"},
						ScanSchedule: &models.ACSSchedule{
							Hour:   2,
							Minute: 30,
						},
					},
					Clusters: []string{"cluster-1"},
				},
			},
		},
		"cluster-2": {
			{
				Name:     "cis-benchmark",
				Profiles: []string{"ocp4-cis", "ocp4-cis-node"},
				Payload: models.ACSCreatePayload{
					ScanName: "cis-benchmark",
					ScanConfig: models.ACSBaseScanConfig{
						Profiles: []string{"ocp4-cis", "ocp4-cis-node"},
						ScanSchedule: &models.ACSSchedule{
							Hour:   2,
							Minute: 30,
						},
					},
					Clusters: []string{"cluster-2"},
				},
			},
		},
	}

	result := MergeSSBs(input)
	if len(result.Merged) != 1 {
		t.Fatalf("expected 1 merged SSB, got %d", len(result.Merged))
	}

	merged := result.Merged[0]
	if merged.Name != "cis-benchmark" {
		t.Errorf("expected SSB name 'cis-benchmark', got %q", merged.Name)
	}

	// Clusters should be merged.
	slices.Sort(merged.Payload.Clusters)
	expected := []string{"cluster-1", "cluster-2"}
	if !stringSlicesEqual(merged.Payload.Clusters, expected) {
		t.Errorf("expected clusters %v, got %v", expected, merged.Payload.Clusters)
	}

	if len(result.Problems) != 0 {
		t.Errorf("expected no problems, got %d", len(result.Problems))
	}
}

// TestIMP_MAP_021_MergeIdenticalSSBsUnion verifies that identical SSBs are
// merged with a union of cluster IDs.
func TestIMP_MAP_021_MergeIdenticalSSBsUnion(t *testing.T) {
	input := map[string][]MappedSSB{
		"cluster-a": {
			{
				Name:     "ssb-1",
				Profiles: []string{"profile-x"},
				Payload: models.ACSCreatePayload{
					ScanName: "ssb-1",
					ScanConfig: models.ACSBaseScanConfig{
						Profiles: []string{"profile-x"},
						ScanSchedule: &models.ACSSchedule{
							Hour:   10,
							Minute: 0,
						},
					},
					Clusters: []string{"cluster-a"},
				},
			},
		},
		"cluster-b": {
			{
				Name:     "ssb-1",
				Profiles: []string{"profile-x"},
				Payload: models.ACSCreatePayload{
					ScanName: "ssb-1",
					ScanConfig: models.ACSBaseScanConfig{
						Profiles: []string{"profile-x"},
						ScanSchedule: &models.ACSSchedule{
							Hour:   10,
							Minute: 0,
						},
					},
					Clusters: []string{"cluster-b"},
				},
			},
		},
		"cluster-c": {
			{
				Name:     "ssb-1",
				Profiles: []string{"profile-x"},
				Payload: models.ACSCreatePayload{
					ScanName: "ssb-1",
					ScanConfig: models.ACSBaseScanConfig{
						Profiles: []string{"profile-x"},
						ScanSchedule: &models.ACSSchedule{
							Hour:   10,
							Minute: 0,
						},
					},
					Clusters: []string{"cluster-c"},
				},
			},
		},
	}

	result := MergeSSBs(input)
	if len(result.Merged) != 1 {
		t.Fatalf("expected 1 merged SSB, got %d", len(result.Merged))
	}

	merged := result.Merged[0]
	slices.Sort(merged.Payload.Clusters)
	expected := []string{"cluster-a", "cluster-b", "cluster-c"}
	if !stringSlicesEqual(merged.Payload.Clusters, expected) {
		t.Errorf("expected clusters %v, got %v", expected, merged.Payload.Clusters)
	}
}

// TestIMP_MAP_020_DifferentProfilesError verifies that SSBs with the same name
// but different profiles produce an error.
func TestIMP_MAP_020_DifferentProfilesError(t *testing.T) {
	input := map[string][]MappedSSB{
		"cluster-1": {
			{
				Name:     "ssb-conflict",
				Profiles: []string{"profile-a"},
				Payload: models.ACSCreatePayload{
					ScanName: "ssb-conflict",
					ScanConfig: models.ACSBaseScanConfig{
						Profiles: []string{"profile-a"},
					},
					Clusters: []string{"cluster-1"},
				},
			},
		},
		"cluster-2": {
			{
				Name:     "ssb-conflict",
				Profiles: []string{"profile-b"},
				Payload: models.ACSCreatePayload{
					ScanName: "ssb-conflict",
					ScanConfig: models.ACSBaseScanConfig{
						Profiles: []string{"profile-b"},
					},
					Clusters: []string{"cluster-2"},
				},
			},
		},
	}

	result := MergeSSBs(input)
	if len(result.Merged) != 0 {
		t.Errorf("expected no merged SSBs when profiles differ, got %d", len(result.Merged))
	}
	if len(result.Problems) != 1 {
		t.Fatalf("expected 1 problem, got %d", len(result.Problems))
	}

	problem := result.Problems[0]
	if problem.Severity != models.SeverityError {
		t.Errorf("expected error severity, got %v", problem.Severity)
	}
	if problem.Category != models.CategoryConflict {
		t.Errorf("expected conflict category, got %v", problem.Category)
	}
}

// TestIMP_MAP_020_DifferentScheduleError verifies that SSBs with the same name
// and profiles but different schedules produce an error.
func TestIMP_MAP_020_DifferentScheduleError(t *testing.T) {
	input := map[string][]MappedSSB{
		"cluster-1": {
			{
				Name:     "ssb-sched-conflict",
				Profiles: []string{"profile-x"},
				Payload: models.ACSCreatePayload{
					ScanName: "ssb-sched-conflict",
					ScanConfig: models.ACSBaseScanConfig{
						Profiles: []string{"profile-x"},
						ScanSchedule: &models.ACSSchedule{
							Hour:   10,
							Minute: 0,
						},
					},
					Clusters: []string{"cluster-1"},
				},
			},
		},
		"cluster-2": {
			{
				Name:     "ssb-sched-conflict",
				Profiles: []string{"profile-x"},
				Payload: models.ACSCreatePayload{
					ScanName: "ssb-sched-conflict",
					ScanConfig: models.ACSBaseScanConfig{
						Profiles: []string{"profile-x"},
						ScanSchedule: &models.ACSSchedule{
							Hour:   14,
							Minute: 30,
						},
					},
					Clusters: []string{"cluster-2"},
				},
			},
		},
	}

	result := MergeSSBs(input)
	if len(result.Merged) != 0 {
		t.Errorf("expected no merged SSBs when schedules differ, got %d", len(result.Merged))
	}
	if len(result.Problems) != 1 {
		t.Fatalf("expected 1 problem, got %d", len(result.Problems))
	}

	problem := result.Problems[0]
	if problem.Severity != models.SeverityError {
		t.Errorf("expected error severity, got %v", problem.Severity)
	}
}

// TestSSBsUniqueToEachCluster verifies that SSBs unique to each cluster are
// not merged.
func TestSSBsUniqueToEachCluster(t *testing.T) {
	input := map[string][]MappedSSB{
		"cluster-1": {
			{
				Name:     "ssb-unique-1",
				Profiles: []string{"profile-a"},
				Payload: models.ACSCreatePayload{
					ScanName: "ssb-unique-1",
					ScanConfig: models.ACSBaseScanConfig{
						Profiles: []string{"profile-a"},
					},
					Clusters: []string{"cluster-1"},
				},
			},
		},
		"cluster-2": {
			{
				Name:     "ssb-unique-2",
				Profiles: []string{"profile-b"},
				Payload: models.ACSCreatePayload{
					ScanName: "ssb-unique-2",
					ScanConfig: models.ACSBaseScanConfig{
						Profiles: []string{"profile-b"},
					},
					Clusters: []string{"cluster-2"},
				},
			},
		},
	}

	result := MergeSSBs(input)
	if len(result.Merged) != 2 {
		t.Fatalf("expected 2 merged SSBs (unique ones not merged), got %d", len(result.Merged))
	}

	names := []string{result.Merged[0].Name, result.Merged[1].Name}
	slices.Sort(names)
	expected := []string{"ssb-unique-1", "ssb-unique-2"}
	if !stringSlicesEqual(names, expected) {
		t.Errorf("expected SSB names %v, got %v", expected, names)
	}

	// Each should have only one cluster.
	for _, merged := range result.Merged {
		if len(merged.Payload.Clusters) != 1 {
			t.Errorf("expected 1 cluster for unique SSB %q, got %d", merged.Name, len(merged.Payload.Clusters))
		}
	}
}
