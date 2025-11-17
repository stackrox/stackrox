package detection

import (
	"slices"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
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
			compiled, err := CompilePolicy(constructPolicy(c.scopes, c.exclusions))
			require.NoError(t, err)
			for _, dep := range c.shouldApplyTo {
				assert.True(t, compiled.AppliesTo(dep), "Failed expectation for %s", dep.GetId())
			}
			for _, dep := range allDeps {
				if slices.Index(c.shouldApplyTo, dep) == -1 {
					assert.False(t, compiled.AppliesTo(dep), "Failed expectation for %s", dep.GetId())
				}
			}
		})
	}
}

func TestNodeEventMatcher(t *testing.T) {
	policy := &storage.Policy{
		PolicyVersion:   policyversion.CurrentVersion().String(),
		Name:            "Node File Access Policy",
		EventSource:     storage.EventSource_NODE_EVENT,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.NodeFilePath,
						Values:    []*storage.PolicyValue{{Value: "/etc/passwd"}},
					},
				},
			},
		},
	}

	compiled, err := CompilePolicy(policy)
	require.NoError(t, err)

	node := &storage.Node{Id: "node-1", Name: "test-node"}
	access := &storage.FileAccess{
		File: &storage.FileAccess_File{NodePath: "/etc/passwd"},
	}

	var cache booleanpolicy.CacheReceptacle
	violations, err := compiled.MatchAgainstNodeAndFileAccess(&cache, node, access)
	require.NoError(t, err)

	assert.NotNil(t, violations.FileAccessViolation)
	assert.NotEmpty(t, violations.FileAccessViolation.Accesses)
}

func TestNodeEventMatcherNoMatch(t *testing.T) {
	policy := &storage.Policy{
		PolicyVersion:   policyversion.CurrentVersion().String(),
		Name:            "Node File Access Policy",
		EventSource:     storage.EventSource_NODE_EVENT,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.NodeFilePath,
						Values:    []*storage.PolicyValue{{Value: "/etc/passwd"}},
					},
				},
			},
		},
	}

	compiled, err := CompilePolicy(policy)
	require.NoError(t, err)

	node := &storage.Node{Id: "node-1", Name: "test-node"}
	access := &storage.FileAccess{
		File: &storage.FileAccess_File{NodePath: "/tmp/other-file"},
	}

	var cache booleanpolicy.CacheReceptacle
	violations, err := compiled.MatchAgainstNodeAndFileAccess(&cache, node, access)
	require.NoError(t, err)
	assert.Nil(t, violations.FileAccessViolation, "expected no violations for non-matching file path")
}

func TestNodeEventMatcherError(t *testing.T) {
	// Test that calling MatchAgainstNodeAndFileAccess on a non-node policy returns error
	policy := &storage.Policy{
		PolicyVersion:   policyversion.CurrentVersion().String(),
		Name:            "Deployment Policy",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.ImageRegistry,
						Values:    []*storage.PolicyValue{{Value: "docker.io"}},
					},
				},
			},
		},
	}

	compiled, err := CompilePolicy(policy)
	require.NoError(t, err)

	node := &storage.Node{Id: "node-1"}
	access := &storage.FileAccess{}

	var cache booleanpolicy.CacheReceptacle
	_, err = compiled.MatchAgainstNodeAndFileAccess(&cache, node, access)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "couldn't match policy")
}

func TestNodeEventMatcherWithMultipleSections(t *testing.T) {
	policy := &storage.Policy{
		PolicyVersion:   policyversion.CurrentVersion().String(),
		Name:            "Node File Access Policy with Multiple Sections",
		EventSource:     storage.EventSource_NODE_EVENT,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "Section 1",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.NodeFilePath,
						Values:    []*storage.PolicyValue{{Value: "/etc/passwd"}},
					},
				},
			},
			{
				SectionName: "Section 2",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.NodeFilePath,
						Values:    []*storage.PolicyValue{{Value: "/etc/shadow"}},
					},
				},
			},
		},
	}

	compiled, err := CompilePolicy(policy)
	require.NoError(t, err)

	node := &storage.Node{Id: "node-1", Name: "test-node"}

	// Match first section
	access1 := &storage.FileAccess{
		File: &storage.FileAccess_File{NodePath: "/etc/passwd"},
	}

	var cache booleanpolicy.CacheReceptacle
	violations1, err := compiled.MatchAgainstNodeAndFileAccess(&cache, node, access1)
	require.NoError(t, err)
	assert.NotNil(t, violations1.FileAccessViolation, "expected violation for /etc/passwd")

	// Match second section
	access2 := &storage.FileAccess{
		File: &storage.FileAccess_File{NodePath: "/etc/shadow"},
	}

	violations2, err := compiled.MatchAgainstNodeAndFileAccess(&cache, node, access2)
	require.NoError(t, err)
	assert.NotNil(t, violations2.FileAccessViolation, "expected violation for /etc/shadow")

	// No match
	access3 := &storage.FileAccess{
		File: &storage.FileAccess_File{NodePath: "/tmp/file"},
	}

	violations3, err := compiled.MatchAgainstNodeAndFileAccess(&cache, node, access3)
	require.NoError(t, err)
	assert.NotNil(t, violations3.FileAccessViolation, "expected no violations for /tmp/file")
}

func TestNodeEventMatcherNegatedOperation(t *testing.T) {
	policy := &storage.Policy{
		PolicyVersion:   policyversion.CurrentVersion().String(),
		Name:            "Node File Access Policy with Negated Operation",
		EventSource:     storage.EventSource_NODE_EVENT,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "Section 1",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.NodeFilePath,
						Values:    []*storage.PolicyValue{{Value: "/etc/passwd"}},
					},
					{
						FieldName: fieldnames.FileOperation,
						Values:    []*storage.PolicyValue{{Value: "CREATE"}},
						Negate:    true,
					},
				},
			},
		},
	}

	compiled, err := CompilePolicy(policy)
	require.NoError(t, err)

	var cache booleanpolicy.CacheReceptacle
	node := &storage.Node{Id: "node-1", Name: "test-node"}

	access := &storage.FileAccess{
		File:      &storage.FileAccess_File{NodePath: "/etc/passwd"},
		Operation: storage.FileAccess_CREATE,
	}

	violations1, err := compiled.MatchAgainstNodeAndFileAccess(&cache, node, access)
	require.NoError(t, err)
	assert.Nil(t, violations1.FileAccessViolation, "expected no violations for /etc/passwd CREATE")

	// change to operation within the accepted set
	access.Operation = storage.FileAccess_OWNERSHIP_CHANGE

	violations2, err := compiled.MatchAgainstNodeAndFileAccess(&cache, node, access)
	require.NoError(t, err)
	assert.NotNil(t, violations2.FileAccessViolation, "expected violations for /etc/passwd OWNERSHIP_CHANGE")

	policy2 := &storage.Policy{
		PolicyVersion:   policyversion.CurrentVersion().String(),
		Name:            "Node File Access Policy with multiple Negated Operations",
		EventSource:     storage.EventSource_NODE_EVENT,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "Section 1",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.NodeFilePath,
						Values:    []*storage.PolicyValue{{Value: "/etc/passwd"}},
					},
					{
						FieldName: fieldnames.FileOperation,
						Values:    []*storage.PolicyValue{{Value: "CREATE"}, {Value: "UNLINK"}},
						Negate:    true,
					},
				},
			},
		},
	}

	compiled2, err := CompilePolicy(policy2)
	require.NoError(t, err)

	violations3, err := compiled2.MatchAgainstNodeAndFileAccess(&cache, node, access)
	require.NoError(t, err)
	assert.NotNil(t, violations3.FileAccessViolation, "expected violations for /etc/passwd OWNERSHIP_CHANGE")
}
