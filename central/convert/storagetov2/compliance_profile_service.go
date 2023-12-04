package storagetov2

import (
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
)

// ComplianceV2Profile converts V2 storage objects to V2 API objects
func ComplianceV2Profile(incoming *storage.ComplianceOperatorProfileV2) *v2.ComplianceProfile {
	var rules []*v2.ComplianceRule
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
	var v2Profiles []*v2.ComplianceProfile
	for _, profile := range incoming {
		v2Profiles = append(v2Profiles, ComplianceV2Profile(profile))
	}

	return v2Profiles
}
