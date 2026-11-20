package centralservices

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	defaultPolicies "github.com/stackrox/rox/pkg/defaults/policies"
	"github.com/stackrox/rox/pkg/protoconv/resources"
	"github.com/stretchr/testify/suite"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// secretEnvVarPolicyID is the ID of the default policy "Environment Variable Contains Secret"
// (pkg/defaults/policies/files/secret_env.json). The policy matches env var NAMES (the KEY
// part of the augmented "SOURCE\tKEY\tVALUE" string, see
// pkg/booleanpolicy/augmentedobjs/construct.go) against the case-insensitive regex
// (.*SECRET.*|.*PASSWORD.*) — the env var VALUE is never inspected.
const secretEnvVarPolicyID = "f4996314-c3d7-4553-803b-b24ce7febe48" // #nosec G101 -- a policy ID, not a credential

type defaultPoliciesOnChartSuite struct {
	baseSuite
}

func TestDefaultPoliciesOnRenderedChart(t *testing.T) {
	suite.Run(t, new(defaultPoliciesOnChartSuite))
}

// TestScannerV4DBDoesNotViolateSecretEnvVarPolicy guards ROX-39749: the default policy
// "Environment Variable Contains Secret" must not fire on our own scanner-v4-db deployment.
//
// The test renders the real central-services chart (scannerV4 enabled via the
// autogenerate-all test values), converts the scanner-v4-db Deployment into a
// storage.Deployment through the same conversion Sensor uses in production
// (pkg/protoconv/resources), and runs the real default policy against it. Init
// containers are part of storage.Deployment.Containers
// (pkg/protoconv/resources/resources.go, populateContainers), so their env vars are
// evaluated by the policy engine just like regular container env vars.
//
// If this test fails, an env var name in the scanner-v4-db deployment (most likely in
// the init-db init container) matches the policy regex, e.g. POSTGRES_PASSWORD_FILE.
// Rename the env var so its name does not contain SECRET or PASSWORD, and update
// scanner/image/db/scripts/init-entrypoint.sh accordingly.
func (s *defaultPoliciesOnChartSuite) TestScannerV4DBDoesNotViolateSecretEnvVarPolicy() {
	policy := s.mustFindPolicyByID(secretEnvVarPolicyID)

	for _, dep := range s.renderScannerV4DBDeployments() {
		violations := s.matchDeployment(policy, dep)
		s.Empty(violations.AlertViolations,
			"default policy %q must not fire on deployment %q: env var names must not match the policy regex (ROX-39749)",
			policy.GetName(), dep.GetName())
	}
}

// TestSecretEnvVarPolicyStillDetectsBadEnvVarNames is the negative control for
// TestScannerV4DBDoesNotViolateSecretEnvVarPolicy: it proves the assertion above can
// actually fail, by re-introducing the pre-ROX-39749 env var name
// (POSTGRES_PASSWORD_FILE, holding only a file path) into a copy of the rendered
// deployment and checking that the policy does fire.
func (s *defaultPoliciesOnChartSuite) TestSecretEnvVarPolicyStillDetectsBadEnvVarNames() {
	policy := s.mustFindPolicyByID(secretEnvVarPolicyID)

	for _, dep := range s.renderScannerV4DBDeployments() {
		depWithBadEnvVar := dep.CloneVT()

		var initContainer *storage.Container
		for _, c := range depWithBadEnvVar.GetContainers() {
			if c.GetType() == storage.ContainerType_INIT {
				initContainer = c
				break
			}
		}
		s.Require().NotNil(initContainer, "scanner-v4-db deployment should have an init container")

		if initContainer.GetConfig() == nil {
			initContainer.Config = &storage.ContainerConfig{}
		}
		initContainer.Config.Env = append(initContainer.Config.Env, &storage.ContainerConfig_EnvironmentConfig{
			Key:          "POSTGRES_PASSWORD_FILE",
			Value:        "/run/secrets/stackrox.io/secrets/password",
			EnvVarSource: storage.ContainerConfig_EnvironmentConfig_RAW,
		})

		violations := s.matchDeployment(policy, depWithBadEnvVar)
		s.NotEmpty(violations.AlertViolations,
			"negative control: policy %q must fire when an env var name matching the policy regex (POSTGRES_PASSWORD_FILE) is (re)introduced",
			policy.GetName())
	}
}

func (s *defaultPoliciesOnChartSuite) mustFindPolicyByID(id string) *storage.Policy {
	policyList, err := defaultPolicies.DefaultPolicies()
	s.Require().NoError(err, "loading default policies")
	for _, p := range policyList {
		if p.GetId() == id {
			return p
		}
	}
	s.FailNowf("default policy with ID %q not found", id)
	return nil
}

// renderScannerV4DBDeployments renders the central-services chart with the
// autogenerate-all values (which enable scannerV4, see testdata/autogenerate-all.yaml)
// and returns the scanner-v4-db Deployment converted to a storage.Deployment.
func (s *defaultPoliciesOnChartSuite) renderScannerV4DBDeployments() []*storage.Deployment {
	_, rendered := s.LoadAndRender(autogenerateAll)

	var result []*storage.Deployment
	for _, obj := range s.ParseObjects(rendered) {
		if obj.GetKind() != "Deployment" || obj.GetName() != "scanner-v4-db" {
			continue
		}
		var k8sDep appsv1.Deployment
		s.Require().NoError(
			runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &k8sDep),
			"converting rendered scanner-v4-db Deployment to a typed object")
		storageDep, err := resources.NewDeploymentFromStaticResource(&k8sDep, "Deployment", "test-cluster", "")
		s.Require().NoError(err, "converting scanner-v4-db Deployment to storage.Deployment")
		s.Require().NotNil(storageDep)
		s.Require().NotEmpty(storageDep.GetContainers(), "scanner-v4-db deployment should have containers")
		result = append(result, storageDep)
	}
	s.Require().NotEmpty(result, "scanner-v4-db Deployment not found in rendered central-services chart")
	return result
}

func (s *defaultPoliciesOnChartSuite) matchDeployment(policy *storage.Policy, dep *storage.Deployment) booleanpolicy.Violations {
	matcher, err := booleanpolicy.BuildDeploymentMatcher(policy)
	s.Require().NoError(err, "building deployment matcher for policy %q", policy.GetName())
	violations, err := matcher.MatchDeployment(nil, booleanpolicy.EnhancedDeployment{
		Deployment: dep,
		// The policy under test does not inspect images; ConstructDeployment only
		// requires one (possibly empty) image per container.
		Images:                 make([]*storage.Image, len(dep.GetContainers())),
		NetworkPoliciesApplied: &augmentedobjs.NetworkPoliciesApplied{},
	})
	s.Require().NoError(err, "matching policy %q against deployment %q", policy.GetName(), dep.GetName())
	return violations
}
