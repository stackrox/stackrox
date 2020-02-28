package testutils

import (
	"github.com/stackrox/rox/central/compliance/framework/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/uuid"
)

// A LightPolicy is a lightweight policy struct that is very convenient to define in tests.
type LightPolicy struct {
	ID              string
	Name            string
	ImageRegistry   string
	Disabled        bool
	CVSSGreaterThan float32
	CVE             string
}

func (l *LightPolicy) convert() *storage.Policy {
	p := &storage.Policy{
		Id:       l.ID,
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
