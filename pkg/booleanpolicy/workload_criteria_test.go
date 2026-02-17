package booleanpolicy

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// WorkloadCriteriaTestSuite tests workload-related policy criteria.
type WorkloadCriteriaTestSuite struct {
	basePoliciesTestSuite
}

func TestWorkloadCriteria(t *testing.T) {
	t.Setenv(features.CVEFixTimestampCriteria.EnvVar(), "true")
	suite.Run(t, new(WorkloadCriteriaTestSuite))
}

func rbacPermissionMessage(level string) []*storage.Alert_Violation {
	permissionToDescMap := map[string]string{
		"NONE":                  "no specified access",
		"DEFAULT":               "default access",
		"ELEVATED_IN_NAMESPACE": "elevated access in namespace",
		"ELEVATED_CLUSTER_WIDE": "elevated access cluster wide",
		"CLUSTER_ADMIN":         "cluster admin access"}
	return []*storage.Alert_Violation{{Message: fmt.Sprintf("Service account permission level with %s", permissionToDescMap[level])}}
}

func (suite *WorkloadCriteriaTestSuite) TestMapPolicyMatchOne() {
	noAnnotation := &storage.Deployment{
		Id: "noAnnotation",
	}
	suite.addDepAndImages(noAnnotation)

	noValidAnnotation := &storage.Deployment{
		Id: "noValidAnnotation",
		Annotations: map[string]string{
			"email":               "notavalidemail",
			"someotherannotation": "vv@stackrox.com",
		},
	}
	suite.addDepAndImages(noValidAnnotation)

	validAnnotation := &storage.Deployment{
		Id: "validAnnotation",
		Annotations: map[string]string{
			"email": "joseph@rules.gov",
		},
	}
	suite.addDepAndImages(validAnnotation)

	policy := suite.defaultPolicies["Required Annotation: Email"]

	m, err := BuildDeploymentMatcher(policy)
	suite.NoError(err)

	for _, testCase := range []struct {
		dep                *storage.Deployment
		expectedViolations []string
	}{
		{
			noAnnotation,
			[]string{"Required annotation not found (found annotations: <empty>)"},
		},
		{
			noValidAnnotation,
			[]string{"Required annotation not found (found annotations: email=notavalidemail, someotherannotation=vv@stackrox.com)"},
		},
		{
			validAnnotation,
			nil,
		},
	} {
		c := testCase
		suite.Run(c.dep.GetId(), func() {
			matched, err := m.MatchDeployment(nil, enhancedDeployment(c.dep, nil))
			suite.NoError(err)
			var expectedMessages []*storage.Alert_Violation
			for _, v := range c.expectedViolations {
				expectedMessages = append(expectedMessages, &storage.Alert_Violation{Message: v})
			}
			protoassert.SlicesEqual(suite.T(), matched.AlertViolations, expectedMessages)
		})
	}
}

func (suite *WorkloadCriteriaTestSuite) TestK8sRBACField() {
	deployments := make(map[string]*storage.Deployment)
	for permissionLevelStr, permissionLevel := range storage.PermissionLevel_value {
		dep := fixtures.GetDeployment().CloneVT()
		dep.ServiceAccountPermissionLevel = storage.PermissionLevel(permissionLevel)
		deployments[permissionLevelStr] = dep
	}

	for _, testCase := range []struct {
		value           string
		negate          bool
		expectedMatches []string
		// Deployment ids to violations
		expectedViolations map[string][]*storage.Alert_Violation
	}{
		{
			"DEFAULT",
			false,
			[]string{"DEFAULT", "ELEVATED_IN_NAMESPACE", "ELEVATED_CLUSTER_WIDE", "CLUSTER_ADMIN"},
			map[string][]*storage.Alert_Violation{
				"DEFAULT":               rbacPermissionMessage("DEFAULT"),
				"ELEVATED_CLUSTER_WIDE": rbacPermissionMessage("ELEVATED_CLUSTER_WIDE"),
				"ELEVATED_IN_NAMESPACE": rbacPermissionMessage("ELEVATED_IN_NAMESPACE"),
				"CLUSTER_ADMIN":         rbacPermissionMessage("CLUSTER_ADMIN"),
			},
		},
		{
			"ELEVATED_CLUSTER_WIDE",
			false,
			[]string{"ELEVATED_CLUSTER_WIDE", "CLUSTER_ADMIN"},
			map[string][]*storage.Alert_Violation{
				"ELEVATED_CLUSTER_WIDE": rbacPermissionMessage("ELEVATED_CLUSTER_WIDE"),
				"CLUSTER_ADMIN":         rbacPermissionMessage("CLUSTER_ADMIN"),
			},
		},
		{
			"cluster_admin",
			false,
			[]string{"CLUSTER_ADMIN"},
			map[string][]*storage.Alert_Violation{
				"CLUSTER_ADMIN": rbacPermissionMessage("CLUSTER_ADMIN"),
			},
		},
		{
			"ELEVATED_CLUSTER_WIDE",
			true,
			[]string{"NONE", "DEFAULT", "ELEVATED_IN_NAMESPACE"},
			map[string][]*storage.Alert_Violation{
				"ELEVATED_IN_NAMESPACE": rbacPermissionMessage("ELEVATED_IN_NAMESPACE"),
				"NONE":                  rbacPermissionMessage("NONE"),
				"DEFAULT":               rbacPermissionMessage("DEFAULT"),
			},
		},
	} {
		c := testCase
		suite.T().Run(fmt.Sprintf("%+v", c.expectedMatches), func(t *testing.T) {
			matcher, err := BuildDeploymentMatcher(policyWithSingleKeyValue(fieldnames.MinimumRBACPermissions, c.value, c.negate))
			require.NoError(t, err)
			matched := set.NewStringSet()
			for depRef, dep := range deployments {
				violations, err := matcher.MatchDeployment(nil, enhancedDeployment(dep, suite.getImagesForDeployment(dep)))
				require.NoError(t, err)
				if len(violations.AlertViolations) > 0 {
					matched.Add(depRef)
					protoassert.SlicesEqual(t, violations.AlertViolations, c.expectedViolations[depRef])
				} else {
					assert.Empty(t, c.expectedViolations[depRef])
				}
			}
			assert.ElementsMatch(t, matched.AsSlice(), c.expectedMatches, "Got %v, expected: %v", matched.AsSlice(), c.expectedMatches)
		})
	}
}

func (suite *WorkloadCriteriaTestSuite) TestPortExposure() {
	deployments := make(map[string]*storage.Deployment)
	for exposureLevelStr, exposureLevel := range storage.PortConfig_ExposureLevel_value {
		dep := fixtures.GetDeployment().CloneVT()
		dep.Ports = []*storage.PortConfig{{ExposureInfos: []*storage.PortConfig_ExposureInfo{{Level: storage.PortConfig_ExposureLevel(exposureLevel)}}}}
		deployments[exposureLevelStr] = dep
	}

	assertMessageMatches := func(t *testing.T, depRef string, violations []*storage.Alert_Violation) {
		depRefToExpectedMsg := map[string]string{
			"EXTERNAL": "exposed with load balancer",
			"NODE":     "exposed on node port",
			"INTERNAL": "using internal cluster IP",
			"HOST":     "exposed on host port",
			"ROUTE":    "exposed with a route",
		}
		require.Len(t, violations, 1)
		assert.Equal(t, fmt.Sprintf("Deployment port(s) %s", depRefToExpectedMsg[depRef]), violations[0].GetMessage())
	}

	for _, testCase := range []struct {
		values          []string
		negate          bool
		expectedMatches []string
	}{
		{
			[]string{"external"},
			false,
			[]string{"EXTERNAL"},
		},
		{
			[]string{"external", "NODE"},
			false,
			[]string{"EXTERNAL", "NODE"},
		},
		{
			[]string{"external", "NODE"},
			true,
			[]string{"INTERNAL", "HOST", "ROUTE"},
		},
	} {
		c := testCase
		suite.T().Run(fmt.Sprintf("%+v", c), func(t *testing.T) {
			matcher, err := BuildDeploymentMatcher(policyWithSingleFieldAndValues(fieldnames.PortExposure, c.values, c.negate, storage.BooleanOperator_OR))
			require.NoError(t, err)
			matched := set.NewStringSet()
			for depRef, dep := range deployments {
				violations, err := matcher.MatchDeployment(nil, enhancedDeployment(dep, suite.getImagesForDeployment(dep)))
				require.NoError(t, err)
				if len(violations.AlertViolations) > 0 {
					assertMessageMatches(t, depRef, violations.AlertViolations)
					matched.Add(depRef)
				}
			}
			assert.ElementsMatch(t, matched.AsSlice(), c.expectedMatches, "Got %v, expected: %v", matched.AsSlice(), c.expectedMatches)
		})
	}
}

func (suite *WorkloadCriteriaTestSuite) TestContainerName() {
	var deps []*storage.Deployment
	for _, containerName := range []string{
		"container_staging",
		"container_prod0",
		"container_prod1",
		"container_internal",
		"external_container",
	} {
		dep := fixtures.GetDeployment().CloneVT()
		dep.Containers = []*storage.Container{
			{
				Name: containerName,
			},
		}
		deps = append(deps, dep)
	}

	for _, testCase := range []struct {
		value           string
		expectedMatches []string
		negate          bool
	}{
		{
			value:           "container_[a-z0-9]*",
			expectedMatches: []string{"container_staging", "container_prod0", "container_prod1", "container_internal"},
			negate:          false,
		},
		{
			value:           "container_prod[a-z0-9]*",
			expectedMatches: []string{"container_prod0", "container_prod1"},
			negate:          false,
		},
		{
			value:           ".*external.*",
			expectedMatches: []string{"external_container"},
			negate:          false,
		},
		{
			value:           "doesnotexist",
			expectedMatches: nil,
			negate:          false,
		},
		{
			value:           ".*internal.*",
			expectedMatches: []string{"container_staging", "container_prod0", "container_prod1", "external_container"},
			negate:          true,
		},
	} {
		c := testCase

		suite.T().Run(fmt.Sprintf("DeploymentMatcher %+v", c), func(t *testing.T) {
			depMatcher, err := BuildDeploymentMatcher(policyWithSingleKeyValue(fieldnames.ContainerName, c.value, c.negate))
			require.NoError(t, err)
			containerNameMatched := set.NewStringSet()
			for _, dep := range deps {
				violations, err := depMatcher.MatchDeployment(nil, enhancedDeployment(dep, suite.getImagesForDeployment(dep)))
				require.NoError(t, err)
				// No match in case we are testing for doesnotexist
				if len(violations.AlertViolations) > 0 {
					containerNameMatched.Add(dep.GetContainers()[0].GetName())
					require.Len(t, violations.AlertViolations, 1)
					assert.Equal(t, fmt.Sprintf("Container has name '%s'", dep.GetContainers()[0].GetName()), violations.AlertViolations[0].GetMessage())
				}
			}
			assert.ElementsMatch(t, containerNameMatched.AsSlice(), c.expectedMatches, "Got %v for policy %v; expected: %v", containerNameMatched.AsSlice(), c.value, c.expectedMatches)
		})
	}
}

func (suite *WorkloadCriteriaTestSuite) TestAllowPrivilegeEscalationPolicyCriteria() {
	const containerAllowPrivEsc = "Container with Privilege Escalation allowed"
	const containerNotAllowPrivEsc = "Container with Privilege Escalation not allowed"

	var deps []*storage.Deployment
	for _, d := range []struct {
		ContainerName            string
		AllowPrivilegeEscalation bool
	}{
		{
			ContainerName:            containerAllowPrivEsc,
			AllowPrivilegeEscalation: true,
		},
		{
			ContainerName:            containerNotAllowPrivEsc,
			AllowPrivilegeEscalation: false,
		},
	} {
		dep := fixtures.GetDeployment().CloneVT()
		dep.Containers[0].Name = d.ContainerName
		if d.AllowPrivilegeEscalation {
			dep.Containers[0].SecurityContext.AllowPrivilegeEscalation = d.AllowPrivilegeEscalation
		}
		deps = append(deps, dep)
	}

	for _, testCase := range []struct {
		CaseName        string
		value           string
		expectedMatches []string
	}{
		{
			CaseName:        "Policy for containers with privilege escalation allowed",
			value:           "true",
			expectedMatches: []string{containerAllowPrivEsc},
		},
		{
			CaseName:        "Policy for containers with privilege escalation not allowed",
			value:           "false",
			expectedMatches: []string{containerNotAllowPrivEsc},
		},
	} {
		c := testCase

		suite.T().Run(c.CaseName, func(t *testing.T) {
			depMatcher, err := BuildDeploymentMatcher(policyWithSingleKeyValue(fieldnames.AllowPrivilegeEscalation, c.value, false))
			require.NoError(t, err)
			containerNameMatched := set.NewStringSet()
			for _, dep := range deps {
				violations, err := depMatcher.MatchDeployment(nil, enhancedDeployment(dep, suite.getImagesForDeployment(dep)))
				require.NoError(t, err)
				if len(violations.AlertViolations) > 0 {
					containerNameMatched.Add(dep.GetContainers()[0].GetName())
					require.Len(t, violations.AlertViolations, 1)
					if c.value == "true" {
						assert.Equal(t, fmt.Sprintf("Container '%s' allows privilege escalation", dep.GetContainers()[0].GetName()), violations.AlertViolations[0].GetMessage())
					} else {
						assert.Equal(t, fmt.Sprintf("Container '%s' does not allow privilege escalation", dep.GetContainers()[0].GetName()), violations.AlertViolations[0].GetMessage())
					}
				}
			}
			assert.ElementsMatch(t, containerNameMatched.AsSlice(), c.expectedMatches, "Matched containers %v for policy %v; expected: %v", containerNameMatched.AsSlice(), c.value, c.expectedMatches)
		})
	}
}

func (suite *WorkloadCriteriaTestSuite) TestAutomountServiceAccountToken() {
	deployments := make(map[string]*storage.Deployment)
	for _, d := range []struct {
		DeploymentName                string
		ServiceAccountName            string
		AutomountServiceAccountTokens bool
	}{
		{
			DeploymentName:                "DefaultSAAutomountedTokens",
			ServiceAccountName:            "default",
			AutomountServiceAccountTokens: true,
		},
		{
			DeploymentName:     "DefaultSANotAutomountedTokens",
			ServiceAccountName: "default",
		},
		{
			DeploymentName:                "CustomSAAutomountedTokens",
			ServiceAccountName:            "custom",
			AutomountServiceAccountTokens: true,
		},
		{
			DeploymentName:     "CustomSANotAutomountedTokens",
			ServiceAccountName: "custom",
		},
	} {
		dep := fixtures.GetDeployment().CloneVT()
		dep.Name = d.DeploymentName
		dep.ServiceAccount = d.ServiceAccountName
		dep.AutomountServiceAccountToken = d.AutomountServiceAccountTokens
		deployments[dep.GetName()] = dep
	}

	automountServiceAccountTokenPolicyGroup := &storage.PolicyGroup{
		FieldName: fieldnames.AutomountServiceAccountToken,
		Values:    []*storage.PolicyValue{{Value: "true"}},
	}
	defaultServiceAccountPolicyGroup := &storage.PolicyGroup{
		FieldName: fieldnames.ServiceAccount,
		Values:    []*storage.PolicyValue{{Value: "default"}},
	}

	allAutomountServiceAccountTokenPolicy := policyWithGroups(storage.EventSource_NOT_APPLICABLE, automountServiceAccountTokenPolicyGroup)
	defaultAutomountServiceAccountTokenPolicy := policyWithGroups(storage.EventSource_NOT_APPLICABLE, automountServiceAccountTokenPolicyGroup, defaultServiceAccountPolicyGroup)

	automountAlert := &storage.Alert_Violation{Message: "Deployment mounts the service account tokens."}
	defaultServiceAccountAlert := &storage.Alert_Violation{Message: "Service Account is set to 'default'"}

	for _, c := range []struct {
		CaseName       string
		Policy         *storage.Policy
		DeploymentName string
		ExpectedAlerts []*storage.Alert_Violation
	}{
		{
			CaseName:       "Automounted default service account tokens should alert on bare automount policy",
			Policy:         allAutomountServiceAccountTokenPolicy,
			DeploymentName: "DefaultSAAutomountedTokens",
			ExpectedAlerts: []*storage.Alert_Violation{automountAlert},
		},
		{
			CaseName:       "Automounted default service account tokens should alert on default only automount policy",
			Policy:         defaultAutomountServiceAccountTokenPolicy,
			DeploymentName: "DefaultSAAutomountedTokens",
			ExpectedAlerts: []*storage.Alert_Violation{automountAlert, defaultServiceAccountAlert},
		},
		{
			CaseName:       "Automounted custom service account tokens should alert on bare automount policy",
			Policy:         allAutomountServiceAccountTokenPolicy,
			DeploymentName: "CustomSAAutomountedTokens",
			ExpectedAlerts: []*storage.Alert_Violation{automountAlert},
		},
		{
			CaseName:       "Not automounted default service account should not alert on bare automount policy",
			Policy:         allAutomountServiceAccountTokenPolicy,
			DeploymentName: "DefaultSANotAutomountedTokens",
		},
		{
			CaseName:       "Not automounted custom service account should not alert on bare automount policy",
			Policy:         allAutomountServiceAccountTokenPolicy,
			DeploymentName: "CustomSANotAutomountedTokens",
		},
	} {
		suite.T().Run(c.CaseName, func(t *testing.T) {
			dep := deployments[c.DeploymentName]
			matcher, err := BuildDeploymentMatcher(c.Policy)
			suite.NoError(err, "deployment matcher creation must succeed")
			violations, err := matcher.MatchDeployment(nil, enhancedDeployment(dep, suite.getImagesForDeployment(dep)))
			suite.NoError(err, "deployment matcher run must succeed")
			suite.Empty(violations.ProcessViolation)
			protoassert.SlicesEqual(suite.T(), c.ExpectedAlerts, violations.AlertViolations)
		})
	}
}

func (suite *WorkloadCriteriaTestSuite) TestRuntimeClass() {
	var deps []*storage.Deployment
	for _, runtimeClass := range []string{
		"",
		"blah",
	} {
		dep := fixtures.GetDeployment().CloneVT()
		dep.RuntimeClass = runtimeClass
		deps = append(deps, dep)
	}

	for _, testCase := range []struct {
		value           string
		negate          bool
		expectedMatches []string
	}{
		{
			value:           ".*",
			negate:          false,
			expectedMatches: []string{"", "blah"},
		},
		{
			value:           ".+",
			negate:          false,
			expectedMatches: []string{"blah"},
		},
		{
			value:           ".+",
			negate:          true,
			expectedMatches: []string{""},
		},
		{
			value:           "blah",
			negate:          true,
			expectedMatches: []string{""},
		},
	} {
		c := testCase

		suite.T().Run(fmt.Sprintf("%+v", c), func(t *testing.T) {
			depMatcher, err := BuildDeploymentMatcher(policyWithSingleKeyValue(fieldnames.RuntimeClass, c.value, c.negate))
			require.NoError(t, err)
			matchedRuntimeClasses := set.NewStringSet()
			for _, dep := range deps {
				violations, err := depMatcher.MatchDeployment(nil, enhancedDeployment(dep, suite.getImagesForDeployment(dep)))
				require.NoError(t, err)
				if len(violations.AlertViolations) > 0 {
					matchedRuntimeClasses.Add(dep.GetRuntimeClass())
					require.Len(t, violations.AlertViolations, 1)
					assert.Equal(t, fmt.Sprintf("Runtime Class is set to '%s'", dep.GetRuntimeClass()), violations.AlertViolations[0].GetMessage())
				}
			}
			assert.ElementsMatch(t, matchedRuntimeClasses.AsSlice(), c.expectedMatches, "Got %v for policy %v; expected: %v", matchedRuntimeClasses.AsSlice(), c.value, c.expectedMatches)
		})
	}
}

func (suite *WorkloadCriteriaTestSuite) TestNamespace() {
	var deps []*storage.Deployment
	for _, namespace := range []string{
		"dep_staging",
		"dep_prod0",
		"dep_prod1",
		"dep_internal",
		"external_dep",
	} {
		dep := fixtures.GetDeployment().CloneVT()
		dep.Namespace = namespace
		deps = append(deps, dep)
	}

	for _, testCase := range []struct {
		value           string
		expectedMatches []string
		negate          bool
	}{
		{
			value:           "dep_[a-z0-9]*",
			expectedMatches: []string{"dep_staging", "dep_prod0", "dep_prod1", "dep_internal"},
			negate:          false,
		},
		{
			value:           "dep_prod[a-z0-9]*",
			expectedMatches: []string{"dep_prod0", "dep_prod1"},
			negate:          false,
		},
		{
			value:           ".*external.*",
			expectedMatches: []string{"external_dep"},
			negate:          false,
		},
		{
			value:           "doesnotexist",
			expectedMatches: nil,
			negate:          false,
		},
		{
			value:           ".*internal.*",
			expectedMatches: []string{"dep_staging", "dep_prod0", "dep_prod1", "external_dep"},
			negate:          true,
		},
	} {
		c := testCase

		suite.T().Run(fmt.Sprintf("DeploymentMatcher %+v", c), func(t *testing.T) {
			depMatcher, err := BuildDeploymentMatcher(policyWithSingleKeyValue(fieldnames.Namespace, c.value, c.negate))
			require.NoError(t, err)
			namespacesMatched := set.NewStringSet()
			for _, dep := range deps {
				violations, err := depMatcher.MatchDeployment(nil, enhancedDeployment(dep, suite.getImagesForDeployment(dep)))
				require.NoError(t, err)
				// No match in case we are testing for doesnotexist
				if len(violations.AlertViolations) > 0 {
					namespacesMatched.Add(dep.GetNamespace())
					require.Len(t, violations.AlertViolations, 1)
					assert.Equal(t, fmt.Sprintf("Namespace has name '%s'", dep.GetNamespace()), violations.AlertViolations[0].GetMessage())
				}
			}
			assert.ElementsMatch(t, namespacesMatched.AsSlice(), c.expectedMatches, "Got %v for policy %v; expected: %v", namespacesMatched.AsSlice(), c.value, c.expectedMatches)
		})
	}
}

func (suite *WorkloadCriteriaTestSuite) TestDropCaps() {
	testCaps := []string{"SYS_MODULE", "SYS_NICE", "SYS_PTRACE", "ALL"}

	deployments := make(map[string]*storage.Deployment)
	for _, idxs := range [][]int{{}, {0}, {1}, {2}, {0, 1}, {1, 2}, {0, 1, 2}, {3}} {
		dep := fixtures.GetDeployment().CloneVT()
		dep.Containers[0].SecurityContext.DropCapabilities = make([]string, 0, len(idxs))
		for _, idx := range idxs {
			dep.Containers[0].SecurityContext.DropCapabilities = append(dep.Containers[0].SecurityContext.DropCapabilities, testCaps[idx])
		}
		deployments[strings.ReplaceAll(strings.Join(dep.GetContainers()[0].GetSecurityContext().GetDropCapabilities(), ","), "SYS_", "")] = dep
	}

	assertMessageMatches := func(t *testing.T, depRef string, violations []*storage.Alert_Violation) {
		depRefToExpectedMsg := map[string]string{
			"":                   "no capabilities",
			"ALL":                "all capabilities",
			"MODULE":             "SYS_MODULE",
			"NICE":               "SYS_NICE",
			"PTRACE":             "SYS_PTRACE",
			"MODULE,NICE":        "SYS_MODULE and SYS_NICE",
			"NICE,PTRACE":        "SYS_NICE and SYS_PTRACE",
			"MODULE,NICE,PTRACE": "SYS_MODULE, SYS_NICE, and SYS_PTRACE",
		}
		require.Len(t, violations, 1)
		assert.Equal(t, fmt.Sprintf("Container 'nginx110container' does not drop expected capabilities (drops %s)", depRefToExpectedMsg[depRef]), violations[0].GetMessage())
	}

	for _, testCase := range []struct {
		values          []string
		op              storage.BooleanOperator
		expectedMatches []string
	}{
		{
			// Nothing drops this capability
			[]string{"SYSLOG"},
			storage.BooleanOperator_OR,
			[]string{"", "MODULE", "NICE", "PTRACE", "MODULE,NICE", "NICE,PTRACE", "MODULE,NICE,PTRACE"},
		},
		{
			[]string{"SYS_NICE"},
			storage.BooleanOperator_OR,
			[]string{"", "MODULE", "PTRACE"},
		},
		{
			[]string{"SYS_NICE", "SYS_PTRACE"},
			storage.BooleanOperator_OR,
			[]string{"", "MODULE"},
		},
		{
			[]string{"SYS_NICE", "SYS_PTRACE"},
			storage.BooleanOperator_AND,
			[]string{"", "MODULE", "PTRACE", "NICE", "MODULE,NICE"},
		},
		{
			[]string{"ALL"},
			storage.BooleanOperator_AND,
			[]string{"", "MODULE", "NICE", "PTRACE", "MODULE,NICE", "NICE,PTRACE", "MODULE,NICE,PTRACE"},
		},
	} {
		c := testCase
		suite.T().Run(fmt.Sprintf("%+v", c), func(t *testing.T) {
			matcher, err := BuildDeploymentMatcher(policyWithSingleFieldAndValues(fieldnames.DropCaps, c.values, false, c.op))
			require.NoError(t, err)
			matched := set.NewStringSet()
			for depRef, dep := range deployments {
				violations, err := matcher.MatchDeployment(nil, enhancedDeployment(dep, suite.getImagesForDeployment(dep)))
				require.NoError(t, err)
				if len(violations.AlertViolations) > 0 {
					matched.Add(depRef)
					assertMessageMatches(t, depRef, violations.AlertViolations)
				}
			}
			assert.ElementsMatch(t, matched.AsSlice(), c.expectedMatches, "Got %v, expected: %v", matched.AsSlice(), c.expectedMatches)
		})
	}
}

func (suite *WorkloadCriteriaTestSuite) TestAddCaps() {
	testCaps := []string{"SYS_MODULE", "SYS_NICE", "SYS_PTRACE"}

	deployments := make(map[string]*storage.Deployment)
	for _, idxs := range [][]int{{}, {0}, {1}, {2}, {0, 1}, {1, 2}, {0, 1, 2}} {
		dep := fixtures.GetDeployment().CloneVT()
		dep.Containers[0].SecurityContext.AddCapabilities = make([]string, 0, len(idxs))
		for _, idx := range idxs {
			dep.Containers[0].SecurityContext.AddCapabilities = append(dep.Containers[0].SecurityContext.AddCapabilities, testCaps[idx])
		}
		deployments[strings.ReplaceAll(strings.Join(dep.GetContainers()[0].GetSecurityContext().GetAddCapabilities(), ","), "SYS_", "")] = dep
	}

	for _, testCase := range []struct {
		values          []string
		op              storage.BooleanOperator
		expectedMatches []string
	}{
		{
			// Nothing adds this capability
			[]string{"SYSLOG"},
			storage.BooleanOperator_OR,
			[]string{},
		},
		{
			[]string{"SYS_NICE"},
			storage.BooleanOperator_OR,
			[]string{"NICE", "MODULE,NICE", "NICE,PTRACE", "MODULE,NICE,PTRACE"},
		},
		{
			[]string{"SYS_NICE", "SYS_PTRACE"},
			storage.BooleanOperator_OR,
			[]string{"NICE", "PTRACE", "MODULE,NICE", "NICE,PTRACE", "MODULE,NICE,PTRACE"},
		},
		{
			[]string{"SYS_NICE", "SYS_PTRACE"},
			storage.BooleanOperator_AND,
			[]string{"NICE,PTRACE", "MODULE,NICE,PTRACE"},
		},
	} {
		c := testCase
		suite.T().Run(fmt.Sprintf("%+v", c), func(t *testing.T) {
			matcher, err := BuildDeploymentMatcher(policyWithSingleFieldAndValues(fieldnames.AddCaps, c.values, false, c.op))
			require.NoError(t, err)
			matched := set.NewStringSet()
			for depRef, dep := range deployments {
				violations, err := matcher.MatchDeployment(nil, enhancedDeployment(dep, suite.getImagesForDeployment(dep)))
				require.NoError(t, err)
				if len(violations.AlertViolations) > 0 {
					matched.Add(depRef)
					require.Len(t, violations.AlertViolations, 1)
				}
			}
			assert.ElementsMatch(t, matched.AsSlice(), c.expectedMatches, "Got %v, expected: %v", matched.AsSlice(), c.expectedMatches)
		})
	}
}

func (suite *WorkloadCriteriaTestSuite) TestReplicasPolicyCriteria() {
	for _, testCase := range []struct {
		caseName    string
		replicas    int64
		policyValue string
		negate      bool
		alerts      []*storage.Alert_Violation
	}{
		{
			caseName:    "Should raise when replicas==5.",
			replicas:    5,
			policyValue: "5",
			negate:      false,
			alerts:      []*storage.Alert_Violation{{Message: "Replicas is set to '5'"}},
		},
		{
			caseName:    "Should not raise unless replicas==3.",
			replicas:    5,
			policyValue: "3",
			negate:      false,
			alerts:      nil,
		},
		{
			caseName:    "Should raise unless replicas==3.",
			replicas:    5,
			policyValue: "3",
			negate:      true,
			alerts:      []*storage.Alert_Violation{{Message: "Replicas is set to '5'"}},
		},
		{
			caseName:    "Should raise when replicas>=5.",
			replicas:    5,
			policyValue: ">=5",
			negate:      false,
			alerts:      []*storage.Alert_Violation{{Message: "Replicas is set to '5'"}},
		},
		{
			caseName:    "Should raise when replicas<=5.",
			replicas:    5,
			policyValue: "<=5",
			negate:      false,
			alerts:      []*storage.Alert_Violation{{Message: "Replicas is set to '5'"}},
		},
		{
			caseName:    "Should raise when replicas<5.",
			replicas:    1,
			policyValue: "<5",
			negate:      false,
			alerts:      []*storage.Alert_Violation{{Message: "Replicas is set to '1'"}},
		},
		{
			caseName:    "Should raise when replicas>5.",
			replicas:    10,
			policyValue: ">5",
			negate:      false,
			alerts:      []*storage.Alert_Violation{{Message: "Replicas is set to '10'"}},
		},
	} {
		suite.Run(testCase.caseName, func() {
			deployment := fixtures.GetDeployment().CloneVT()
			deployment.Replicas = testCase.replicas
			policy := policyWithSingleKeyValue(fieldnames.Replicas, testCase.policyValue, testCase.negate)

			matcher, err := BuildDeploymentMatcher(policy)
			suite.NoError(err, "deployment matcher creation must succeed")
			violations, err := matcher.MatchDeployment(nil, enhancedDeployment(deployment, suite.getImagesForDeployment(deployment)))
			suite.NoError(err, "deployment matcher run must succeed")

			suite.Empty(violations.ProcessViolation)
			protoassert.SlicesEqual(suite.T(), violations.AlertViolations, testCase.alerts)
		})
	}
}

func (suite *WorkloadCriteriaTestSuite) TestLivenessProbePolicyCriteria() {
	for _, testCase := range []struct {
		caseName    string
		containers  []*storage.Container
		policyValue string
		alerts      []*storage.Alert_Violation
	}{
		{
			caseName: "Should raise alert since liveness probe is defined.",
			containers: []*storage.Container{
				{Name: "container", LivenessProbe: &storage.LivenessProbe{Defined: true}},
			},
			policyValue: "true",
			alerts: []*storage.Alert_Violation{
				{Message: "Liveness probe is defined for container 'container'"},
			},
		},
		{
			caseName: "Should not raise alert since liveness probe is defined.",
			containers: []*storage.Container{
				{Name: "container", LivenessProbe: &storage.LivenessProbe{Defined: true}},
			},
			policyValue: "false",
			alerts:      nil,
		},
		{
			caseName: "Should not raise alert since liveness probe is not defined.",
			containers: []*storage.Container{
				{Name: "container", LivenessProbe: &storage.LivenessProbe{Defined: false}},
			},
			policyValue: "true",
			alerts:      nil,
		},
		{
			caseName: "Should raise alert since liveness probe is not defined.",
			containers: []*storage.Container{
				{Name: "container", LivenessProbe: &storage.LivenessProbe{Defined: false}},
			},
			policyValue: "false",
			alerts: []*storage.Alert_Violation{
				{Message: "Liveness probe is not defined for container 'container'"},
			},
		},
		{
			caseName: "Should raise alert for both containers.",
			containers: []*storage.Container{
				{Name: "container-1", LivenessProbe: &storage.LivenessProbe{Defined: false}},
				{Name: "container-2", LivenessProbe: &storage.LivenessProbe{Defined: false}},
			},
			policyValue: "false",
			alerts: []*storage.Alert_Violation{
				{Message: "Liveness probe is not defined for container 'container-1'"},
				{Message: "Liveness probe is not defined for container 'container-2'"},
			},
		},
		{
			caseName: "Should raise alert only for container-2.",
			containers: []*storage.Container{
				{Name: "container-1", LivenessProbe: &storage.LivenessProbe{Defined: true}},
				{Name: "container-2", LivenessProbe: &storage.LivenessProbe{Defined: false}},
			},
			policyValue: "false",
			alerts: []*storage.Alert_Violation{
				{Message: "Liveness probe is not defined for container 'container-2'"},
			},
		},
	} {
		suite.Run(testCase.caseName, func() {
			deployment := fixtures.GetDeployment().CloneVT()
			deployment.Containers = testCase.containers
			policy := policyWithSingleKeyValue(fieldnames.LivenessProbeDefined, testCase.policyValue, false)

			matcher, err := BuildDeploymentMatcher(policy)
			suite.NoError(err, "deployment matcher creation must succeed")
			violations, err := matcher.MatchDeployment(nil, enhancedDeployment(deployment, suite.getImagesForDeployment(deployment)))
			suite.NoError(err, "deployment matcher run must succeed")

			suite.Empty(violations.ProcessViolation)
			protoassert.SlicesEqual(suite.T(), violations.AlertViolations, testCase.alerts)
		})
	}
}

func (suite *WorkloadCriteriaTestSuite) TestReadinessProbePolicyCriteria() {
	for _, testCase := range []struct {
		caseName    string
		containers  []*storage.Container
		policyValue string
		alerts      []*storage.Alert_Violation
	}{
		{
			caseName: "Should raise alert since readiness probe is defined.",
			containers: []*storage.Container{
				{Name: "container", ReadinessProbe: &storage.ReadinessProbe{Defined: true}},
			},
			policyValue: "true",
			alerts: []*storage.Alert_Violation{
				{Message: "Readiness probe is defined for container 'container'"},
			},
		},
		{
			caseName: "Should not raise alert since readiness probe is defined.",
			containers: []*storage.Container{
				{Name: "container", ReadinessProbe: &storage.ReadinessProbe{Defined: true}},
			},
			policyValue: "false",
			alerts:      nil,
		},
		{
			caseName: "Should not raise alert since readiness probe is not defined.",
			containers: []*storage.Container{
				{Name: "container", ReadinessProbe: &storage.ReadinessProbe{Defined: false}},
			},
			policyValue: "true",
			alerts:      nil,
		},
		{
			caseName: "Should raise alert since readiness probe is not defined.",
			containers: []*storage.Container{
				{Name: "container", ReadinessProbe: &storage.ReadinessProbe{Defined: false}},
			},
			policyValue: "false",
			alerts: []*storage.Alert_Violation{
				{Message: "Readiness probe is not defined for container 'container'"},
			},
		},
		{
			caseName: "Should raise alert for both containers.",
			containers: []*storage.Container{
				{Name: "container-1", ReadinessProbe: &storage.ReadinessProbe{Defined: false}},
				{Name: "container-2", ReadinessProbe: &storage.ReadinessProbe{Defined: false}},
			},
			policyValue: "false",
			alerts: []*storage.Alert_Violation{
				{Message: "Readiness probe is not defined for container 'container-1'"},
				{Message: "Readiness probe is not defined for container 'container-2'"},
			},
		},
		{
			caseName: "Should raise alert only for container-2.",
			containers: []*storage.Container{
				{Name: "container-1", ReadinessProbe: &storage.ReadinessProbe{Defined: true}},
				{Name: "container-2", ReadinessProbe: &storage.ReadinessProbe{Defined: false}},
			},
			policyValue: "false",
			alerts: []*storage.Alert_Violation{
				{Message: "Readiness probe is not defined for container 'container-2'"},
			},
		},
	} {
		suite.Run(testCase.caseName, func() {
			deployment := fixtures.GetDeployment().CloneVT()
			deployment.Containers = testCase.containers
			policy := policyWithSingleKeyValue(fieldnames.ReadinessProbeDefined, testCase.policyValue, false)

			matcher, err := BuildDeploymentMatcher(policy)
			suite.NoError(err, "deployment matcher creation must succeed")
			violations, err := matcher.MatchDeployment(nil, enhancedDeployment(deployment, suite.getImagesForDeployment(deployment)))
			suite.NoError(err, "deployment matcher run must succeed")

			suite.Empty(violations.ProcessViolation)
			protoassert.SlicesEqual(suite.T(), violations.AlertViolations, testCase.alerts)
		})
	}
}
