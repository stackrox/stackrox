package role

import (
	"context"
	"testing"

	"github.com/graph-gophers/graphql-go/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/tests/resource"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	v12 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

var (
	NginxDeployment       = resource.YamlTestFile{Kind: "Deployment", File: "nginx.yaml"}
	NginxRole             = resource.YamlTestFile{Kind: "Role", File: "nginx-role.yaml"}
	NginxRoleBinding      = resource.YamlTestFile{Kind: "Binding", File: "nginx-binding.yaml"}
	NginxRoleGroupBinding = resource.YamlTestFile{Kind: "Binding", File: "nginx-binding-group.yaml"}
)

type RoleDependencySuite struct {
	resyncEnabled bool
	testContext   *resource.TestContext
	suite.Suite
}

func Test_RoleDependency(t *testing.T) {
	suite.Run(t, &RoleDependencySuite{resyncEnabled: false})
}

var _ suite.SetupAllSuite = &RoleDependencySuite{}
var _ suite.TearDownTestSuite = &RoleDependencySuite{}

func (s *RoleDependencySuite) TearDownTest() {
	// Clear any messages received in fake central during the test run
	s.testContext.GetFakeCentral().ClearReceivedBuffer()
}

func (s *RoleDependencySuite) SetupSuite() {
	config := resource.DefaultTestSetupConfig()
	if !s.resyncEnabled {
		s.T().Setenv("ROX_RESYNC_DISABLED", "true")
		config.SensorResyncTime = 0
	}
	if testContext, err := resource.NewContextWithConfig(s.T(), config); err != nil {
		s.Fail("failed to setup test context: %s", err)
	} else {
		s.testContext = testContext
	}
}

func assertPermissionLevel(permissionLevel storage.PermissionLevel) resource.AssertFunc {
	return func(deployment *storage.Deployment) error {
		if deployment.ServiceAccountPermissionLevel != permissionLevel {
			return errors.Errorf("expected permission level %s but found %s", permissionLevel, deployment.ServiceAccountPermissionLevel)
		}
		return nil
	}

}

func assertBindingHasRoleId(roleId string) resource.AssertFuncAny {
	return func(obj interface{}) error {
		evt, ok := obj.(*central.SensorEvent)
		if !ok {
			return errors.Errorf("not an event")
		}
		binding := evt.GetBinding()
		if binding.GetRoleId() != roleId {
			return errors.Errorf("expected \"%s\" but found \"%s\"", roleId, binding.GetRoleId())
		}
		return nil
	}
}

func (s *RoleDependencySuite) Test_PermutationTest() {
	s.testContext.GetFakeCentral().ClearReceivedBuffer()
	s.testContext.RunWithResourcesPermutation(
		[]resource.YamlTestFile{
			NginxDeployment,
			NginxRole,
			NginxRoleBinding,
		}, "RoleDependency", func(t *testing.T, testC *resource.TestContext, objects map[string]k8s.Object) {
			testC.LastDeploymentState("nginx-deployment",
				assertPermissionLevel(storage.PermissionLevel_ELEVATED_IN_NAMESPACE),
				"Permission level has to be elevated in namespace")
			testC.GetFakeCentral().ClearReceivedBuffer()
		},
	)
}

func matchBinding(namespace, id string) resource.MatchResource {
	return func(resource *central.MsgFromSensor) bool {
		if resource.GetEvent() == nil || resource.GetEvent().GetBinding() == nil {
			return false
		}
		return resource.GetEvent().GetBinding().GetId() == id && resource.GetEvent().GetBinding().GetNamespace() == namespace
	}
}

func (s *RoleDependencySuite) Test_BindingHasNoRoleId() {
	s.testContext.RunBare("Binding should get an update if role gets created after", func(t *testing.T, testC *resource.TestContext, _ map[string]k8s.Object) {
		deleteDep, err := testC.ApplyFileNoObject(context.Background(), "sensor-integration", NginxDeployment)
		defer utils.IgnoreError(deleteDep)
		require.NoError(t, err)

		var binding v12.RoleBinding
		deleteRoleBinding, err := testC.ApplyFile(context.Background(), "sensor-integration", NginxRoleBinding, &binding)
		defer utils.IgnoreError(deleteRoleBinding)
		require.NoError(t, err)

		testC.LastResourceState(matchBinding(binding.GetNamespace(), string(binding.GetUID())), assertBindingHasRoleId(""), "No RoleID")

		var role v12.Role
		deleteRole, err := testC.ApplyFile(context.Background(), "sensor-integration", NginxRole, &role)
		defer utils.IgnoreError(deleteRole)
		require.NoError(t, err)

		testC.LastResourceState(matchBinding(binding.GetNamespace(), string(binding.GetUID())), assertBindingHasRoleId(string(role.GetUID())), "Has RoleID")

		testC.GetFakeCentral().ClearReceivedBuffer()

		utils.IgnoreError(deleteRole)
		utils.IgnoreError(deleteRoleBinding)
		utils.IgnoreError(deleteDep)

		testC.GetFakeCentral().ClearReceivedBuffer()
	})
}

func (s *RoleDependencySuite) Test_GroupSubjects() {
	s.T().Skip()
	s.testContext.RunWithResources(
		[]resource.YamlTestFile{
			NginxDeployment,
			NginxRole,
			NginxRoleGroupBinding,
		},
		func(t *testing.T, testC *resource.TestContext, _ map[string]k8s.Object) {
			// This test expects that a reference to a non-ServiceAccount subject (i.e. Group or User) does not
			// reference any deployments and thus a deployment object should not be updated. However, Groups can be
			// used to reference a set of ServiceAccounts, see: https://kubernetes.io/docs/reference/access-authn-authz/rbac/#role-binding-examples
			// Using `system:serviceaccounts:sensor-integration` as a group to set PermissionLevel to all deployments
			// with any ServiceAccount is not supported by ACS.
			testC.LastDeploymentState("nginx-deployment",
				assertPermissionLevel(storage.PermissionLevel_NONE),
				"Group / User permission levels should be ignored")
			testC.GetFakeCentral().ClearReceivedBuffer()
		},
	)
}

func (s *RoleDependencySuite) Test_PermissionLevelIsNone() {
	s.testContext.RunWithResources(
		[]resource.YamlTestFile{
			NginxDeployment,
			NginxRole,
		}, func(t *testing.T, testC *resource.TestContext, _ map[string]k8s.Object) {
			testC.LastDeploymentState("nginx-deployment",
				assertPermissionLevel(storage.PermissionLevel_NONE),
				"Permission level has to be none if role binding is missing")
			testC.GetFakeCentral().ClearReceivedBuffer()
		})
}

func (s *RoleDependencySuite) Test_MultipleDeploymentUpdates() {
	s.testContext.RunBare("Update permission level", func(t *testing.T, testC *resource.TestContext, _ map[string]k8s.Object) {
		deleteDep, err := testC.ApplyFileNoObject(context.Background(), "sensor-integration", NginxDeployment)
		defer utils.IgnoreError(deleteDep)
		require.NoError(t, err)

		deleteRoleBinding, err := testC.ApplyFileNoObject(context.Background(), "sensor-integration", NginxRoleBinding)
		defer utils.IgnoreError(deleteRoleBinding)
		require.NoError(t, err)

		deleteRole, err := testC.ApplyFileNoObject(context.Background(), "sensor-integration", NginxRole)

		defer utils.IgnoreError(deleteRole)
		require.NoError(t, err)

		testC.LastDeploymentState("nginx-deployment",
			assertPermissionLevel(storage.PermissionLevel_ELEVATED_IN_NAMESPACE),
			"Permission level has to be elevated in namespace")
		testC.GetFakeCentral().ClearReceivedBuffer()

		utils.IgnoreError(deleteRole)
		utils.IgnoreError(deleteRoleBinding)

		testC.LastDeploymentState("nginx-deployment",
			assertPermissionLevel(storage.PermissionLevel_NONE),
			"Permission level has to be none after deleting role and binding")
		testC.GetFakeCentral().ClearReceivedBuffer()
	})
}
