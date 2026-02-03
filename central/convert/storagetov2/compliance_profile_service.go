package storagetov2

import (
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
)

// ComplianceV2Profile converts V2 storage objects to V2 API objects
func ComplianceV2Profile(incoming *storage.ComplianceOperatorProfileV2, benchmarks []*storage.ComplianceOperatorBenchmarkV2) *v2.ComplianceProfile {
	rules := make([]*v2.ComplianceRule, 0, len(incoming.GetRules()))
	for _, rule := range incoming.GetRules() {
		rules = append(rules, &v2.ComplianceRule{
			Name: rule.GetRuleName(),
		})
	}

	convertedBenchmarks := make([]*v2.ComplianceBenchmark, 0, len(benchmarks))
	for _, benchmark := range benchmarks {
		convertedBenchmarks = append(convertedBenchmarks, &v2.ComplianceBenchmark{
			Name:        benchmark.GetName(),
			Version:     benchmark.GetVersion(),
			Description: benchmark.GetDescription(),
			Provider:    benchmark.GetProvider(),
			ShortName:   benchmark.GetShortName(),
		})
	}

	profile := &v2.ComplianceProfile{
		Id:             incoming.GetId(),
		Name:           incoming.GetName(),
		ProfileVersion: incoming.GetProfileVersion(),
		ProductType:    incoming.GetProductType(),
		Standards:      convertedBenchmarks,
		Description:    incoming.GetDescription(),
		Rules:          rules,
		Product:        incoming.GetProduct(),
		Title:          incoming.GetTitle(),
		Values:         incoming.GetValues(),
	}

	if td := incoming.GetTailoredDetails(); td != nil {
		profile.TailoredDetails = convertTailoredProfileDetailsToV2(td)
	}

	return profile
}

// convertTailoredProfileDetailsToV2 converts storage tailored profile details to API v2 format
func convertTailoredProfileDetailsToV2(td *storage.StorageTailoredProfileDetails) *v2.TailoredProfileDetails {
	result := &v2.TailoredProfileDetails{
		Extends:      td.GetExtends(),
		State:        td.GetState(),
		ErrorMessage: td.GetErrorMessage(),
	}

	for _, r := range td.GetDisabledRules() {
		result.DisabledRules = append(result.DisabledRules, &v2.TailoredProfileRuleModification{
			Name:      r.GetName(),
			Rationale: r.GetRationale(),
		})
	}

	for _, r := range td.GetEnabledRules() {
		result.EnabledRules = append(result.EnabledRules, &v2.TailoredProfileRuleModification{
			Name:      r.GetName(),
			Rationale: r.GetRationale(),
		})
	}

	for _, r := range td.GetManualRules() {
		result.ManualRules = append(result.ManualRules, &v2.TailoredProfileRuleModification{
			Name:      r.GetName(),
			Rationale: r.GetRationale(),
		})
	}

	for _, v := range td.GetSetValues() {
		result.SetValues = append(result.SetValues, &v2.TailoredProfileValueOverride{
			Name:      v.GetName(),
			Value:     v.GetValue(),
			Rationale: v.GetRationale(),
		})
	}

	return result
}

// ComplianceV2Profiles converts V2 storage objects to V2 API objects
func ComplianceV2Profiles(incoming []*storage.ComplianceOperatorProfileV2, benchmarkProfileMap map[string][]*storage.ComplianceOperatorBenchmarkV2) []*v2.ComplianceProfile {
	v2Profiles := make([]*v2.ComplianceProfile, 0, len(incoming))
	for _, profile := range incoming {
		v2Profiles = append(v2Profiles, ComplianceV2Profile(profile, benchmarkProfileMap[profile.GetName()]))
	}

	return v2Profiles
}

// ComplianceProfileSummary converts summary object to V2 API summary object
func ComplianceProfileSummary(incoming []*storage.ComplianceOperatorProfileV2, benchmarkProfileMap map[string][]*storage.ComplianceOperatorBenchmarkV2) []*v2.ComplianceProfileSummary {
	// incoming will contain all the profiles matching the clusters.  This is a non-distinct
	// list that needs to be reduced to a summary and only include profiles that match all profiles.
	profileClusterMap := make(map[string][]string, len(incoming))
	profileSummaryMap := make(map[string]*v2.ComplianceProfileSummary)
	profileBenchmarkNameMap := make(map[string][]*v2.ComplianceBenchmark)

	for profileName, benchmarks := range benchmarkProfileMap {
		convertedBenchmarks := make([]*v2.ComplianceBenchmark, 0, len(benchmarks))
		for _, benchmark := range benchmarks {
			convertedBenchmarks = append(convertedBenchmarks, &v2.ComplianceBenchmark{
				Name:        benchmark.GetName(),
				Version:     benchmark.GetVersion(),
				Description: benchmark.GetDescription(),
				Provider:    benchmark.GetProvider(),
				ShortName:   benchmark.GetShortName(),
			})
		}
		profileBenchmarkNameMap[profileName] = convertedBenchmarks
	}

	// Used to maintain sort order from the query since maps are unordered.
	var orderedProfiles []string

	for _, summary := range incoming {
		profileClusters, clusterFound := profileClusterMap[summary.GetName()]

		// First time seeing this profile.
		if !clusterFound {
			profileClusterMap[summary.GetName()] = []string{summary.GetClusterId()}
		} else {
			// Append the new cluster status to the profile cluster map.
			profileClusterMap[summary.GetName()] = append(profileClusters, summary.GetClusterId())
		}
		if _, found := profileSummaryMap[summary.GetName()]; !found {
			summaryObj := &v2.ComplianceProfileSummary{
				Name:           summary.GetName(),
				ProductType:    summary.GetProductType(),
				Description:    summary.GetDescription(),
				Title:          summary.GetTitle(),
				RuleCount:      int32(len(summary.GetRules())),
				ProfileVersion: summary.GetProfileVersion(),
				Standards:      profileBenchmarkNameMap[summary.GetName()],
			}
			if td := summary.GetTailoredDetails(); td != nil {
				summaryObj.TailoredDetails = convertTailoredProfileDetailsToV2(td)
			}
			profileSummaryMap[summary.GetName()] = summaryObj
			orderedProfiles = append(orderedProfiles, summary.GetName())
		}
	}

	summaries := make([]*v2.ComplianceProfileSummary, 0, len(profileSummaryMap))
	for _, profileName := range orderedProfiles {
		summaries = append(summaries, profileSummaryMap[profileName])
	}

	return summaries
}
