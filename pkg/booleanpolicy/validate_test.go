package booleanpolicy

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
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
			name:    "addCapabilities",
			valid:   []string{"SYS_ADMIN"},
			invalid: []string{"", "CAP_N_CRUNCH", "CAP_SYS_ADMIN", "N_CRUNCH"},
			r:       addCapabilitiesValueRegex,
		},
		{
			name:    "dropCapabilities",
			valid:   []string{"ALL"},
			invalid: []string{"", "CAP_N_CRUNCH", "CAP_SYS_ADMIN", "N_CRUNCH"},
			r:       dropCapabilitiesValueRegex,
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

func (s *PolicyValueValidator) TestEnvKeyValuePolicyValidation() {
	for _, p := range []storage.ContainerConfig_EnvironmentConfig_EnvVarSource{
		storage.ContainerConfig_EnvironmentConfig_UNSET,
		storage.ContainerConfig_EnvironmentConfig_UNKNOWN,
		storage.ContainerConfig_EnvironmentConfig_RAW,
	} {
		assert.NoError(s.T(), Validate(&storage.Policy{
			Name:          "some-policy",
			PolicyVersion: policyversion.CurrentVersion().String(),
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

		assert.NoError(s.T(), Validate(&storage.Policy{
			Name:          "some-policy",
			PolicyVersion: policyversion.CurrentVersion().String(),
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
		assert.Error(s.T(), Validate(&storage.Policy{
			Name:          "some-policy",
			PolicyVersion: policyversion.CurrentVersion().String(),
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

		assert.NoError(s.T(), Validate(&storage.Policy{
			Name:          "some-policy",
			PolicyVersion: policyversion.CurrentVersion().String(),
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

func (s *PolicyValueValidator) TestValidateMultipleSections() {
	group := &storage.PolicyGroup{FieldName: fieldnames.CVE, Values: []*storage.PolicyValue{{Value: "CVE-2017-1234"}}}
	assert.NoError(s.T(), Validate(&storage.Policy{Name: "name", PolicyVersion: policyversion.CurrentVersion().String(), PolicySections: []*storage.PolicySection{
		{SectionName: "good", PolicyGroups: []*storage.PolicyGroup{group}},
	}}))
	assert.Error(s.T(), Validate(&storage.Policy{Name: "name", PolicyVersion: policyversion.CurrentVersion().String(), PolicySections: []*storage.PolicySection{
		{SectionName: "bad", PolicyGroups: []*storage.PolicyGroup{group, group}},
	}}))
}

func (s *PolicyValueValidator) TestValidateKubeResourceSpecifiedForAuditEventSource() {
	assert.NoError(s.T(), Validate(&storage.Policy{
		Name:            "runtime-policy-valid",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     storage.EventSource_AUDIT_LOG_EVENT,
		PolicyVersion:   policyversion.CurrentVersion().String(),
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.KubeResource,
						Values: []*storage.PolicyValue{
							{
								Value: "SECRETS",
							},
						},
					},
					{
						FieldName: fieldnames.KubeAPIVerb,
						Values: []*storage.PolicyValue{
							{
								Value: "GET",
							},
						},
					},
				},
			},
		},
	}, ValidateSourceIsAuditLogEvents()))

	assert.Error(s.T(), Validate(&storage.Policy{
		Name:            "runtime-policy-no-resource",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     storage.EventSource_AUDIT_LOG_EVENT,
		PolicyVersion:   policyversion.CurrentVersion().String(),
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.KubeAPIVerb,
						Values: []*storage.PolicyValue{
							{
								Value: "GET",
							},
						},
					},
				},
			},
		},
	}, ValidateSourceIsAuditLogEvents()))
}

func (s *PolicyValueValidator) TestValidateKubeAPIVerbSpecifiedForAuditEventSource() {
	assert.NoError(s.T(), Validate(&storage.Policy{
		Name:            "runtime-policy-valid",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     storage.EventSource_AUDIT_LOG_EVENT,
		PolicyVersion:   policyversion.CurrentVersion().String(),
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.KubeResource,
						Values: []*storage.PolicyValue{
							{
								Value: "SECRETS",
							},
						},
					},
					{
						FieldName: fieldnames.KubeAPIVerb,
						Values: []*storage.PolicyValue{
							{
								Value: "GET",
							},
						},
					},
				},
			},
		},
	}, ValidateSourceIsAuditLogEvents()))

	assert.Error(s.T(), Validate(&storage.Policy{
		Name:            "runtime-policy-no-resource",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     storage.EventSource_AUDIT_LOG_EVENT,
		PolicyVersion:   policyversion.CurrentVersion().String(),
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.KubeResource,
						Values: []*storage.PolicyValue{
							{
								Value: "SECRETS",
							},
						},
					},
				},
			},
		},
	}, ValidateSourceIsAuditLogEvents()))
}

func (s *PolicyValueValidator) TestValidatePolicyCriteriaForAuditEventSource() {
	assert.NoError(s.T(), Validate(&storage.Policy{
		Name:            "runtime-policy-valid-criteria",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     storage.EventSource_AUDIT_LOG_EVENT,
		PolicyVersion:   policyversion.CurrentVersion().String(),
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.KubeResource,
						Values: []*storage.PolicyValue{
							{
								Value: "SECRETS",
							},
						},
					},
					{
						FieldName: fieldnames.KubeAPIVerb,
						Values: []*storage.PolicyValue{
							{
								Value: "GET",
							},
						},
					},
				},
			},
		},
	}, ValidateSourceIsAuditLogEvents()))

	assert.Error(s.T(), Validate(&storage.Policy{
		Name:            "runtime-policy-incorrect-criteria",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     storage.EventSource_AUDIT_LOG_EVENT,
		PolicyVersion:   policyversion.CurrentVersion().String(),
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.KubeResource,
						Values: []*storage.PolicyValue{
							{
								Value: "CONFIGMAPS",
							},
						},
					},
					{
						FieldName: fieldnames.ProcessName,
						Values: []*storage.PolicyValue{
							{
								Value: "ps",
							},
						},
					},
				},
			},
		},
	}, ValidateSourceIsAuditLogEvents()))
}

func (s *PolicyValueValidator) TestValidatePolicyValueRegexForAuditEventSource() {
	cases := []struct {
		name         string
		resourceType string
		verb         string
		fieldName    string
		fieldValue   string
		errExpected  bool
	}{
		{
			name:         "Valid resource name and verb should not error",
			resourceType: "CONFIGMAPS",
			verb:         "GET",
			errExpected:  false,
		},
		{
			name:         "Valid resource name and verb regex are case insensitive",
			resourceType: "secrets",
			verb:         "patch",
			errExpected:  false,
		},
		{
			name:         "Invalid resource name should be an error",
			resourceType: "RBAC",
			verb:         "GET",
			errExpected:  true,
		},
		{
			name:         "Invalid verb should be an error",
			resourceType: "secrets",
			verb:         "WATCH",
			errExpected:  true,
		},
		{
			name:         "Valid resource name should not be an error",
			resourceType: "secrets",
			verb:         "GET",
			fieldName:    fieldnames.KubeResourceName,
			fieldValue:   "central-htpasswd",
			errExpected:  false,
		},
		{
			name:         "Valid user name should not be an error",
			resourceType: "secrets",
			verb:         "GET",
			fieldName:    fieldnames.KubeUserName,
			fieldValue:   "system:serviceaccount:openshift-kube-apiserver:localhost-recovery-client",
			errExpected:  false,
		},
		{
			name:         "Valid user group name should not be an error",
			resourceType: "secrets",
			verb:         "GET",
			fieldName:    fieldnames.KubeUserGroups,
			fieldValue:   "system:serviceaccounts:openshift-kube-apiserver",
			errExpected:  false,
		},
		{
			name:         "Valid IPV4 source address should not be an error",
			resourceType: "secrets",
			verb:         "GET",
			fieldName:    fieldnames.SourceIPAddress,
			fieldValue:   "10.0.0.2",
			errExpected:  false,
		},
		{
			name:         "Valid IPV6 source address should not be an error",
			resourceType: "secrets",
			verb:         "GET",
			fieldName:    fieldnames.SourceIPAddress,
			fieldValue:   "2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF",
			errExpected:  false,
		},
		{
			name:         "Valid user agent should not be an error",
			resourceType: "secrets",
			verb:         "GET",
			fieldName:    fieldnames.UserAgent,
			fieldValue:   "cluster-kube-apiserver-operator/v0.0.0 (linux/amd64) kubernetes/$Format",
			errExpected:  false,
		},
		{
			name:         "Valid user agent from oc should not be an error",
			resourceType: "secrets",
			verb:         "GET",
			fieldName:    fieldnames.UserAgent,
			fieldValue:   "oc/4.7.0 (darwin/amd64) kubernetes/c66c03f",
			errExpected:  false,
		},
		{
			name:         "Valid IsImpersonatedUser should not be an error",
			resourceType: "secrets",
			verb:         "GET",
			fieldName:    fieldnames.IsImpersonatedUser,
			fieldValue:   "true",
			errExpected:  false,
		},
		{
			name:         "Resource name should be case insensitive",
			resourceType: "secrets",
			verb:         "GET",
			fieldName:    fieldnames.KubeResourceName,
			fieldValue:   "central-HTPASSWD",
			errExpected:  false,
		},
		{
			name:         "Valid user name should be case insensitive",
			resourceType: "secrets",
			verb:         "GET",
			fieldName:    fieldnames.KubeUserName,
			fieldValue:   "system:SERVICEACCOUNT:openshift-kube-apiserver:localhost-recovery-client",
			errExpected:  false,
		},
		{
			name:         "Valid user group name should be case insensitive",
			resourceType: "secrets",
			verb:         "GET",
			fieldName:    fieldnames.KubeUserGroups,
			fieldValue:   "System:serviceaccounts:openshift-kube-apiserver",
			errExpected:  false,
		},
		{
			name:         "Valid user agent should be case insensitive",
			resourceType: "secrets",
			verb:         "GET",
			fieldName:    fieldnames.UserAgent,
			fieldValue:   "cluster-KUBE-apiserver-operator/v0.0.0 (linux/amd64) kubernetes/$Format",
			errExpected:  false,
		},
		{
			name:         "Valid IsImpersonatedUser should be case insensitive",
			resourceType: "secrets",
			verb:         "GET",
			fieldName:    fieldnames.IsImpersonatedUser,
			fieldValue:   "FALSE",
			errExpected:  false,
		},
		{
			name:         "If IsImpersonatedUser is not boolean it should be an error",
			resourceType: "secrets",
			verb:         "GET",
			fieldName:    fieldnames.IsImpersonatedUser,
			fieldValue:   "yea",
			errExpected:  true,
		},
		{
			name:         "Invalid source address should be an error",
			resourceType: "secrets",
			verb:         "GET",
			fieldName:    fieldnames.SourceIPAddress,
			fieldValue:   "999:999.999.222",
			errExpected:  true,
		},
	}

	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {
			policy := &storage.Policy{
				Name:            "audit-policy-should-pass-regex-checks",
				LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
				EventSource:     storage.EventSource_AUDIT_LOG_EVENT,
				PolicyVersion:   policyversion.CurrentVersion().String(),
				PolicySections: []*storage.PolicySection{
					{
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: fieldnames.KubeResource,
								Values:    []*storage.PolicyValue{{Value: c.resourceType}},
							},
							{
								FieldName: fieldnames.KubeAPIVerb,
								Values:    []*storage.PolicyValue{{Value: c.verb}},
							},
						},
					},
				},
			}

			if c.fieldName != "" {
				group := &storage.PolicyGroup{
					FieldName: c.fieldName,
					Values:    []*storage.PolicyValue{{Value: c.fieldValue}},
				}
				policy.PolicySections[0].PolicyGroups = append(policy.PolicySections[0].PolicyGroups, group)
			}

			if c.errExpected {
				s.Error(Validate(policy, ValidateSourceIsAuditLogEvents()))
			} else {
				s.NoError(Validate(policy, ValidateSourceIsAuditLogEvents()))
			}
		})
	}
}

func (s *PolicyValueValidator) TestValidatePolicyHasCorrectVersion() {
	group := &storage.PolicyGroup{FieldName: fieldnames.CVE, Values: []*storage.PolicyValue{{Value: "CVE-2017-1234"}}}
	s.NoError(Validate(&storage.Policy{Name: "name", PolicyVersion: policyversion.CurrentVersion().String(), PolicySections: []*storage.PolicySection{
		{SectionName: "good", PolicyGroups: []*storage.PolicyGroup{group}},
	}}))

	s.Error(Validate(&storage.Policy{Name: "name", PolicyVersion: "", PolicySections: []*storage.PolicySection{
		{SectionName: "good", PolicyGroups: []*storage.PolicyGroup{group}},
	}}))

	s.Error(Validate(&storage.Policy{Name: "name", PolicyVersion: "1", PolicySections: []*storage.PolicySection{
		{SectionName: "good", PolicyGroups: []*storage.PolicyGroup{group}},
	}}))

	s.Error(Validate(&storage.Policy{Name: "name", PolicyVersion: "x.y.z", PolicySections: []*storage.PolicySection{
		{SectionName: "good", PolicyGroups: []*storage.PolicyGroup{group}},
	}}))
}
