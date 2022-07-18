package pod

import (
	"testing"
	"time"

	"github.com/stackrox/rox/sensor/tests/resource"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
)

var (
	NginxDeployment = resource.YamlTestFile{Kind: "Deployment", File: "nginx.yaml"}
	NginxPod        = resource.YamlTestFile{Kind: "Pod", File: "nginx-pod.yaml"}
)

type PodHierarchySuite struct {
	testContext *resource.TestContext
	suite.Suite
}

func Test_PodHierarchy(t *testing.T) {
	suite.Run(t, new(PodHierarchySuite))
}

var _ suite.SetupAllSuite = &PodHierarchySuite{}
var _ suite.TearDownTestSuite = &PodHierarchySuite{}

func (s *PodHierarchySuite) SetupSuite() {
	if testContext, err := resource.NewContext(s.T()); err != nil {
		s.Fail("failed to setup test context: %s", err)
	} else {
		s.testContext = testContext
	}
}

func (s *PodHierarchySuite) TearDownTest() {
	// Clear any messages received in fake central during the test run
	s.testContext.GetFakeCentral().ClearReceivedBuffer()
}

func (s *PodHierarchySuite) Test_ContainerSpecOnDeployment() {
	s.testContext.RunWithResources([]resource.YamlTestFile{
		NginxDeployment,
	}, func(t *testing.T, testC *resource.TestContext, objects map[string]k8s.Object) {
		// wait until pods are created
		err := wait.For(conditions.New(testC.Resources()).ResourceMatch(objects[NginxDeployment.File], func(object k8s.Object) bool {
			d := object.(*appsv1.Deployment)
			return d.Status.AvailableReplicas == 3 && d.Status.ReadyReplicas == 3
		}), wait.WithTimeout(time.Second*10))

		s.Require().NoError(err)

		time.Sleep(1 * time.Second)
		messages := testC.GetFakeCentral().GetAllMessages()
		deployment := resource.GetLastMessageWithDeploymentName(messages, "sensor-integration", "nginx-deployment").
			GetEvent().
			GetDeployment()
		require.NotNil(t, deployment, "Deployment object can't be nil")

		s.Require().Len(deployment.GetContainers(), 1, "Should have 1 container in deployment object")
		container := deployment.GetContainers()[0]
		s.Equal("docker.io/library/nginx:1.14.2", container.GetImage().GetName().GetFullName(),
			"Should have correct image name")

		uniquePodNames := resource.GetUniquePodNamesFromPrefix(messages, "sensor-integration", "nginx-")
		s.Require().Len(uniquePodNames, 3, "Should have received three different pod events")
	})
}

func (s *PodHierarchySuite) Test_ParentlessPodsAreTreatedAsDeployments() {
	s.testContext.RunWithResources([]resource.YamlTestFile{
		NginxDeployment,
		NginxPod,
	}, func(t *testing.T, testC *resource.TestContext, objects map[string]k8s.Object) {
		// wait until pods are created
		err := wait.For(conditions.New(testC.Resources()).ResourceMatch(objects[NginxDeployment.File], func(object k8s.Object) bool {
			d := object.(*appsv1.Deployment)
			return d.Status.AvailableReplicas == 3 && d.Status.ReadyReplicas == 3
		}), wait.WithTimeout(time.Second*10))

		s.Require().NoError(err)

		time.Sleep(1 * time.Second)
		messages := testC.GetFakeCentral().GetAllMessages()
		uniqueDeployments := resource.GetUniqueDeploymentNames(messages, "sensor-integration")
		s.Contains(uniqueDeployments, "nginx-deployment",
			"Should have receiving at least one deployment with nginx-deployment name")
		s.Contains(uniqueDeployments, "nginx-rogue",
			"Should have receiving at least one deployment with nginx-rogue name")

		deployment := resource.GetLastMessageWithDeploymentName(messages, "sensor-integration", "nginx-rogue").
			GetEvent().
			GetDeployment()
		require.NotNil(t, deployment)
		s.Len(deployment.GetContainers(), 1)
		container := deployment.GetContainers()[0]
		s.Equal("docker.io/library/nginx:1.14.1", container.GetImage().GetName().GetFullName(),
			"Should have correct image name")

		uniquePodNames := resource.GetUniquePodNamesFromPrefix(messages, "sensor-integration", "nginx-")
		s.Require().Len(uniquePodNames, 4,
			"Should have received four different pod events (3 from nginx-deployment and 1 from nginx-rouge")
	})
}
