package detection

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func constructPolicy(scopes []*storage.Scope, whitelists []*storage.Whitelist) *storage.Policy {
	return &storage.Policy{PolicyVersion: booleanpolicy.Version, Name: "testname", Scope: scopes, Whitelists: whitelists, LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY}}
}

func newDeployment(id string) *storage.Deployment {
	dep := fixtures.GetDeployment()
	dep.Id = id
	return dep
}

func TestCompiledPolicyScopesAndWhitelists(t *testing.T) {
	envIsolator := testutils.NewEnvIsolator(t)
	defer envIsolator.RestoreAll()
	envIsolator.Setenv(features.BooleanPolicyLogic.EnvVar(), "true")

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
		whitelists    []*storage.Whitelist
		shouldApplyTo []*storage.Deployment
	}{
		{
			desc:          "no scopes or whitelists",
			shouldApplyTo: []*storage.Deployment{stackRoxNSDep, defaultNSDep, appStackRoxDep},
		},
		{
			desc:          "only stackrox ns",
			scopes:        []*storage.Scope{stackRoxNSScope},
			shouldApplyTo: []*storage.Deployment{stackRoxNSDep, appStackRoxDep},
		},
		{
			desc:          "only stackrox ns, but app=stackrox whitelisted",
			scopes:        []*storage.Scope{stackRoxNSScope},
			whitelists:    []*storage.Whitelist{{Deployment: &storage.Whitelist_Deployment{Scope: appStackRoxScope}}},
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
			compiled, err := NewPolicyCompiler().CompilePolicy(constructPolicy(c.scopes, c.whitelists))
			require.NoError(t, err)
			for _, dep := range c.shouldApplyTo {
				assert.True(t, compiled.AppliesTo(dep), "Failed expectation for %s", dep.GetId())
			}
			for _, dep := range allDeps {
				if sliceutils.Find(c.shouldApplyTo, dep) == -1 {
					assert.False(t, compiled.AppliesTo(dep), "Failed expectation for %s", dep.GetId())
				}
			}
		})
	}
}
