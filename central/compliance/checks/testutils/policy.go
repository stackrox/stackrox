package testutils

import (
	"github.com/stackrox/rox/central/compliance/framework/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/features"
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
		Fields: &storage.PolicyFields{
			ImageName: &storage.ImageNamePolicy{Registry: l.ImageRegistry},
			Cve:       l.CVE,
		},
	}
	if l.CVSSGreaterThan > 0 {
		p.Fields.Cvss = &storage.NumericalPolicy{Value: l.CVSSGreaterThan}
	}
	if l.EnvKey != "" || l.EnvValue != "" {
		p.Fields.Env = &storage.KeyValuePolicy{Key: l.EnvKey, Value: l.EnvValue}
	}
	if l.Enforced {
		p.EnforcementActions = append(p.EnforcementActions, storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT)
	}
	if features.BooleanPolicyLogic.Enabled() {
		if err := booleanpolicy.EnsureConverted(p); err != nil {
			panic(err)
		}
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
