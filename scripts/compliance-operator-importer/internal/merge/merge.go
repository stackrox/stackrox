// Package merge handles merging of ScanSettingBindings across multiple clusters.
package merge

import (
	"fmt"
	"slices"
	"strings"

	"github.com/stackrox/co-acs-importer/internal/models"
)

// MappedSSB represents a ScanSettingBinding that has been mapped to an ACS payload.
type MappedSSB struct {
	Name     string // SSB name
	Profiles []string
	Payload  models.ACSCreatePayload
}

// MergeResult holds the output of the merge operation.
type MergeResult struct {
	Merged   []MappedSSB
	Problems []models.Problem
}

// MergeSSBs merges ScanSettingBindings from multiple clusters.
//
// Input: map of clusterID → []MappedSSB
// Output: []MergedSSB (one per unique SSB name, with merged cluster IDs)
//
// Logic (IMP-MAP-019, IMP-MAP-020, IMP-MAP-021):
//   - Group by SSB name across all clusters.
//   - For each group:
//   - If all SSBs have identical profiles (sorted) and identical schedule: merge into one, union cluster IDs.
//   - If profiles or schedule differ: error for that SSB name, add problem entry.
func MergeSSBs(clusterSSBs map[string][]MappedSSB) MergeResult {
	// Group SSBs by name.
	groups := make(map[string][]clusterSSBEntry)
	for clusterID, ssbs := range clusterSSBs {
		for _, ssb := range ssbs {
			groups[ssb.Name] = append(groups[ssb.Name], clusterSSBEntry{
				clusterID: clusterID,
				ssb:       ssb,
			})
		}
	}

	var merged []MappedSSB
	var problems []models.Problem

	for ssbName, entries := range groups {
		if len(entries) == 1 {
			// Only one cluster has this SSB; no merging needed.
			merged = append(merged, entries[0].ssb)
			continue
		}

		// Check if all SSBs in the group are identical (same profiles and schedule).
		first := entries[0].ssb
		identical := true
		var conflictClusters []string

		for _, entry := range entries[1:] {
			if !ssbsAreIdentical(first, entry.ssb) {
				identical = false
				conflictClusters = append(conflictClusters, entry.clusterID)
			}
		}

		if !identical {
			// IMP-MAP-020: profiles or schedule differ.
			conflictClusters = append([]string{entries[0].clusterID}, conflictClusters...)
			problems = append(problems, models.Problem{
				Severity:    models.SeverityError,
				Category:    models.CategoryConflict,
				ResourceRef: ssbName,
				Description: fmt.Sprintf(
					"ScanSettingBinding %q exists in multiple clusters with different profiles or schedules: %s",
					ssbName, strings.Join(conflictClusters, ", "),
				),
				FixHint: "Ensure SSBs with the same name have identical profiles and schedules across all clusters, or rename them uniquely per cluster.",
				Skipped: true,
			})
			continue
		}

		// IMP-MAP-019, IMP-MAP-021: merge clusters.
		mergedSSB := first
		var allClusters []string
		for _, entry := range entries {
			allClusters = append(allClusters, entry.clusterID)
		}
		slices.Sort(allClusters)
		mergedSSB.Payload.Clusters = allClusters
		merged = append(merged, mergedSSB)
	}

	return MergeResult{
		Merged:   merged,
		Problems: problems,
	}
}

// clusterSSBEntry pairs a cluster ID with an SSB.
type clusterSSBEntry struct {
	clusterID string
	ssb       MappedSSB
}

// ssbsAreIdentical checks if two SSBs have the same profiles and schedule.
func ssbsAreIdentical(a, b MappedSSB) bool {
	// Compare sorted profiles.
	aProfiles := make([]string, len(a.Profiles))
	bProfiles := make([]string, len(b.Profiles))
	copy(aProfiles, a.Profiles)
	copy(bProfiles, b.Profiles)
	slices.Sort(aProfiles)
	slices.Sort(bProfiles)

	if !stringSlicesEqual(aProfiles, bProfiles) {
		return false
	}

	// Compare schedules.
	return schedulesEqual(a.Payload.ScanConfig.ScanSchedule, b.Payload.ScanConfig.ScanSchedule)
}

// stringSlicesEqual checks if two string slices are equal.
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// schedulesEqual checks if two ACS schedules are equal.
func schedulesEqual(a, b *models.ACSSchedule) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	if a.Hour != b.Hour || a.Minute != b.Minute {
		return false
	}

	if a.IntervalType != b.IntervalType {
		return false
	}

	// Compare DaysOfWeek.
	if (a.DaysOfWeek == nil) != (b.DaysOfWeek == nil) {
		return false
	}
	if a.DaysOfWeek != nil && b.DaysOfWeek != nil {
		if !int32SlicesEqual(a.DaysOfWeek.Days, b.DaysOfWeek.Days) {
			return false
		}
	}

	// Compare DaysOfMonth.
	if (a.DaysOfMonth == nil) != (b.DaysOfMonth == nil) {
		return false
	}
	if a.DaysOfMonth != nil && b.DaysOfMonth != nil {
		if !int32SlicesEqual(a.DaysOfMonth.Days, b.DaysOfMonth.Days) {
			return false
		}
	}

	return true
}

// int32SlicesEqual checks if two int32 slices are equal.
func int32SlicesEqual(a, b []int32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
