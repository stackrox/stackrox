package role

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/tests/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

var (
	NginxDeployment  = resource.YamlTestFile{Kind: "Deployment", File: "nginx.yaml"}
	NginxRole        = resource.YamlTestFile{Kind: "Role", File: "nginx-role.yaml"}
	NginxRoleBinding = resource.YamlTestFile{Kind: "Binding", File: "nginx-binding.yaml"}
)

func GetLastMessageWithDeploymentName(messages []*central.MsgFromSensor, n string) *central.MsgFromSensor {
	var lastMessage *central.MsgFromSensor
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].GetEvent().GetDeployment().GetName() == n {
			lastMessage = messages[i]
			break
		}
	}
	return lastMessage
}

func assertLastDeploymentHasPermissionLevel(t *testing.T, messages []*central.MsgFromSensor, permissionLevel storage.PermissionLevel) {
	lastNginxDeploymentUpdate := GetLastMessageWithDeploymentName(messages, "nginx-deployment")
	require.NotNil(t, lastNginxDeploymentUpdate, "should have found a message for nginx-deployment")
	deployment := lastNginxDeploymentUpdate.GetEvent().GetDeployment()
	assert.Equal(
		t,
		deployment.ServiceAccountPermissionLevel,
		permissionLevel,
		fmt.Sprintf("permission level has to be %s", permissionLevel),
	)
}

type RoleDependencySuite struct {
	testContext *resource.TestContext
	suite.Suite
}

func Test_RoleDependency(t *testing.T) {
	suite.Run(t, new(RoleDependencySuite))
}

var _ suite.SetupAllSuite = &RoleDependencySuite{}
var _ suite.TearDownTestSuite = &RoleDependencySuite{}

func (s *RoleDependencySuite) TearDownTest() {
	// Clear any messages received in fake central during the test run
	s.testContext.GetFakeCentral().ClearReceivedBuffer()
}

func (s *RoleDependencySuite) SetupSuite() {
	if testContext, err := resource.NewContext(s.T()); err != nil {
		s.Fail("failed to setup test context: %s", err)
	} else {
		s.testContext = testContext
	}
}

func (s *RoleDependencySuite) Test_PermutationTest() {
	s.testContext.RunWithResourcesPermutation(
		[]resource.YamlTestFile{
			NginxDeployment,
			NginxRole,
			NginxRoleBinding,
		}, "Role Dependency", func(t *testing.T, testC *resource.TestContext, _ map[string]k8s.Object) {
			// Test context already takes care of creating and destroying resources
			time.Sleep(2 * time.Second)
			assertLastDeploymentHasPermissionLevel(
				t,
				testC.GetFakeCentral().GetAllMessages(),
				storage.PermissionLevel_ELEVATED_IN_NAMESPACE,
			)
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
			// Test context already takes care of creating and destroying resources
			time.Sleep(2 * time.Second)
			assertLastDeploymentHasPermissionLevel(
				t,
				testC.GetFakeCentral().GetAllMessages(),
				storage.PermissionLevel_NONE,
			)
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

		// Wait because of re-sync
		time.Sleep(3 * time.Second)

		assertLastDeploymentHasPermissionLevel(
			t,
			testC.GetFakeCentral().GetAllMessages(),
			storage.PermissionLevel_ELEVATED_IN_NAMESPACE,
		)
		testC.GetFakeCentral().ClearReceivedBuffer()

		utils.IgnoreError(deleteRole)
		utils.IgnoreError(deleteRoleBinding)

		// Wait because of re-sync
		time.Sleep(3 * time.Second)

		assertLastDeploymentHasPermissionLevel(
			t,
			testC.GetFakeCentral().GetAllMessages(),
			storage.PermissionLevel_NONE,
		)
		testC.GetFakeCentral().ClearReceivedBuffer()
	})
}
