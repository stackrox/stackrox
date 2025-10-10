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
	id := stringutils.OrDefault(l.ID, uuid.NewV4().String())
	name := l.Name
	disabled := l.Disabled
	notifiers := l.Notifiers
	policyVersion := "1.1"
	sectionName := "section-1"

	// Create PolicyValue for ImageRegistry
	imageRegistryValue := l.ImageRegistry
	imageRegistryPolicyValue := storage.PolicyValue_builder{
		Value: &imageRegistryValue,
	}.Build()

	// Create PolicyValue for CVE
	cveValue := l.CVE
	cvePolicyValue := storage.PolicyValue_builder{
		Value: &cveValue,
	}.Build()

	// Create PolicyGroup for ImageRegistry
	imageRegistryFieldName := fieldnames.ImageRegistry
	imageRegistryValues := []*storage.PolicyValue{imageRegistryPolicyValue}
	imageRegistryGroup := storage.PolicyGroup_builder{
		FieldName: &imageRegistryFieldName,
		Values:    imageRegistryValues,
	}.Build()

	// Create PolicyGroup for CVE
	cveFieldName := fieldnames.CVE
	cveValues := []*storage.PolicyValue{cvePolicyValue}
	cveGroup := storage.PolicyGroup_builder{
		FieldName: &cveFieldName,
		Values:    cveValues,
	}.Build()

	// Create PolicySection
	policyGroups := []*storage.PolicyGroup{imageRegistryGroup, cveGroup}
	policySection := storage.PolicySection_builder{
		SectionName:  &sectionName,
		PolicyGroups: policyGroups,
	}.Build()

	policySections := []*storage.PolicySection{policySection}

	p := storage.Policy_builder{
		Id:             &id,
		Name:           &name,
		Disabled:       &disabled,
		PolicySections: policySections,
		Notifiers:      notifiers,
		PolicyVersion:  &policyVersion,
	}.Build()
	if l.CVSSGreaterThan > 0 {
		s := fmt.Sprintf("> %0.3f", l.CVSSGreaterThan)

		// Create PolicyValue for CVSS
		cvssValue := storage.PolicyValue_builder{
			Value: &s,
		}.Build()

		// Create PolicyGroup for CVSS
		cvssFieldName := fieldnames.CVSS
		cvssValues := []*storage.PolicyValue{cvssValue}
		cvssGroup := storage.PolicyGroup_builder{
			FieldName: &cvssFieldName,
			Values:    cvssValues,
		}.Build()

		p.PolicySections[0].PolicyGroups = append(p.PolicySections[0].PolicyGroups, cvssGroup)
	}
	if l.EnvKey != "" || l.EnvValue != "" {
		// Create PolicyValue for EnvironmentVariable
		envValueStr := fmt.Sprintf("=%s=%s", l.EnvKey, l.EnvValue)
		envValue := storage.PolicyValue_builder{
			Value: &envValueStr,
		}.Build()

		// Create PolicyGroup for EnvironmentVariable
		envFieldName := fieldnames.EnvironmentVariable
		envValues := []*storage.PolicyValue{envValue}
		envGroup := storage.PolicyGroup_builder{
			FieldName: &envFieldName,
			Values:    envValues,
		}.Build()

		p.PolicySections[0].PolicyGroups = append(p.PolicySections[0].PolicyGroups, envGroup)
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
