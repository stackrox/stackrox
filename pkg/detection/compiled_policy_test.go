package detection

import (
	"context"
	"slices"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func constructPolicy(scopes []*storage.Scope, exclusions []*storage.Exclusion) *storage.Policy {
	return &storage.Policy{
		PolicyVersion:   policyversion.CurrentVersion().String(),
		Name:            "testname",
		Scope:           scopes,
		Exclusions:      exclusions,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		PolicySections:  []*storage.PolicySection{{PolicyGroups: []*storage.PolicyGroup{{FieldName: fieldnames.VolumeName, Values: []*storage.PolicyValue{{Value: "something"}}}}}},
	}
}

func newDeployment(id string) *storage.Deployment {
	dep := fixtures.GetDeployment()
	dep.Id = id
	return dep
}

func TestCompiledPolicyScopesAndExclusions(t *testing.T) {
	stackRoxNSScope := &storage.Scope{Namespace: "stackr.*"}
	defaultNSScope := &storage.Scope{Namespace: "default"}
	appStackRoxScope := &storage.Scope{Label: &storage.Scope_Label{Key: "app", Value: "stackrox"}}

	stackRoxNSDep := newDeployment("STACKROXDEP")

	defaultNSDep := newDeployment("DEFAULTDEP")
	defaultNSDep.Namespace = "default"

	appStackRoxDep := newDeployment("APPSTACKROXDEP")
	appStackRoxDep.Labels["app"] = "stackrox"

	allDeps := []*storage.Deployment{appStackRoxDep, defaultNSDep, stackRoxNSDep}

	for _, testCase := range []struct {
		desc          string
		scopes        []*storage.Scope
		exclusions    []*storage.Exclusion
		shouldApplyTo []*storage.Deployment
	}{
		{
			desc:          "no scopes or excluded scopes",
			shouldApplyTo: []*storage.Deployment{stackRoxNSDep, defaultNSDep, appStackRoxDep},
		},
		{
			desc:          "only stackrox ns",
			scopes:        []*storage.Scope{stackRoxNSScope},
			shouldApplyTo: []*storage.Deployment{stackRoxNSDep, appStackRoxDep},
		},
		{
			desc:          "only stackrox ns, but app=stackrox excluded",
			scopes:        []*storage.Scope{stackRoxNSScope},
			exclusions:    []*storage.Exclusion{{Deployment: &storage.Exclusion_Deployment{Scope: appStackRoxScope}}},
			shouldApplyTo: []*storage.Deployment{stackRoxNSDep},
		},
		{
			desc:          "only default ns",
			scopes:        []*storage.Scope{defaultNSScope},
			shouldApplyTo: []*storage.Deployment{defaultNSDep},
		},
		{
			desc:          "either default ns or app=stackrox",
			scopes:        []*storage.Scope{defaultNSScope, appStackRoxScope},
			shouldApplyTo: []*storage.Deployment{defaultNSDep, appStackRoxDep},
		},
	} {
		c := testCase
		t.Run(c.desc, func(t *testing.T) {
			compiled, err := CompilePolicy(constructPolicy(c.scopes, c.exclusions), nil, nil)
			require.NoError(t, err)
			for _, dep := range c.shouldApplyTo {
				assert.True(t, compiled.AppliesTo(context.Background(), dep), "Failed expectation for %s", dep.GetId())
			}
			for _, dep := range allDeps {
				if slices.Index(c.shouldApplyTo, dep) == -1 {
					assert.False(t, compiled.AppliesTo(context.Background(), dep), "Failed expectation for %s", dep.GetId())
				}
			}
		})
	}
}

// TestProcessAndFileAccessMatchers verifies that when a policy contains both Process and FileAccess fields,
// only the file access matcher is created, not both matchers.
func TestProcessAndFileAccessMatchers(t *testing.T) {
	t.Setenv(features.SensitiveFileActivity.EnvVar(), "true")
	if !features.SensitiveFileActivity.Enabled() {
		t.Fatal("Failed to enable SensitiveFileActivity feature flag")
	}

	type matcherType int
	const (
		noMatcher matcherType = iota
		processMatcher
		fileAccessMatcher
	)

	tests := []struct {
		name                   string
		policySections         []*storage.PolicySection
		lifecycleStages        []storage.LifecycleStage
		eventSource            storage.EventSource
		expectedMatcherType    matcherType
		expectCompilationError bool
	}{
		{
			name: "Process only - should create process matcher",
			policySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.ProcessName,
							Values:    []*storage.PolicyValue{{Value: "bash"}},
						},
					},
				},
			},
			lifecycleStages:     []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
			eventSource:         storage.EventSource_DEPLOYMENT_EVENT,
			expectedMatcherType: processMatcher,
		},
		{
			name: "FileAccess only - should create file access matcher",
			policySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.FilePath,
							Values:    []*storage.PolicyValue{{Value: "/etc/passwd"}},
						},
					},
				},
			},
			lifecycleStages:     []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
			eventSource:         storage.EventSource_DEPLOYMENT_EVENT,
			expectedMatcherType: fileAccessMatcher,
		},
		{
			name: "Process + FileAccess in same section - should create ONLY file access matcher",
			policySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.ProcessName,
							Values:    []*storage.PolicyValue{{Value: "bash"}},
						},
						{
							FieldName: fieldnames.FilePath,
							Values:    []*storage.PolicyValue{{Value: "/etc/passwd"}},
						},
					},
				},
			},
			lifecycleStages:     []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
			eventSource:         storage.EventSource_DEPLOYMENT_EVENT,
			expectedMatcherType: fileAccessMatcher,
		},
		{
			name: "Multiple sections with Process + FileAccess - should create ONLY file access matcher",
			policySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.ProcessName,
							Values:    []*storage.PolicyValue{{Value: "bash"}},
						},
						{
							FieldName: fieldnames.FilePath,
							Values:    []*storage.PolicyValue{{Value: "/etc/passwd"}},
						},
					},
				},
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.ProcessUID,
							Values:    []*storage.PolicyValue{{Value: "0"}},
						},
						{
							FieldName: fieldnames.FilePath,
							Values:    []*storage.PolicyValue{{Value: "/etc/shadow"}},
						},
						{
							FieldName: fieldnames.FileOperation,
							Values:    []*storage.PolicyValue{{Value: "open"}},
						},
					},
				},
			},
			lifecycleStages:     []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
			eventSource:         storage.EventSource_DEPLOYMENT_EVENT,
			expectedMatcherType: fileAccessMatcher,
		},
		{
			name: "FileAccess-only section alongside Process+FileAccess section - should create ONLY file access matcher",
			policySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.FilePath,
							Values:    []*storage.PolicyValue{{Value: "/tmp/*"}},
						},
					},
				},
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.ProcessName,
							Values:    []*storage.PolicyValue{{Value: "vim"}},
						},
						{
							FieldName: fieldnames.FilePath,
							Values:    []*storage.PolicyValue{{Value: "/etc/shadow"}},
						},
					},
				},
			},
			lifecycleStages:     []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
			eventSource:         storage.EventSource_DEPLOYMENT_EVENT,
			expectedMatcherType: fileAccessMatcher,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			policy := &storage.Policy{
				PolicyVersion:   policyversion.CurrentVersion().String(),
				Name:            "test-policy",
				PolicySections:  tc.policySections,
				LifecycleStages: tc.lifecycleStages,
				EventSource:     tc.eventSource,
				Severity:        storage.Severity_HIGH_SEVERITY,
				Categories:      []string{"Test"},
			}

			compiled, err := CompilePolicy(policy, nil, nil)
			if tc.expectCompilationError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			cp := compiled.(*compiledPolicy)

			switch tc.expectedMatcherType {
			case processMatcher:
				assert.NotNil(t, cp.deploymentWithProcessMatcher, "Expected process matcher to be set")
				assert.True(t, cp.hasProcessSection, "Expected hasProcessSection to be true")
				assert.Nil(t, cp.deploymentWithFileAccessMatcher, "Expected file access matcher to be nil")
				assert.False(t, cp.hasFileAccessSection, "Expected hasFileAccessSection to be false")
			case fileAccessMatcher:
				assert.NotNil(t, cp.deploymentWithFileAccessMatcher, "Expected file access matcher to be set")
				assert.True(t, cp.hasFileAccessSection, "Expected hasFileAccessSection to be true")
				assert.Nil(t, cp.deploymentWithProcessMatcher, "Expected process matcher to be nil")
				assert.False(t, cp.hasProcessSection, "Expected hasProcessSection to be false")
			case noMatcher:
				assert.Nil(t, cp.deploymentWithProcessMatcher, "Expected process matcher to be nil")
				assert.False(t, cp.hasProcessSection, "Expected hasProcessSection to be false")
				assert.Nil(t, cp.deploymentWithFileAccessMatcher, "Expected file access matcher to be nil")
				assert.False(t, cp.hasFileAccessSection, "Expected hasFileAccessSection to be false")
			}
		})
	}
}
