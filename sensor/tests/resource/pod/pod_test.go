package pod

import (
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/tests/resource"
	"github.com/stretchr/testify/suite"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
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

func sortAlphabetically(list []string) {
	sort.Slice(list, func(a, b int) bool {
		return list[a] > list[b]
	})
}

func assertDeploymentContainerImages(images ...string) resource.AssertFunc {
	return func(deployment *storage.Deployment) error {
		if len(deployment.GetContainers()) != len(images) {
			return errors.Errorf("number of containers does not match slice of images provided: %d != %d", len(deployment.GetContainers()), len(images))
		}
		containerImages := []string{}
		for _, container := range deployment.GetContainers() {
			containerImages = append(containerImages, container.GetImage().GetName().GetFullName())
		}

		sortAlphabetically(containerImages)
		sortAlphabetically(images)

		if !cmp.Equal(containerImages, images) {
			return errors.Errorf("container images don't match: %s", cmp.Diff(containerImages, images))
		}
		return nil
	}
}

func (s *PodHierarchySuite) Test_ContainerSpecOnDeployment() {
	s.testContext.RunWithResources([]resource.YamlTestFile{
		NginxDeployment,
	}, func(t *testing.T, testC *resource.TestContext, objects map[string]k8s.Object) {
		// wait until pods are created
		err := wait.For(resource.WaitForResourceMatchFunc(testC.Resources())(objects[NginxDeployment.File], func(object k8s.Object) bool {
			d := object.(*appsv1.Deployment)
			return d.Status.AvailableReplicas == 3 && d.Status.ReadyReplicas == 3
		}), wait.WithTimeout(time.Second*10))

		s.Require().NoError(err)

		testC.LastDeploymentState("nginx-deployment",
			assertDeploymentContainerImages("docker.io/library/nginx:1.14.2"),
			"nginx deployment should have a single container with nginx:1.14.2 image")

		messages := testC.GetFakeCentral().GetAllMessages()
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
		err := wait.For(resource.WaitForResourceMatchFunc(testC.Resources())(objects[NginxDeployment.File], func(object k8s.Object) bool {
			d := object.(*appsv1.Deployment)
			return d.Status.AvailableReplicas == 3 && d.Status.ReadyReplicas == 3
		}), wait.WithTimeout(time.Second*10))

		s.Require().NoError(err)

		testC.LastDeploymentState("nginx-rogue",
			assertDeploymentContainerImages("docker.io/library/nginx:1.14.1"),
			"nginx standalone pod should have a single container with nginx:1.14.1 image")

		messages := testC.GetFakeCentral().GetAllMessages()
		uniqueDeployments := resource.GetUniqueDeploymentNames(messages, "sensor-integration")
		s.Contains(uniqueDeployments, "nginx-deployment",
			"Should have receiving at least one deployment with nginx-deployment name")
		s.Contains(uniqueDeployments, "nginx-rogue",
			"Should have receiving at least one deployment with nginx-rogue name")

		uniquePodNames := resource.GetUniquePodNamesFromPrefix(messages, "sensor-integration", "nginx-")
		s.Require().Len(uniquePodNames, 4,
			"Should have received four different pod events (3 from nginx-deployment and 1 from nginx-rouge")
	})
}
