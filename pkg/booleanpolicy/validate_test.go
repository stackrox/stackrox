package booleanpolicy

import (
	"regexp"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/image/policies"
	"github.com/stackrox/rox/pkg/defaults"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestPolicyValueValidator(t *testing.T) {
	suite.Run(t, new(PolicyValueValidator))
}

type PolicyValueValidator struct {
	suite.Suite
}

func (s *PolicyValueValidator) TestValidationDefaultPolicies() {
	envIsolator := testutils.NewEnvIsolator(s.T())
	defer envIsolator.RestoreAll()
	envIsolator.Setenv(features.BooleanPolicyLogic.EnvVar(), "true")

	defaults.PoliciesPath = policies.Directory()
	defaultPolicies, err := defaults.Policies()
	s.Require().NoError(err)

	policies := make(map[string]*storage.Policy, len(defaultPolicies))
	for _, legacyPolicy := range defaultPolicies {
		policies[legacyPolicy.GetName()], err = CloneAndEnsureConverted(legacyPolicy)
		assert.Nil(s.T(), err)
	}
	for _, p := range policies {
		s.T().Run(p.GetName(), func(t *testing.T) {
			assert.NoError(s.T(), Validate(p), p)
		})
	}
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
			valid:   []string{"ADD=."},
			invalid: []string{"", "ADD", "=."},
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
			valid:   []string{"CONFIG_MAP_KEY=ENV=a", "SECRET_KEY=a=1", "UNSET=ENV=a", "SECRET_KEY=e0=.", "SECRET_KEY=a==", "SECRET_KEY=="},
			invalid: []string{"", "a=", "a=b", "=", "==", "===", "=1", "=ENV=a", "SECRET_KEY", "a=ENV=a", "a=="},
			r:       environmentVariableWithSourceRegex,
		},
		{
			name:    "String",
			valid:   []string{"a", "\n\n.\n\n", " a\n", " a"},
			invalid: []string{"", " ", "\n"},
			r:       stringValueRegex,
		},
		{
			name:    "capabilities",
			valid:   []string{"CAP_SYS_ADMIN"},
			invalid: []string{"", "CAP_N_CRUNCH"},
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
