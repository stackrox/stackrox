package storagetov2

import (
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
)

// ComplianceV2Profile converts V2 storage objects to V2 API objects
func ComplianceV2Profile(incoming *storage.ComplianceOperatorProfileV2) *v2.ComplianceProfile {
	rules := make([]*v2.ComplianceRule, 0, len(incoming.GetRules()))
	for _, rule := range incoming.GetRules() {
		rules = append(rules, &v2.ComplianceRule{
			Name: rule.GetRuleName(),
		})
	}

	return &v2.ComplianceProfile{
		Id:             incoming.GetId(),
		Name:           incoming.GetName(),
		ProfileVersion: incoming.GetProfileVersion(),
		ProductType:    incoming.GetProductType(),
		Standard:       incoming.GetStandard(),
		Description:    incoming.GetDescription(),
		Rules:          rules,
		Product:        incoming.GetProduct(),
		Title:          incoming.GetTitle(),
		Values:         incoming.GetValues(),
	}
}

// ComplianceV2Profiles converts V2 storage objects to V2 API objects
func ComplianceV2Profiles(incoming []*storage.ComplianceOperatorProfileV2) []*v2.ComplianceProfile {
	v2Profiles := make([]*v2.ComplianceProfile, 0, len(incoming))
	for _, profile := range incoming {
		v2Profiles = append(v2Profiles, ComplianceV2Profile(profile))
	}

	return v2Profiles
}

// ComplianceProfileSummary converts summary object to V2 API summary object
func ComplianceProfileSummary(incoming []*storage.ComplianceOperatorProfileV2, clusterIDs []string) []*v2.ComplianceProfileSummary {
	// incoming will contain all the profiles matching the clusters.  This is a non-distinct
	// list that needs to be reduced to a summary and only include profiles that match all profiles.
	profileClusterMap := make(map[string][]string, len(incoming))
	profileSummaryMap := make(map[string]*v2.ComplianceProfileSummary)

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
				Name:           summary.Name,
				ProductType:    summary.ProductType,
				Description:    summary.Description,
				Title:          summary.Title,
				RuleCount:      int32(len(summary.Rules)),
				ProfileVersion: summary.ProfileVersion,
			}
		}
	}

	summaries := make([]*v2.ComplianceProfileSummary, 0, len(profileSummaryMap))
	for k, v := range profileSummaryMap {
		// Verify the profile is in all clusters
		if len(profileClusterMap[k]) != len(clusterIDs) {
			continue
		}

		summaries = append(summaries, &v2.ComplianceProfileSummary{
			Name:           v.Name,
			ProductType:    v.ProductType,
			Description:    v.Description,
			Title:          v.Title,
			RuleCount:      v.RuleCount,
			ProfileVersion: v.ProfileVersion,
		})
	}

	return summaries
}
