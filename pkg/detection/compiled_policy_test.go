package detection

import (
	"context"
	"slices"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/uuid"
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

// TestCompiledPolicyEvaluationFilter verifies that EvaluationFilters on a policy are
// honoured when the policy is compiled via newCompiledPolicy.
//
// This is the regression guard for the blank import of filtercompilers in
// compiled_policy.go. Without that import the IMAGE_LAYER compiler is never
// registered and the filter silently does nothing — this test would fail because
// both base-layer and app-layer components would produce violations regardless of
// the filter.
func TestCompiledPolicyEvaluationFilter(t *testing.T) {
	img := &storage.Image{
		Id:            uuid.NewV4().String(),
		Name:          &storage.ImageName{FullName: "docker.io/test/img"},
		BaseImageInfo: []*storage.BaseImageInfo{{MaxLayerIndex: 1}},
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "base-comp",
					Version: "1.0",
					HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{
						LayerIndex: 0,
					},
				},
				{
					Name:    "app-comp",
					Version: "1.0",
					HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{
						LayerIndex: 2,
					},
				},
			},
		},
	}
	dep := &storage.Deployment{
		Id:   uuid.NewV4().String(),
		Name: "test-dep",
		Containers: []*storage.Container{
			{Id: img.GetId(), Name: "c0"},
		},
	}
	ed := booleanpolicy.EnhancedDeployment{
		Deployment: dep,
		Images:     []*storage.Image{img},
		NetworkPoliciesApplied: &augmentedobjs.NetworkPoliciesApplied{
			HasIngressNetworkPolicy: true,
			HasEgressNetworkPolicy:  true,
		},
	}

	makePolicy := func(compName string, filter *storage.EvaluationFilter) *storage.Policy {
		return &storage.Policy{
			PolicyVersion:   policyversion.CurrentVersion().String(),
			Name:            "eval-filter-test-" + compName,
			LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
			PolicySections: []*storage.PolicySection{{
				PolicyGroups: []*storage.PolicyGroup{{
					FieldName: fieldnames.ImageComponent,
					Values:    []*storage.PolicyValue{{Value: compName + "="}},
				}},
			}},
			EvaluationFilter: filter,
		}
	}

	baseOnly := &storage.EvaluationFilter{SkipImageLayers: storage.SkipImageLayers_SKIP_APP}
	appOnly := &storage.EvaluationFilter{SkipImageLayers: storage.SkipImageLayers_SKIP_BASE}

	compile := func(t *testing.T, policy *storage.Policy) CompiledPolicy {
		t.Helper()
		cp, err := newCompiledPolicy(policy, nil, nil)
		require.NoError(t, err)
		return cp
	}

	fires := func(t *testing.T, cp CompiledPolicy) bool {
		t.Helper()
		var cache booleanpolicy.CacheReceptacle
		v, err := cp.MatchAgainstDeployment(&cache, ed)
		require.NoError(t, err)
		return len(v.AlertViolations) > 0
	}

	// BASE filter
	t.Run("base-comp/BASE fires", func(t *testing.T) {
		assert.True(t, fires(t, compile(t, makePolicy("base-comp", baseOnly))),
			"base-comp must match under BASE filter — FAIL means filtercompilers not registered in compiled_policy.go")
	})
	t.Run("app-comp/BASE does not fire", func(t *testing.T) {
		assert.False(t, fires(t, compile(t, makePolicy("app-comp", baseOnly))),
			"app-comp must not match under BASE filter")
	})

	// APP filter
	t.Run("app-comp/APP fires", func(t *testing.T) {
		assert.True(t, fires(t, compile(t, makePolicy("app-comp", appOnly))),
			"app-comp must match under APP filter — FAIL means filtercompilers not registered in compiled_policy.go")
	})
	t.Run("base-comp/APP does not fire", func(t *testing.T) {
		assert.False(t, fires(t, compile(t, makePolicy("base-comp", appOnly))),
			"base-comp must not match under APP filter")
	})

	// No filter: both visible.
	t.Run("no filter/both fire", func(t *testing.T) {
		assert.True(t, fires(t, compile(t, makePolicy("base-comp", nil))), "base-comp fires with no filter")
		assert.True(t, fires(t, compile(t, makePolicy("app-comp", nil))), "app-comp fires with no filter")
	})
}

// TestEvaluationFilterBothBranches verifies that skip_image_layers applies consistently
// to both Scan.Components and Metadata.V1.Layers on the same image through the full
// newCompiledPolicy → MatchAgainstDeployment path.
//
// Image layout (MaxLayerIndex = 1):
//
//	Layer 0: FROM  base:latest       → base  (j=0)
//	Layer 1: RUN   install-base-pkg  → base  (j=1, last base layer)
//	Layer 2: COPY  . /app            → app   (j=2)
//	Layer 3: RUN   install-app-pkg   → app   (j=3)
//
//	Component "base-pkg"  LayerIndex=0 → base
//	Component "app-pkg"   LayerIndex=2 → app
func TestEvaluationFilterBothBranches(t *testing.T) {
	img := &storage.Image{
		Id:            uuid.NewV4().String(),
		Name:          &storage.ImageName{FullName: "docker.io/test/full"},
		BaseImageInfo: []*storage.BaseImageInfo{{MaxLayerIndex: 1}},
		Metadata: &storage.ImageMetadata{V1: &storage.V1Metadata{
			Layers: []*storage.ImageLayer{
				{Instruction: "FROM", Value: "base:latest"},    // j=0 base
				{Instruction: "RUN", Value: "install-base-pkg"}, // j=1 base
				{Instruction: "COPY", Value: ". /app"},          // j=2 app
				{Instruction: "RUN", Value: "install-app-pkg"},  // j=3 app
			},
		}},
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{
				{Name: "base-pkg", Version: "1.0",
					HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0}},
				{Name: "app-pkg", Version: "1.0",
					HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 2}},
			},
		},
	}
	dep := &storage.Deployment{
		Id:         uuid.NewV4().String(),
		Containers: []*storage.Container{{Id: img.GetId(), Name: "c0"}},
	}
	ed := booleanpolicy.EnhancedDeployment{
		Deployment: dep,
		Images:     []*storage.Image{img},
		NetworkPoliciesApplied: &augmentedobjs.NetworkPoliciesApplied{
			HasIngressNetworkPolicy: true,
			HasEgressNetworkPolicy:  true,
		},
	}

	compile := func(t *testing.T, fieldName, value string, filter *storage.EvaluationFilter) CompiledPolicy {
		t.Helper()
		cp, err := newCompiledPolicy(&storage.Policy{
			PolicyVersion:   policyversion.CurrentVersion().String(),
			Name:            uuid.NewV4().String(),
			LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
			PolicySections: []*storage.PolicySection{{
				PolicyGroups: []*storage.PolicyGroup{{
					FieldName: fieldName,
					Values:    []*storage.PolicyValue{{Value: value}},
				}},
			}},
			EvaluationFilter: filter,
		}, nil, nil)
		require.NoError(t, err)
		return cp
	}

	fires := func(t *testing.T, cp CompiledPolicy) bool {
		t.Helper()
		var cache booleanpolicy.CacheReceptacle
		v, err := cp.MatchAgainstDeployment(&cache, ed)
		require.NoError(t, err)
		return len(v.AlertViolations) > 0
	}

	skipApp  := &storage.EvaluationFilter{SkipImageLayers: storage.SkipImageLayers_SKIP_APP}
	skipBase := &storage.EvaluationFilter{SkipImageLayers: storage.SkipImageLayers_SKIP_BASE}

	// ── Scan.Components branch ────────────────────────────────────────────────
	t.Run("component/base-pkg/SKIP_APP fires", func(t *testing.T) {
		assert.True(t, fires(t, compile(t, fieldnames.ImageComponent, "base-pkg=", skipApp)))
	})
	t.Run("component/app-pkg/SKIP_APP does not fire", func(t *testing.T) {
		assert.False(t, fires(t, compile(t, fieldnames.ImageComponent, "app-pkg=", skipApp)))
	})
	t.Run("component/app-pkg/SKIP_BASE fires", func(t *testing.T) {
		assert.True(t, fires(t, compile(t, fieldnames.ImageComponent, "app-pkg=", skipBase)))
	})
	t.Run("component/base-pkg/SKIP_BASE does not fire", func(t *testing.T) {
		assert.False(t, fires(t, compile(t, fieldnames.ImageComponent, "base-pkg=", skipBase)))
	})

	// ── Metadata.V1.Layers branch ─────────────────────────────────────────────
	t.Run("dockerfile/base RUN/SKIP_APP fires", func(t *testing.T) {
		assert.True(t, fires(t, compile(t, fieldnames.DockerfileLine, "RUN=install-base-pkg", skipApp)))
	})
	t.Run("dockerfile/app RUN/SKIP_APP does not fire", func(t *testing.T) {
		assert.False(t, fires(t, compile(t, fieldnames.DockerfileLine, "RUN=install-app-pkg", skipApp)))
	})
	t.Run("dockerfile/app RUN/SKIP_BASE fires", func(t *testing.T) {
		assert.True(t, fires(t, compile(t, fieldnames.DockerfileLine, "RUN=install-app-pkg", skipBase)))
	})
	t.Run("dockerfile/base RUN/SKIP_BASE does not fire", func(t *testing.T) {
		assert.False(t, fires(t, compile(t, fieldnames.DockerfileLine, "RUN=install-base-pkg", skipBase)))
	})

	// ── Consistency: same boundary applied to both branches ───────────────────
	t.Run("SKIP_APP: base component fires, app component silent", func(t *testing.T) {
		assert.True(t, fires(t, compile(t, fieldnames.ImageComponent, "base-pkg=", skipApp)))
		assert.False(t, fires(t, compile(t, fieldnames.ImageComponent, "app-pkg=", skipApp)))
	})
	t.Run("SKIP_APP: base dockerfile fires, app dockerfile silent", func(t *testing.T) {
		assert.True(t, fires(t, compile(t, fieldnames.DockerfileLine, "RUN=install-base-pkg", skipApp)))
		assert.False(t, fires(t, compile(t, fieldnames.DockerfileLine, "RUN=install-app-pkg", skipApp)))
	})
}

// TestEvaluationFilterSkipContainerType verifies that skip_container_types filters
// containers by type through the full newCompiledPolicy → MatchAgainstDeployment path.
//
// Deployment layout:
//   - Container "regular-c" (type=REGULAR): has component "regular-pkg" and dockerfile RUN
//   - Container "init-c"    (type=INIT):    has component "init-pkg"    and dockerfile RUN
//
// With SKIP_INIT set, the init container's subtree is never evaluated.
func TestEvaluationFilterSkipContainerType(t *testing.T) {
	regularImg := &storage.Image{
		Id:            uuid.NewV4().String(),
		Name:          &storage.ImageName{FullName: "docker.io/test/regular"},
		BaseImageInfo: []*storage.BaseImageInfo{{MaxLayerIndex: 0}},
		Metadata: &storage.ImageMetadata{V1: &storage.V1Metadata{
			Layers: []*storage.ImageLayer{{Instruction: "RUN", Value: "regular-cmd"}},
		}},
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{
				{Name: "regular-pkg", Version: "1.0",
					HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0}},
			},
		},
	}
	initImg := &storage.Image{
		Id:            uuid.NewV4().String(),
		Name:          &storage.ImageName{FullName: "docker.io/test/init"},
		BaseImageInfo: []*storage.BaseImageInfo{{MaxLayerIndex: 0}},
		Metadata: &storage.ImageMetadata{V1: &storage.V1Metadata{
			Layers: []*storage.ImageLayer{{Instruction: "RUN", Value: "init-cmd"}},
		}},
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{
				{Name: "init-pkg", Version: "1.0",
					HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0}},
			},
		},
	}

	dep := &storage.Deployment{
		Id: uuid.NewV4().String(),
		Containers: []*storage.Container{
			{Id: regularImg.GetId(), Name: "regular-c", Type: storage.ContainerType_REGULAR},
			{Id: initImg.GetId(), Name: "init-c", Type: storage.ContainerType_INIT},
		},
	}
	ed := booleanpolicy.EnhancedDeployment{
		Deployment: dep,
		Images:     []*storage.Image{regularImg, initImg},
		NetworkPoliciesApplied: &augmentedobjs.NetworkPoliciesApplied{
			HasIngressNetworkPolicy: true,
			HasEgressNetworkPolicy:  true,
		},
	}

	skipInit := &storage.EvaluationFilter{
		SkipContainerTypes: []storage.SkipContainerType{storage.SkipContainerType_SKIP_INIT},
	}

	compile := func(t *testing.T, fieldName, value string, filter *storage.EvaluationFilter) CompiledPolicy {
		t.Helper()
		cp, err := newCompiledPolicy(&storage.Policy{
			PolicyVersion:   policyversion.CurrentVersion().String(),
			Name:            uuid.NewV4().String(),
			LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
			PolicySections: []*storage.PolicySection{{
				PolicyGroups: []*storage.PolicyGroup{{
					FieldName: fieldName,
					Values:    []*storage.PolicyValue{{Value: value}},
				}},
			}},
			EvaluationFilter: filter,
		}, nil, nil)
		require.NoError(t, err)
		return cp
	}

	fires := func(t *testing.T, cp CompiledPolicy) bool {
		t.Helper()
		var cache booleanpolicy.CacheReceptacle
		v, err := cp.MatchAgainstDeployment(&cache, ed)
		require.NoError(t, err)
		return len(v.AlertViolations) > 0
	}

	// Without filter: both containers evaluated.
	t.Run("no filter/regular-pkg fires", func(t *testing.T) {
		assert.True(t, fires(t, compile(t, fieldnames.ImageComponent, "regular-pkg=", nil)))
	})
	t.Run("no filter/init-pkg fires", func(t *testing.T) {
		assert.True(t, fires(t, compile(t, fieldnames.ImageComponent, "init-pkg=", nil)))
	})

	// With SKIP_INIT: only regular container evaluated.
	t.Run("SKIP_INIT/regular-pkg fires", func(t *testing.T) {
		assert.True(t, fires(t, compile(t, fieldnames.ImageComponent, "regular-pkg=", skipInit)))
	})
	t.Run("SKIP_INIT/init-pkg does not fire", func(t *testing.T) {
		assert.False(t, fires(t, compile(t, fieldnames.ImageComponent, "init-pkg=", skipInit)),
			"init container must be skipped entirely — its component must not produce a violation")
	})

	// Dockerfile layer criteria in the init container are also skipped.
	t.Run("SKIP_INIT/init dockerfile does not fire", func(t *testing.T) {
		assert.False(t, fires(t, compile(t, fieldnames.DockerfileLine, "RUN=init-cmd", skipInit)),
			"dockerfile criteria in init container must not fire when SKIP_INIT is set")
	})
	t.Run("SKIP_INIT/regular dockerfile fires", func(t *testing.T) {
		assert.True(t, fires(t, compile(t, fieldnames.DockerfileLine, "RUN=regular-cmd", skipInit)))
	})
}

// TestEvaluationFilterCombined verifies that skip_image_layers and skip_container_types
// work together on the same deployment through newCompiledPolicy → MatchAgainstDeployment.
//
// Deployment layout (MaxLayerIndex = 1 on all images):
//
//	Container "regular-c" (REGULAR): base-comp (Li=0), app-comp (Li=2), base RUN, app RUN
//	Container "init-c"    (INIT):    base-comp (Li=0), app-comp (Li=2), base RUN, app RUN
//
// Filters applied together:
//
//	skip_image_layers     = SKIP_APP   → only base-layer elements visible
//	skip_container_types  = [SKIP_INIT] → init container entirely excluded
//
// Expected outcome (AND semantics):
//
//	Regular + base → visible   (passes both filters)
//	Regular + app  → excluded  (image layer filter)
//	Init    + base → excluded  (container type filter)
//	Init    + app  → excluded  (both filters)
func TestEvaluationFilterCombined(t *testing.T) {
	makeImage := func(fullName string) *storage.Image {
		return &storage.Image{
			Id:            uuid.NewV4().String(),
			Name:          &storage.ImageName{FullName: fullName},
			BaseImageInfo: []*storage.BaseImageInfo{{MaxLayerIndex: 1}},
			Metadata: &storage.ImageMetadata{V1: &storage.V1Metadata{
				Layers: []*storage.ImageLayer{
					{Instruction: "RUN", Value: "base-cmd"}, // j=0 base
					{Instruction: "RUN", Value: "base-cmd2"}, // j=1 base
					{Instruction: "RUN", Value: "app-cmd"},  // j=2 app
				},
			}},
			Scan: &storage.ImageScan{
				Components: []*storage.EmbeddedImageScanComponent{
					{Name: "base-comp", Version: "1.0",
						HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0}},
					{Name: "app-comp", Version: "1.0",
						HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 2}},
				},
			},
		}
	}

	regularImg := makeImage("docker.io/test/regular")
	initImg    := makeImage("docker.io/test/init")

	dep := &storage.Deployment{
		Id: uuid.NewV4().String(),
		Containers: []*storage.Container{
			{Id: regularImg.GetId(), Name: "regular-c", Type: storage.ContainerType_REGULAR},
			{Id: initImg.GetId(),    Name: "init-c",    Type: storage.ContainerType_INIT},
		},
	}
	ed := booleanpolicy.EnhancedDeployment{
		Deployment: dep,
		Images:     []*storage.Image{regularImg, initImg},
		NetworkPoliciesApplied: &augmentedobjs.NetworkPoliciesApplied{
			HasIngressNetworkPolicy: true,
			HasEgressNetworkPolicy:  true,
		},
	}

	// Both filters active simultaneously.
	combined := &storage.EvaluationFilter{
		SkipImageLayers:    storage.SkipImageLayers_SKIP_APP,
		SkipContainerTypes: []storage.SkipContainerType{storage.SkipContainerType_SKIP_INIT},
	}

	compile := func(t *testing.T, fieldName, value string, filter *storage.EvaluationFilter) CompiledPolicy {
		t.Helper()
		cp, err := newCompiledPolicy(&storage.Policy{
			PolicyVersion:   policyversion.CurrentVersion().String(),
			Name:            uuid.NewV4().String(),
			LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
			PolicySections: []*storage.PolicySection{{
				PolicyGroups: []*storage.PolicyGroup{{
					FieldName: fieldName,
					Values:    []*storage.PolicyValue{{Value: value}},
				}},
			}},
			EvaluationFilter: filter,
		}, nil, nil)
		require.NoError(t, err)
		return cp
	}

	fires := func(t *testing.T, cp CompiledPolicy) bool {
		t.Helper()
		var cache booleanpolicy.CacheReceptacle
		v, err := cp.MatchAgainstDeployment(&cache, ed)
		require.NoError(t, err)
		return len(v.AlertViolations) > 0
	}

	// ── Component criteria ────────────────────────────────────────────────────

	t.Run("regular+base-comp fires (passes both filters)", func(t *testing.T) {
		assert.True(t, fires(t, compile(t, fieldnames.ImageComponent, "base-comp=", combined)))
	})
	t.Run("regular+app-comp silent (image layer filter)", func(t *testing.T) {
		assert.False(t, fires(t, compile(t, fieldnames.ImageComponent, "app-comp=", combined)))
	})
	t.Run("init+base-comp silent (container type filter)", func(t *testing.T) {
		// base-comp exists only in the init container after we use "init-c" image only.
		// Since both images have the same component names, we distinguish by using the
		// no-filter baseline: with only SKIP_INIT, base-comp in init container is excluded.
		skipInitOnly := &storage.EvaluationFilter{
			SkipContainerTypes: []storage.SkipContainerType{storage.SkipContainerType_SKIP_INIT},
		}
		// Without filter: base-comp fires (from both containers).
		assert.True(t, fires(t, compile(t, fieldnames.ImageComponent, "base-comp=", nil)))
		// With SKIP_INIT only: base-comp still fires (from regular container).
		assert.True(t, fires(t, compile(t, fieldnames.ImageComponent, "base-comp=", skipInitOnly)))
		// With combined: base-comp fires (regular container, base layer — both pass).
		assert.True(t, fires(t, compile(t, fieldnames.ImageComponent, "base-comp=", combined)))
		// With combined: app-comp does NOT fire (image layer filter blocks it in regular;
		// container type filter blocks init entirely).
		assert.False(t, fires(t, compile(t, fieldnames.ImageComponent, "app-comp=", combined)))
	})

	// ── Dockerfile layer criteria ─────────────────────────────────────────────

	t.Run("regular+base RUN fires (passes both filters)", func(t *testing.T) {
		assert.True(t, fires(t, compile(t, fieldnames.DockerfileLine, "RUN=base-cmd", combined)))
	})
	t.Run("regular+app RUN silent (image layer filter)", func(t *testing.T) {
		assert.False(t, fires(t, compile(t, fieldnames.DockerfileLine, "RUN=app-cmd", combined)))
	})
	t.Run("init dockerfile silent regardless of layer (container type filter)", func(t *testing.T) {
		// Both images have the same RUN values, so we test the combined effect:
		// app-cmd is an app-layer RUN — blocked by image layer filter even in regular container,
		// and the entire init container is excluded by the container type filter.
		assert.False(t, fires(t, compile(t, fieldnames.DockerfileLine, "RUN=app-cmd", combined)))
	})

	// ── Confirm no-filter baseline: all four combinations fire ─────────────────

	t.Run("no filter: base-comp fires from both containers", func(t *testing.T) {
		assert.True(t, fires(t, compile(t, fieldnames.ImageComponent, "base-comp=", nil)))
	})
	t.Run("no filter: app-comp fires from both containers", func(t *testing.T) {
		assert.True(t, fires(t, compile(t, fieldnames.ImageComponent, "app-comp=", nil)))
	})
	t.Run("no filter: base RUN fires", func(t *testing.T) {
		assert.True(t, fires(t, compile(t, fieldnames.DockerfileLine, "RUN=base-cmd", nil)))
	})
	t.Run("no filter: app RUN fires", func(t *testing.T) {
		assert.True(t, fires(t, compile(t, fieldnames.DockerfileLine, "RUN=app-cmd", nil)))
	})
}
