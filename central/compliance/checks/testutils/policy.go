package testutils

import (
	"fmt"

	"github.com/stackrox/rox/central/compliance/framework/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/uuid"
)

// A LightPolicy is a lightweight policy struct that is very convenient to define in tests.
type LightPolicy struct {
	ID       string
	Name     string
	Disabled bool
	Enforced bool

	ImageRegistry string
	Notifiers     []string

	CVSSGreaterThan float32
	CVE             string

	EnvKey   string
	EnvValue string
}

func (l *LightPolicy) convert() *storage.Policy {
	p := storage.Policy_builder{
		Id:       stringutils.OrDefault(l.ID, uuid.NewV4().String()),
		Name:     l.Name,
		Disabled: l.Disabled,
		PolicySections: []*storage.PolicySection{
			storage.PolicySection_builder{
				SectionName: "section-1",
				PolicyGroups: []*storage.PolicyGroup{
					storage.PolicyGroup_builder{
						FieldName: fieldnames.ImageRegistry,
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: l.ImageRegistry,
							}.Build(),
						},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: fieldnames.CVE,
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: l.CVE,
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
		},
		Notifiers:     l.Notifiers,
		PolicyVersion: "1.1",
	}.Build()
	if l.CVSSGreaterThan > 0 {
		s := fmt.Sprintf("> %0.3f", l.CVSSGreaterThan)
		pv := &storage.PolicyValue{}
		pv.SetValue(s)
		pg := &storage.PolicyGroup{}
		pg.SetFieldName(fieldnames.CVSS)
		pg.SetValues([]*storage.PolicyValue{
			pv,
		})
		p.GetPolicySections()[0].SetPolicyGroups(append(p.GetPolicySections()[0].GetPolicyGroups(), pg))
	}
	if l.EnvKey != "" || l.EnvValue != "" {
		pv := &storage.PolicyValue{}
		pv.SetValue(fmt.Sprintf("=%s=%s", l.EnvKey, l.EnvValue))
		pg := &storage.PolicyGroup{}
		pg.SetFieldName(fieldnames.EnvironmentVariable)
		pg.SetValues([]*storage.PolicyValue{
			pv,
		})
		p.GetPolicySections()[0].SetPolicyGroups(append(p.GetPolicySections()[0].GetPolicyGroups(), pg))
	}
	if l.Enforced {
		p.SetEnforcementActions(append(p.GetEnforcementActions(), storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT))
	}

	return p
}

// MockOutLightPolicies injects the given light policies into the mock data repository.
func MockOutLightPolicies(mockData *mocks.MockComplianceDataRepository, policies []LightPolicy) {
	policiesMap := make(map[string]*storage.Policy)
	for _, p := range policies {
		name := stringutils.OrDefault(p.Name, uuid.NewV4().String())
		policiesMap[name] = p.convert()
	}
	mockData.EXPECT().Policies().Return(policiesMap)
}
