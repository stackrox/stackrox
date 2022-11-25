package resources

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/registry"
	"github.com/stackrox/rox/sensor/common/service"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/kubernetes/orchestratornamespaces"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type deploymentStoreSuite struct {
	suite.Suite
	deploymentStore *DeploymentStore
	namespaceStore  *namespaceStore
	mockPodLister   *mockPodLister
}

func TestDeploymentStoreSuite(t *testing.T) {
	suite.Run(t, new(deploymentStoreSuite))
}

var _ suite.SetupTestSuite = &deploymentStoreSuite{}

func (s *deploymentStoreSuite) SetupTest() {
	s.namespaceStore = newNamespaceStore()
	s.namespaceStore.addNamespace(&storage.NamespaceMetadata{Name: "test-ns", Id: "1"})
	s.deploymentStore = newDeploymentStore()
	s.mockPodLister = &mockPodLister{}
}

func (s *deploymentStoreSuite) createDeploymentWrap(deploymentObj interface{}) *deploymentWrap {
	action := central.ResourceAction_CREATE_RESOURCE
	wrap := newDeploymentEventFromResource(deploymentObj, &action,
		"deployment", "", s.mockPodLister, s.namespaceStore, hierarchyFromPodLister(s.mockPodLister), "", orchestratornamespaces.Singleton(), registry.Singleton())
	return wrap
}

func (s *deploymentStoreSuite) Test_BuildDeploymentWithDependencies() {
	uid := uuid.NewV4()
	wrap := s.createDeploymentWrap(makeDeploymentObject("test-deployment", "test-ns", types.UID(uid.String())))
	s.deploymentStore.addOrUpdateDeployment(wrap)

	expectedExposureInfo := storage.PortConfig_ExposureInfo{
		Level:       storage.PortConfig_EXTERNAL,
		ServiceName: "test.service",
		ServicePort: 5432,
	}

	deployment, err := s.deploymentStore.BuildDeploymentWithDependencies(uid.String(), store.Dependencies{
		PermissionLevel: storage.PermissionLevel_CLUSTER_ADMIN,
		Exposures: []map[service.PortRef][]*storage.PortConfig_ExposureInfo{
			{
				service.PortRefOf(stubService()): []*storage.PortConfig_ExposureInfo{&expectedExposureInfo},
			},
		},
	})

	s.NoError(err, "should not have error building dependencies")

	s.Require().Len(deployment.GetPorts(), 1)
	s.Require().Len(deployment.GetPorts()[0].GetExposureInfos(), 1)

	s.Equal(expectedExposureInfo, *deployment.GetPorts()[0].GetExposureInfos()[0])
	s.Equal(storage.PermissionLevel_CLUSTER_ADMIN, deployment.GetServiceAccountPermissionLevel(), "Service account permission level")
}

func (s *deploymentStoreSuite) Test_BuildDeploymentWithDependencies_NoDeployment() {
	_, err := s.deploymentStore.BuildDeploymentWithDependencies("some-uuid", store.Dependencies{
		PermissionLevel: storage.PermissionLevel_CLUSTER_ADMIN,
		Exposures:       []map[service.PortRef][]*storage.PortConfig_ExposureInfo{},
	})

	s.ErrorContains(err, "some-uuid doesn't exist")
}

func makeDeploymentObject(name, namespace string, id types.UID) *v1.Deployment {
	return &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       id,
		},
	}
}

func stubService() corev1.ServicePort {
	return corev1.ServicePort{
		Name:        "test.service",
		Protocol:    "TCP",
		AppProtocol: nil,
		Port:        5432,
		TargetPort: intstr.IntOrString{
			IntVal: 4321,
		},
		NodePort: 0,
	}
}
