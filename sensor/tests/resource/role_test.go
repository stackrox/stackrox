package resource

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	centralDebug "github.com/stackrox/rox/sensor/debugger/central"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getLastMessageWithDeploymentName(messages []*central.MsgFromSensor, n string) *central.MsgFromSensor {
	var lastMessage *central.MsgFromSensor
	for i := len(messages) - 1; i > 0; i-- {
		if messages[i].GetEvent().GetDeployment().GetName() == n {
			lastMessage = messages[i]
			break
		}
	}
	return lastMessage
}

func Test_RoleDeploymentDependency(t *testing.T) {
	testContext, err := NewContext(t)
	if err != nil {
		t.Fatalf("failed to create test context: %s", err)
	}

	testContext.RunPermutationTest(
		[]yamlTestFile{
			Nginx,
			NginxRole,
			NginxRoleBinding,
		}, "Role Dependency", func(t *testing.T, fakeCentral *centralDebug.FakeService) {
			// Test context already takes care of creating and destroying resources
			time.Sleep(2 * time.Second)
			lastNginxDeploymentUpdate := getLastMessageWithDeploymentName(fakeCentral.GetAllMessages(), "nginx-deployment")
			require.NotNil(t, lastNginxDeploymentUpdate, "should have found a message for nginx-deployment")
			deployment := lastNginxDeploymentUpdate.GetEvent().GetDeployment()
			assert.Equal(
				t,
				deployment.ServiceAccountPermissionLevel,
				storage.PermissionLevel_ELEVATED_IN_NAMESPACE,
				"permission level has to be ELEVATED_IN_NAMESPACE",
			)
		},
	)

	testContext.RunTest(
		[]yamlTestFile{
			Nginx,
			NginxRole,
		}, "Permission level is set to None if no binding is found", func(t *testing.T, fakeCentral *centralDebug.FakeService) {
			// Test context already takes care of creating and destroying resources
			time.Sleep(2 * time.Second)
			lastNginxDeploymentUpdate := getLastMessageWithDeploymentName(fakeCentral.GetAllMessages(), "nginx-deployment")
			require.NotNil(t, lastNginxDeploymentUpdate, "should have found a message for nginx-deployment")
			deployment := lastNginxDeploymentUpdate.GetEvent().GetDeployment()
			assert.Equal(
				t,
				deployment.ServiceAccountPermissionLevel,
				storage.PermissionLevel_NONE,
				"permission level has to be NONE",
			)
		})
}
