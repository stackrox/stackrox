package detection

import (
	"context"
	"slices"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/testutils"
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
			exclusions:    []*storage.Exclusion{{Matcher: &storage.Exclusion_Deployment_{Deployment: &storage.Exclusion_Deployment{Scope: appStackRoxScope}}}},
			shouldApplyTo: []*storage.Deployment{stackRoxNSDep},
		},
		{
			desc:          "image-only exclusion does not exclude any deployment",
			exclusions:    []*storage.Exclusion{{Image: &storage.Exclusion_Image{Name: "docker.io/library/unrelated"}}},
			shouldApplyTo: []*storage.Deployment{stackRoxNSDep, defaultNSDep, appStackRoxDep},
		},
		{
			desc:          "only stackrox ns, with an image-only exclusion",
			scopes:        []*storage.Scope{stackRoxNSScope},
			exclusions:    []*storage.Exclusion{{Image: &storage.Exclusion_Image{Name: "docker.io/library/unrelated"}}},
			shouldApplyTo: []*storage.Deployment{stackRoxNSDep, appStackRoxDep},
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

func TestCompiledPolicyWorkloadKindExclusion(t *testing.T) {
	testutils.MustUpdateFeature(t, features.PolicyWorkloadKindExclusion, true)

	cronJob := newDeployment("CRONJOB")
	cronJob.Name = "batch-cron"
	cronJob.Type = kubernetes.CronJob
	regular := newDeployment("DEPLOYMENT")
	regular.Name = "web"
	regular.Type = kubernetes.Deployment

	policy := constructPolicy(nil, []*storage.Exclusion{
		{
			Matcher: &storage.Exclusion_ExcludeByKind_{
				ExcludeByKind: &storage.Exclusion_ExcludeByKind{
					Kinds: []storage.Exclusion_WorkloadKind{storage.Exclusion_CRON_JOB},
				},
			},
		},
		{
			Matcher: &storage.Exclusion_Deployment_{
				Deployment: &storage.Exclusion_Deployment{Name: regular.GetName()},
			},
		},
	})
	compiled, err := CompilePolicy(policy, nil, nil)
	require.NoError(t, err)
	assert.False(t, compiled.AppliesTo(context.Background(), cronJob))
	assert.False(t, compiled.AppliesTo(context.Background(), regular))

	other := newDeployment("OTHER")
	other.Name = "other"
	other.Type = kubernetes.DaemonSet
	assert.True(t, compiled.AppliesTo(context.Background(), other))
}

func TestCompiledPolicyWorkloadKindExclusionFlagOff(t *testing.T) {
	testutils.MustUpdateFeature(t, features.PolicyWorkloadKindExclusion, false)

	cronJob := newDeployment("CRONJOB")
	cronJob.Type = kubernetes.CronJob
	policy := constructPolicy(nil, []*storage.Exclusion{
		{
			Matcher: &storage.Exclusion_ExcludeByKind_{
				ExcludeByKind: &storage.Exclusion_ExcludeByKind{
					Kinds: []storage.Exclusion_WorkloadKind{storage.Exclusion_CRON_JOB},
				},
			},
		},
	})
	compiled, err := CompilePolicy(policy, nil, nil)
	require.NoError(t, err)
	assert.True(t, compiled.AppliesTo(context.Background(), cronJob))
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

func initAndRegularDeployment() (*storage.Deployment, []*storage.Image) {
	dep := &storage.Deployment{
		Id:        "dep",
		Name:      "dep",
		Namespace: "ns",
		Containers: []*storage.Container{
			{
				Name:            "init-setup",
				Type:            storage.ContainerType_INIT,
				SecurityContext: &storage.SecurityContext{Privileged: true},
				Image:           &storage.ContainerImage{Id: "init-img", Name: &storage.ImageName{FullName: "busybox:latest"}},
			},
			{
				Name:            "app",
				Type:            storage.ContainerType_REGULAR,
				SecurityContext: &storage.SecurityContext{},
				Image:           &storage.ContainerImage{Id: "app-img", Name: &storage.ImageName{FullName: "nginx:1.25"}},
			},
		},
	}
	images := []*storage.Image{
		{Id: "init-img", Name: &storage.ImageName{FullName: "busybox:latest"}},
		{Id: "app-img", Name: &storage.ImageName{FullName: "nginx:1.25"}},
	}
	return dep, images
}

func TestEvaluationFilterSkipsContainers(t *testing.T) {
	t.Setenv(features.EvaluationFilter.EnvVar(), "true")

	dep, images := initAndRegularDeployment()
	ed := booleanpolicy.EnhancedDeployment{Deployment: dep, Images: images}

	skipInit := &storage.EvaluationFilter{
		SkipContainerTypes: []storage.ContainerType{storage.ContainerType_INIT},
	}
	privilegedPolicy := &storage.Policy{
		PolicyVersion:   policyversion.CurrentVersion().String(),
		Name:            "privileged",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		PolicySections: []*storage.PolicySection{{
			PolicyGroups: []*storage.PolicyGroup{{
				FieldName: fieldnames.PrivilegedContainer,
				Values:    []*storage.PolicyValue{{Value: "true"}},
			}},
		}},
		EvaluationFilter: skipInit,
	}

	compiled, err := CompilePolicy(privilegedPolicy, nil, nil)
	require.NoError(t, err)

	violations, err := compiled.MatchAgainstDeployment(nil, ed)
	require.NoError(t, err)
	assert.Empty(t, violations.AlertViolations, "privileged init container is skipped")

	dep.Containers[0].SecurityContext.Privileged = false
	dep.Containers[1].SecurityContext.Privileged = true
	violations, err = compiled.MatchAgainstDeployment(nil, ed)
	require.NoError(t, err)
	require.Len(t, violations.AlertViolations, 1)
	assert.Contains(t, violations.AlertViolations[0].GetMessage(), "app")

	skipRegular := privilegedPolicy.CloneVT()
	skipRegular.EvaluationFilter.SkipContainerTypes = []storage.ContainerType{storage.ContainerType_REGULAR}
	compiled, err = CompilePolicy(skipRegular, nil, nil)
	require.NoError(t, err)
	violations, err = compiled.MatchAgainstDeployment(nil, ed)
	require.NoError(t, err)
	assert.Empty(t, violations.AlertViolations, "privileged regular container is skipped")

	dep.Containers[0].SecurityContext.Privileged = true
	dep.Containers[1].SecurityContext.Privileged = false
	violations, err = compiled.MatchAgainstDeployment(nil, ed)
	require.NoError(t, err)
	require.Len(t, violations.AlertViolations, 1)
	assert.Contains(t, violations.AlertViolations[0].GetMessage(), "init-setup")

	unfiltered := privilegedPolicy.CloneVT()
	unfiltered.EvaluationFilter = nil
	compiled, err = CompilePolicy(unfiltered, nil, nil)
	require.NoError(t, err)
	violations, err = compiled.MatchAgainstDeployment(nil, ed)
	require.NoError(t, err)
	require.NotEmpty(t, violations.AlertViolations)
	assert.Contains(t, violations.AlertViolations[0].GetMessage(), "init-setup")

	processPolicy := &storage.Policy{
		PolicyVersion:   policyversion.CurrentVersion().String(),
		Name:            "process",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     storage.EventSource_DEPLOYMENT_EVENT,
		PolicySections: []*storage.PolicySection{{
			PolicyGroups: []*storage.PolicyGroup{{
				FieldName: fieldnames.ProcessName,
				Values:    []*storage.PolicyValue{{Value: "bash"}},
			}},
		}},
		EvaluationFilter: skipInit,
	}
	compiled, err = CompilePolicy(processPolicy, nil, nil)
	require.NoError(t, err)

	appProcess := &storage.ProcessIndicator{
		ContainerName: "app",
		Signal:        &storage.ProcessSignal{Name: "bash", ExecFilePath: "/bin/bash"},
	}
	violations, err = compiled.MatchAgainstDeploymentAndProcess(nil, ed, appProcess, true)
	require.NoError(t, err)
	assert.NotNil(t, violations.ProcessViolation)
}
