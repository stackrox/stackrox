package storagetov2

import (
	"github.com/pkg/errors"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
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

	return &v2.ComplianceProfile{
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
		OperatorKind:   convertProfileOperatorKind(incoming.GetOperatorKind()),
	}
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
			profileSummaryMap[summary.GetName()] = &v2.ComplianceProfileSummary{
				Name:           summary.GetName(),
				ProductType:    summary.GetProductType(),
				Description:    summary.GetDescription(),
				Title:          summary.GetTitle(),
				RuleCount:      int32(len(summary.GetRules())),
				ProfileVersion: summary.GetProfileVersion(),
				Standards:      profileBenchmarkNameMap[summary.GetName()],
				OperatorKind:   convertProfileSummaryOperatorKind(summary.GetOperatorKind()),
			}
			orderedProfiles = append(orderedProfiles, summary.GetName())
		}
	}

	summaries := make([]*v2.ComplianceProfileSummary, 0, len(profileSummaryMap))
	for _, profileName := range orderedProfiles {
		summaries = append(summaries, profileSummaryMap[profileName])
	}

	return summaries
}

func convertProfileOperatorKind(kind storage.ComplianceOperatorProfileV2_OperatorKind) v2.ComplianceProfile_OperatorKind {
	switch kind {
	case storage.ComplianceOperatorProfileV2_PROFILE:
		return v2.ComplianceProfile_PROFILE
	case storage.ComplianceOperatorProfileV2_TAILORED_PROFILE:
		return v2.ComplianceProfile_TAILORED_PROFILE
	case storage.ComplianceOperatorProfileV2_OPERATOR_KIND_UNSPECIFIED:
		// Older centrals may have stored profiles without OperatorKind,
		// so UNSPECIFIED is treated as PROFILE. This fallback can be removed when
		// versions that don't set OperatorKind (<= 4.10) are not supported.
		return v2.ComplianceProfile_PROFILE
	default:
		utils.Should(errors.Errorf("unhandled profile operator kind %s", kind))
		return v2.ComplianceProfile_OPERATOR_KIND_UNSPECIFIED
	}
}

func convertProfileSummaryOperatorKind(kind storage.ComplianceOperatorProfileV2_OperatorKind) v2.ComplianceProfileSummary_OperatorKind {
	switch kind {
	case storage.ComplianceOperatorProfileV2_PROFILE:
		return v2.ComplianceProfileSummary_PROFILE
	case storage.ComplianceOperatorProfileV2_TAILORED_PROFILE:
		return v2.ComplianceProfileSummary_TAILORED_PROFILE
	case storage.ComplianceOperatorProfileV2_OPERATOR_KIND_UNSPECIFIED:
		return v2.ComplianceProfileSummary_PROFILE
	default:
		utils.Should(errors.Errorf("unhandled profile summary operator kind %s", kind))
		return v2.ComplianceProfileSummary_OPERATOR_KIND_UNSPECIFIED
	}
}
