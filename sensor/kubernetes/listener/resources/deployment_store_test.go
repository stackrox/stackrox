package resources

import (
	"sort"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/registry"
	"github.com/stackrox/rox/sensor/common/service"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/kubernetes/orchestratornamespaces"
	"github.com/stackrox/rox/sensor/kubernetes/selector"
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

func (s *deploymentStoreSuite) Test_FindDeploymentIDsWithServiceAccount() {
	deployments := []*v1.Deployment{
		withServiceAccount(makeDeploymentObject("d1", "ns1", "uuid1"), "sa1"),
		withServiceAccount(makeDeploymentObject("d2", "ns1", "uuid2"), "sa1"),
		withServiceAccount(makeDeploymentObject("d3", "ns1", "uuid3"), "sa2"),
		withServiceAccount(makeDeploymentObject("d4", "ns2", "uuid4"), "sa1"),
		withServiceAccount(makeDeploymentObject("d5", "ns2", "uuid5"), "sa3"),
	}

	testCases := map[string]struct {
		queryNs, querySa string
		expectedIDs      []string
	}{
		"Two deployments with same SA in ns1": {
			queryNs:     "ns1",
			querySa:     "sa1",
			expectedIDs: []string{"uuid1", "uuid2"},
		},
		"One deployment with SA sa2 in ns1": {
			queryNs:     "ns1",
			querySa:     "sa2",
			expectedIDs: []string{"uuid3"},
		},
		"One deployment with SA sa1 in ns2": {
			queryNs:     "ns2",
			querySa:     "sa1",
			expectedIDs: []string{"uuid4"},
		},
		"One deployment with SA sa3 in ns2": {
			queryNs:     "ns2",
			querySa:     "sa3",
			expectedIDs: []string{"uuid5"},
		},
		"No deployments for valid SA and empty namespace": {
			queryNs:     "",
			querySa:     "sa1",
			expectedIDs: nil,
		},
		"No deployment for valid namespace and empty ServiceAccount": {
			queryNs:     "ns1",
			querySa:     "",
			expectedIDs: nil,
		},
	}

	for _, deployment := range deployments {
		s.deploymentStore.addOrUpdateDeployment(s.createDeploymentWrap(deployment))
	}

	for name, testCase := range testCases {
		s.Run(name, func() {

			ids := s.deploymentStore.FindDeploymentIDsWithServiceAccount(testCase.queryNs, testCase.querySa)
			s.Require().Len(ids, len(testCase.expectedIDs), "FindDeploymentIDsWithServiceAccount returned incorrect number of elements")
			sort.Strings(testCase.expectedIDs)
			sort.Strings(ids)
			s.Equal(testCase.expectedIDs, ids)
		})
	}
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

func withLabels(deployment *v1.Deployment, labels map[string]string) *v1.Deployment {
	deployment.Spec.Template.Labels = labels
	return deployment
}

func (s *deploymentStoreSuite) Test_FindDeploymentIDsByLabels() {
	deployments := []*v1.Deployment{
		withLabels(makeDeploymentObject("d-1", "test-ns", "uuid-1"), map[string]string{}),
		withLabels(makeDeploymentObject("d-2", "test-ns", "uuid-2"), map[string]string{
			"app": "nginx",
		}),
		withLabels(makeDeploymentObject("d-3", "test-ns", "uuid-3"), map[string]string{
			"no": "match",
		}),
		withLabels(makeDeploymentObject("d-4", "test-ns-no-match", "uuid-4"), map[string]string{
			"app": "nginx",
		}),
		withLabels(makeDeploymentObject("d-5", "test-ns", "uuid-5"), map[string]string{
			"app":  "nginx-2",
			"role": "backend",
		}),
	}
	for _, d := range deployments {
		s.deploymentStore.addOrUpdateDeployment(s.createDeploymentWrap(d))
	}
	cases := map[string]struct {
		namespace   string
		labels      map[string]string
		expectedIDs []string
	}{
		"No labels": {
			namespace:   "test-ns",
			labels:      nil,
			expectedIDs: nil,
		},
		"Match": {
			namespace: "test-ns",
			labels: map[string]string{
				"app": "nginx",
			},
			expectedIDs: []string{"uuid-2"},
		},
		"Labels do not match": {
			namespace: "test-ns",
			labels: map[string]string{
				"app": "nginx",
			},
			expectedIDs: []string{"uuid-2"},
		},
		"Namespace do not match": {
			namespace: "test-ns",
			labels: map[string]string{
				"app": "nginx",
			},
			expectedIDs: []string{"uuid-2"},
		},
		"Deployment with two labels vs a subset Selector": {
			namespace: "test-ns",
			labels: map[string]string{
				"app": "nginx-2",
			},
			expectedIDs: []string{"uuid-5"},
		},
		"Deployment with two labels vs a superset Selector": {
			namespace: "test-ns",
			labels: map[string]string{
				"app":  "nginx-2",
				"role": "backend",
				"l3":   "val3",
			},
			expectedIDs: nil,
		},
	}
	for testName, c := range cases {
		s.Run(testName, func() {
			ids := s.deploymentStore.FindDeploymentIDsByLabels(c.namespace, selector.CreateSelector(c.labels))
			s.Equal(len(c.expectedIDs), len(ids))
			sort.Strings(ids)
			sort.Strings(c.expectedIDs)
			s.Equal(c.expectedIDs, ids)
		})
	}
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

func withServiceAccount(d *v1.Deployment, name string) *v1.Deployment {
	d.Spec.Template.Spec.ServiceAccountName = name
	return d
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
