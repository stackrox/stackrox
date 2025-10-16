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
	p := &storage.Policy{
		Id:       stringutils.OrDefault(l.ID, uuid.NewV4().String()),
		Name:     l.Name,
		Disabled: l.Disabled,
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "section-1",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.ImageRegistry,
						Values: []*storage.PolicyValue{
							{
								Value: l.ImageRegistry,
							},
						},
					},
					{
						FieldName: fieldnames.CVE,
						Values: []*storage.PolicyValue{
							{
								Value: l.CVE,
							},
						},
					},
				},
			},
		},
		Notifiers:     l.Notifiers,
		PolicyVersion: "1.1",
	}
	if l.CVSSGreaterThan > 0 {
		s := fmt.Sprintf("> %0.3f", l.CVSSGreaterThan)
		p.PolicySections[0].PolicyGroups = append(p.PolicySections[0].PolicyGroups, &storage.PolicyGroup{
			FieldName: fieldnames.CVSS,
			Values: []*storage.PolicyValue{
				{
					Value: s,
				},
			},
		})
	}
	if l.EnvKey != "" || l.EnvValue != "" {
		p.PolicySections[0].PolicyGroups = append(p.PolicySections[0].PolicyGroups, &storage.PolicyGroup{
			FieldName: fieldnames.EnvironmentVariable,
			Values: []*storage.PolicyValue{
				{
					Value: fmt.Sprintf("=%s=%s", l.EnvKey, l.EnvValue),
				},
			},
		})
	}
	if l.Enforced {
		p.EnforcementActions = append(p.EnforcementActions, storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT)
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
