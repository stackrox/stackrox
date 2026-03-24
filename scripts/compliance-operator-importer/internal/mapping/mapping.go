package mapping

import (
	"fmt"
	"slices"

	"github.com/stackrox/co-acs-importer/internal/cofetch"
	"github.com/stackrox/co-acs-importer/internal/models"
)

// MappingResult is returned per ScanSettingBinding.
// Exactly one of Payload or Problem will be non-nil.
type MappingResult struct {
	// Payload is non-nil on success and contains the ACS create payload.
	Payload *models.ACSCreatePayload
	// Problem is non-nil when the binding should be skipped, with details about why.
	Problem *models.Problem
}

// MapBinding converts one ScanSettingBinding and its referenced ScanSetting into an
// ACS create payload, or returns a Problem if the binding should be skipped.
//
// Rules applied:
//   - IMP-MAP-001: scanName = binding.Name; profiles = sorted+deduped list of profile names.
//   - IMP-MAP-002: missing profile kind defaults to "Profile" (ProfileRef.Kind is "" => Profile).
//   - IMP-MAP-003: oneTimeScan=false when a schedule is present.
//   - IMP-MAP-004: scanSchedule set from ConvertCronToACSSchedule.
//   - IMP-MAP-005: description contains "Imported from CO ScanSettingBinding <ns>/<name>".
//   - IMP-MAP-006: description includes the ScanSetting name.
//   - IMP-MAP-007: clusters = [cfg.ACSClusterID].
//   - IMP-MAP-008..011: nil ScanSetting => Problem{category:mapping, skipped:true}.
//   - IMP-MAP-012..015: invalid cron => Problem{category:mapping, skipped:true}.
func MapBinding(binding cofetch.ScanSettingBinding, ss *cofetch.ScanSetting, cfg *models.Config) MappingResult {
	ref := fmt.Sprintf("%s/%s", binding.Namespace, binding.Name)

	// IMP-MAP-008, IMP-MAP-009, IMP-MAP-010: missing ScanSetting.
	if ss == nil {
		return MappingResult{
			Problem: &models.Problem{
				Severity:    models.SeverityError,
				Category:    models.CategoryMapping,
				ResourceRef: ref,
				Description: fmt.Sprintf(
					"ScanSettingBinding %q references ScanSetting %q which could not be found",
					ref, binding.ScanSettingName,
				),
				FixHint: fmt.Sprintf(
					"Ensure ScanSetting %q exists in namespace %q and is readable by the importer. "+
						"Verify with: kubectl get scansetting %s -n %s",
					binding.ScanSettingName, binding.Namespace,
					binding.ScanSettingName, binding.Namespace,
				),
				Skipped: true,
			},
		}
	}

	// IMP-MAP-004, IMP-MAP-012..015: convert cron schedule.
	schedule, err := ConvertCronToACSSchedule(ss.Schedule)
	if err != nil {
		return MappingResult{
			Problem: &models.Problem{
				Severity:    models.SeverityError,
				Category:    models.CategoryMapping,
				ResourceRef: ref,
				Description: fmt.Sprintf(
					"schedule conversion failed for ScanSettingBinding %q (ScanSetting %q, schedule %q): %v",
					ref, ss.Name, ss.Schedule, err,
				),
				FixHint: fmt.Sprintf(
					"Update ScanSetting %q to use a supported 5-field cron expression, for example: "+
						"\"0 2 * * *\" (daily at 02:00), \"0 2 * * 0\" (weekly on Sunday), "+
						"\"0 2 1 * *\" (monthly on the 1st). "+
						"Step and range notation in the cron expression are not supported.",
					ss.Name,
				),
				Skipped: true,
			},
		}
	}

	// IMP-MAP-001, IMP-MAP-002: collect profiles, dedup, sort.
	// ProfileRef.Kind being empty is equivalent to "Profile" (IMP-MAP-002).
	// Only the profile name is used in the ACS payload; kind determines lookup but
	// both Profile and TailoredProfile names go into the same ACS profiles list.
	profileSet := make(map[string]struct{}, len(binding.Profiles))
	for _, p := range binding.Profiles {
		profileSet[p.Name] = struct{}{}
	}
	profiles := make([]string, 0, len(profileSet))
	for name := range profileSet {
		profiles = append(profiles, name)
	}
	slices.Sort(profiles) // IMP-MAP-001: deterministic sorted order

	// IMP-MAP-005, IMP-MAP-006: build description.
	description := fmt.Sprintf(
		"Imported from CO ScanSettingBinding %s/%s (ScanSetting: %s)",
		binding.Namespace, binding.Name, ss.Name,
	)

	return MappingResult{
		Payload: &models.ACSCreatePayload{
			ScanName: binding.Name, // IMP-MAP-001
			ScanConfig: models.ACSBaseScanConfig{
				OneTimeScan:  false,       // IMP-MAP-003
				Profiles:     profiles,    // IMP-MAP-001
				ScanSchedule: schedule,    // IMP-MAP-004
				Description:  description, // IMP-MAP-005, IMP-MAP-006
			},
			Clusters: []string{cfg.ACSClusterID}, // IMP-MAP-007
		},
	}
}
