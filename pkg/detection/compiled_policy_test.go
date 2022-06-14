package detection

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/sliceutils"
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
				if sliceutils.Find(c.shouldApplyTo, dep) == -1 {
					assert.False(t, compiled.AppliesTo(dep), "Failed expectation for %s", dep.GetId())
				}
			}
		})
	}
}
