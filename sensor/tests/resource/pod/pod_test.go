package pod

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/tests/helper"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
)

var (
	NginxDeployment = helper.K8sResourceInfo{Kind: "Deployment", YamlFile: "nginx.yaml", Name: "nginx-deployment"}
	NginxPod        = helper.K8sResourceInfo{Kind: "Pod", YamlFile: "nginx-pod.yaml", Name: "nginx-rogue"}
)

type PodHierarchySuite struct {
	testContext *helper.TestContext
	suite.Suite
}

func Test_PodHierarchy(t *testing.T) {
	suite.Run(t, new(PodHierarchySuite))
}

var _ suite.SetupAllSuite = &PodHierarchySuite{}
var _ suite.TearDownTestSuite = &PodHierarchySuite{}

func (s *PodHierarchySuite) SetupSuite() {
	if testContext, err := helper.NewContext(s.T()); err != nil {
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

func assertDeploymentContainerImages(images ...string) helper.AssertFunc {
	return func(deployment *storage.Deployment, _ central.ResourceAction) error {
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
	s.testContext.RunTest(s.T(),
		helper.WithResources([]helper.K8sResourceInfo{
			NginxDeployment,
		}),
		helper.WithTestCase(func(t *testing.T, testC *helper.TestContext, objects map[string]k8s.Object) {
			// wait until pods are created
			err := wait.For(conditions.New(testC.Resources()).ResourceMatch(objects[NginxDeployment.Name], func(object k8s.Object) bool {
				d := object.(*appsv1.Deployment)
				return d.Status.AvailableReplicas == 3 && d.Status.ReadyReplicas == 3
			}), wait.WithTimeout(time.Second*10))

			s.Require().NoError(err)

			testC.LastDeploymentState(t, "nginx-deployment",
				assertDeploymentContainerImages("docker.io/library/nginx:1.14.2"),
				"nginx deployment should have a single container with nginx:1.14.2 image")

			messages := testC.GetFakeCentral().GetAllMessages()
			uniquePodNames := helper.GetUniquePodNamesFromPrefix(messages, "sensor-integration", "nginx-")
			s.Require().Lenf(uniquePodNames, 3, "Should have received three different pod events: %v", uniquePodNames)
		}),
	)
}

func (s *PodHierarchySuite) Test_ParentlessPodsAreTreatedAsDeployments() {
	s.testContext.RunTest(s.T(),
		helper.WithResources([]helper.K8sResourceInfo{
			NginxDeployment,
			NginxPod,
		}),
		helper.WithTestCase(func(t *testing.T, testC *helper.TestContext, objects map[string]k8s.Object) {
			// wait until pods are created
			err := wait.For(conditions.New(testC.Resources()).ResourceMatch(objects[NginxDeployment.Name], func(object k8s.Object) bool {
				d := object.(*appsv1.Deployment)
				return d.Status.AvailableReplicas == 3 && d.Status.ReadyReplicas == 3
			}), wait.WithTimeout(time.Second*10))

			s.Require().NoError(err)

			testC.LastDeploymentState(t, "nginx-rogue",
				assertDeploymentContainerImages("docker.io/library/nginx:1.14.1"),
				"nginx standalone pod should have a single container with nginx:1.14.1 image")

			messages := testC.GetFakeCentral().GetAllMessages()
			uniqueDeployments := helper.GetUniqueDeploymentNames(messages, "sensor-integration")
			s.Contains(uniqueDeployments, "nginx-deployment",
				"Should have receiving at least one deployment with nginx-deployment name")
			s.Contains(uniqueDeployments, "nginx-rogue",
				"Should have receiving at least one deployment with nginx-rogue name")

			uniquePodNames := helper.GetUniquePodNamesFromPrefix(messages, "sensor-integration", "nginx-")
			s.Require().Len(uniquePodNames, 4,
				"Should have received four different pod events (3 from nginx-deployment and 1 from nginx-rouge")
		}),
	)
}

func (s *PodHierarchySuite) Test_DeleteDeployment() {
	s.testContext.RunTest(s.T(), helper.WithTestCase(func(t *testing.T, testC *helper.TestContext, _ map[string]k8s.Object) {
		var id string
		k8sDeployment := &appsv1.Deployment{}
		deleteDep, err := testC.ApplyResourceAndWait(context.Background(), t, helper.DefaultNamespace, &NginxDeployment, k8sDeployment, nil)
		require.NoError(t, err)
		id = string(k8sDeployment.GetUID())
		// Check the deployment is processed
		testC.WaitForDeploymentEvent(t, "nginx-deployment")
		testC.GetFakeCentral().ClearReceivedBuffer()

		// Delete the deployment
		require.NoError(t, deleteDep())

		// Check deployment and action
		testC.LastDeploymentStateWithID(t, id, func(_ *storage.Deployment, action central.ResourceAction) error {
			if action != central.ResourceAction_REMOVE_RESOURCE {
				return errors.New("ResourceAction should be REMOVE_RESOURCE")
			}
			return nil
		}, "deployment should be deleted", time.Minute)
		testC.LastViolationStateByIDWithTimeout(t, id, func(alertResults *central.AlertResults) error {
			if alertResults.GetAlerts() != nil && len(alertResults.GetAlerts()) > 0 {
				var alertNames []string
				for _, a := range alertResults.GetAlerts() {
					alertNames = append(alertNames, a.GetPolicy().GetName())
				}
				t.Logf("AlertResults are not empty: %v", alertNames)
				return errors.New("AlertResults should be empty")
			}
			return nil
		}, "Should have an empty violation", true, time.Minute)
		testC.GetFakeCentral().ClearReceivedBuffer()
	}))
}

func (s *PodHierarchySuite) Test_DeletePod() {
	s.testContext.RunTest(s.T(), helper.WithTestCase(func(t *testing.T, testC *helper.TestContext, _ map[string]k8s.Object) {
		var id string
		k8sPod := &v1.Pod{}
		deletePod, err := testC.ApplyResourceAndWait(context.Background(), t, helper.DefaultNamespace, &NginxPod, k8sPod, nil)
		require.NoError(t, err)
		id = string(k8sPod.GetUID())
		// Check the pod is processed
		testC.WaitForDeploymentEvent(t, "nginx-rogue")
		testC.GetFakeCentral().ClearReceivedBuffer()

		// Delete the pod
		require.NoError(t, deletePod())

		// Check pod and action
		testC.LastDeploymentStateWithTimeout(t, "nginx-rogue", func(_ *storage.Deployment, action central.ResourceAction) error {
			if action != central.ResourceAction_REMOVE_RESOURCE {
				return errors.New("ResourceAction should be REMOVE_RESOURCE")
			}
			return nil
		}, "rogue pod should be deleted", 5*time.Minute)
		testC.LastViolationStateByIDWithTimeout(t, id, func(alertResults *central.AlertResults) error {
			if alertResults.GetAlerts() != nil {
				return errors.New("AlertResults should be empty")
			}
			return nil
		}, "Should have an empty violation", true, 5*time.Minute)
		testC.GetFakeCentral().ClearReceivedBuffer()
	}))
}
