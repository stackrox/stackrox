package booleanpolicy

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestPolicyValueValidator(t *testing.T) {
	suite.Run(t, new(PolicyValueValidator))
}

type PolicyValueValidator struct {
	suite.Suite
}

func (s *PolicyValueValidator) TestRegex() {
	cases := []struct {
		name    string
		valid   []string
		invalid []string
		r       *regexp.Regexp
	}{
		{
			name:    "Decimal with comparator",
			valid:   []string{"0", ">0", "<=1.2", ".1", "0.1", ">=0.1"},
			invalid: []string{"", "0<", ">", "3>0", "."},
			r:       comparatorDecimalValueRegex,
		},
		{
			name:    "Integer",
			valid:   []string{"0", "12", "1", "111111"},
			invalid: []string{"", "0<", ">", "3>0", ".", ".1", "0.1"},
			r:       integerValueRegex,
		},
		{
			name:    "Boolean",
			valid:   []string{"true", "false", "False"},
			invalid: []string{"", "asdf", "FALS", "trueFalse", "falsef"},
			r:       booleanValueRegex,
		},
		{
			name:    "Dockerfile Line",
			valid:   []string{"ADD=.", "=.", "ADD=", "="},
			invalid: []string{"", "ADD"},
			r:       dockerfileLineValueRegex,
		},
		{
			name:    "Key Value",
			valid:   []string{"a=b", `.*\d=.*`, "1=1"},
			invalid: []string{"", "=", "=a=b"},
			r:       keyValueValueRegex,
		},
		{
			name:    "Environment Variable",
			valid:   []string{"UNKNOWN=ENV=a", "UNSET=ENV=a", "RAW=ENV=a", "CONFIG_MAP_KEY=key=", "FIELD=key=", "RESOURCE_FIELD=key=", "SECRET_KEY==", "=ENV=a", "==", "==="},
			invalid: []string{"", "a=", "a=b", "=", "=1", "SECRET_KEY", "a=ENV=a", "a==", "CONFIG_MAP_KEY=ENV=a", "SECRET_KEY=a=1", "FIELD=ENV=a", "RESOURCE_FIELD=ENV=a", "SECRET_KEY=e0=.", "SECRET_KEY=a=="},
			r:       environmentVariableWithSourceStrictRegex,
		},
		{
			name:    "String",
			valid:   []string{"a", "\n\n.\n\n", " a\n", " a"},
			invalid: []string{"", " ", "\n"},
			r:       stringValueRegex,
		},
		{
			name:    "capabilities",
			valid:   []string{"SYS_ADMIN"},
			invalid: []string{"", "CAP_N_CRUNCH", "CAP_SYS_ADMIN", "N_CRUNCH"},
			r:       capabilitiesValueRegex,
		},
		{
			name:    "cve",
			valid:   []string{"CVE-2020-0001", "cve-1-1"},
			invalid: []string{"", "\n", " "},
			r:       stringValueRegex,
		},
		{
			name:    "rbac permission",
			valid:   []string{"Elevated_Cluster_Wide", "CLUSTER_ADMIN"},
			invalid: []string{"", " ", "asdf"},
			r:       rbacPermissionValueRegex,
		},
		{
			name:    "port value",
			valid:   []string{"22", "8000"},
			invalid: []string{" ", ".", "-1", "3.0"},
			r:       integerValueRegex,
		},
		{
			name:    "port exposure",
			valid:   []string{"NODE", "Host"},
			invalid: []string{"", " "},
			r:       portExposureValueRegex,
		},
	}

	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {
			for _, valid := range c.valid {
				assert.Equal(t, true, c.r.MatchString(valid), valid)
			}
			for _, invalid := range c.invalid {
				assert.Equal(t, false, c.r.MatchString(invalid), invalid)
			}
		})
	}
}

func TestEnvKeyValuePolicyValidation(t *testing.T) {
	for _, p := range []storage.ContainerConfig_EnvironmentConfig_EnvVarSource{
		storage.ContainerConfig_EnvironmentConfig_UNSET,
		storage.ContainerConfig_EnvironmentConfig_UNKNOWN,
		storage.ContainerConfig_EnvironmentConfig_RAW,
	} {
		assert.NoError(t, Validate(&storage.Policy{
			Name:          "some-policy",
			PolicyVersion: CurrentVersion().String(),
			Fields: &storage.PolicyFields{
				Env: &storage.KeyValuePolicy{
					Key:          "key",
					Value:        "value",
					EnvVarSource: p,
				},
			},
			PolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.EnvironmentVariable,
							Values: []*storage.PolicyValue{
								{
									Value: fmt.Sprintf("%s=key=value", p),
								},
							},
						},
					},
				},
			},
		}, ValidateEnvVarSourceRestrictions()))

		assert.NoError(t, Validate(&storage.Policy{
			Name:          "some-policy",
			PolicyVersion: CurrentVersion().String(),
			Fields: &storage.PolicyFields{
				Env: &storage.KeyValuePolicy{
					Key:          "key",
					EnvVarSource: p,
				},
			},
			PolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.EnvironmentVariable,
							Values: []*storage.PolicyValue{
								{
									Value: fmt.Sprintf("%s=key=", p),
								},
							},
						},
					},
				},
			},
		}, ValidateEnvVarSourceRestrictions()))
	}

	for _, p := range []storage.ContainerConfig_EnvironmentConfig_EnvVarSource{
		storage.ContainerConfig_EnvironmentConfig_SECRET_KEY,
		storage.ContainerConfig_EnvironmentConfig_CONFIG_MAP_KEY,
		storage.ContainerConfig_EnvironmentConfig_FIELD,
		storage.ContainerConfig_EnvironmentConfig_RESOURCE_FIELD,
	} {
		assert.Error(t, Validate(&storage.Policy{
			Name:          "some-policy",
			PolicyVersion: CurrentVersion().String(),
			Fields: &storage.PolicyFields{
				Env: &storage.KeyValuePolicy{
					Key:          "key",
					Value:        "value",
					EnvVarSource: p,
				},
			},
			PolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.EnvironmentVariable,
							Values: []*storage.PolicyValue{
								{
									Value: fmt.Sprintf("%s=key=value", p),
								},
							},
						},
					},
				},
			},
		}, ValidateEnvVarSourceRestrictions()))

		assert.NoError(t, Validate(&storage.Policy{
			Name:          "some-policy",
			PolicyVersion: CurrentVersion().String(),
			Fields: &storage.PolicyFields{
				Env: &storage.KeyValuePolicy{
					Key:          "key",
					EnvVarSource: p,
				},
			},
			PolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.EnvironmentVariable,
							Values: []*storage.PolicyValue{
								{
									Value: fmt.Sprintf("%s=key=", p),
								},
							},
						},
					},
				},
			},
		}, ValidateEnvVarSourceRestrictions()))
	}
}

func TestValidateMultipleSections(t *testing.T) {
	group := &storage.PolicyGroup{FieldName: fieldnames.CVE, Values: []*storage.PolicyValue{{Value: "CVE-2017-1234"}}}
	assert.NoError(t, Validate(&storage.Policy{Name: "name", PolicyVersion: CurrentVersion().String(), PolicySections: []*storage.PolicySection{
		{SectionName: "good", PolicyGroups: []*storage.PolicyGroup{group}},
	}}))
	assert.Error(t, Validate(&storage.Policy{Name: "name", PolicyVersion: CurrentVersion().String(), PolicySections: []*storage.PolicySection{
		{SectionName: "bad", PolicyGroups: []*storage.PolicyGroup{group, group}},
	}}))
}
