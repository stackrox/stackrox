package resources

import (
	"fmt"
	"sort"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/deduper"
	"github.com/stackrox/rox/sensor/common/registry"
	"github.com/stackrox/rox/sensor/common/selector"
	"github.com/stackrox/rox/sensor/common/service"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/kubernetes/orchestratornamespaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		"deployment", "", s.mockPodLister, s.namespaceStore, hierarchyFromPodLister(s.mockPodLister), "", orchestratornamespaces.NewOrchestratorNamespaces(), registry.NewRegistryStore(nil))
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
	uid := uuid.NewV4().String()
	wrap := s.createDeploymentWrap(makeDeploymentObject("test-deployment", "test-ns", types.UID(uid)))
	s.deploymentStore.addOrUpdateDeployment(wrap)

	expectedExposureInfo := storage.PortConfig_ExposureInfo{
		Level:       storage.PortConfig_EXTERNAL,
		ServiceName: "test.service",
		ServicePort: 5432,
	}

	_, isBuilt := s.deploymentStore.GetBuiltDeployment(uid)
	s.Assert().False(isBuilt, "deployment should not be fully built yet")

	deployment, err := s.deploymentStore.BuildDeploymentWithDependencies(uid, store.Dependencies{
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

	_, isBuilt = s.deploymentStore.GetBuiltDeployment(uid)
	s.Assert().True(isBuilt, "deployment should be fully built")
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
				"app": "no-match",
			},
			expectedIDs: nil,
		},
		"Namespaces do not match": {
			namespace: "ns-no-match",
			labels: map[string]string{
				"app": "nginx",
			},
			expectedIDs: nil,
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
			s.ElementsMatch(c.expectedIDs, ids)
		})
	}
}

func withImage(deployment *v1.Deployment, image string) *v1.Deployment {
	deployment.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Image: image,
		},
	}
	return deployment
}

func newImage(id string, fullName string) *storage.Image {
	return &storage.Image{
		Id: id,
		Name: &storage.ImageName{
			FullName: fullName,
		},
	}
}

func (s *deploymentStoreSuite) Test_FindDeploymentIDsByImages() {
	resources := []struct {
		deployment *v1.Deployment
		imageID    string
	}{
		{
			deployment: withImage(makeDeploymentObject("d-1", "test-ns", "uuid-1"), "nginx:1.2.3"),
			imageID:    "image-uuid-1",
		},
		{
			deployment: withImage(makeDeploymentObject("d-2", "test-ns", "uuid-2"), "private-registry.io/nginx:1.2.3"),
			imageID:    "image-uuid-1",
		},
		{
			deployment: withImage(makeDeploymentObject("d-3", "test-ns", "uuid-3"), "private-registry.io/main:3.2.1"),
			imageID:    "image-uuid-2",
		},
	}
	for _, r := range resources {
		wrap := s.createDeploymentWrap(r.deployment)
		// Manually set the ID for testing purposes
		for i := range wrap.GetDeployment().GetContainers() {
			wrap.GetDeployment().GetContainers()[i].GetImage().Id = r.imageID
		}
		s.deploymentStore.addOrUpdateDeployment(wrap)
	}
	cases := map[string]struct {
		images      []*storage.Image
		expectedIDs []string
	}{
		"No images": {
			images:      nil,
			expectedIDs: nil,
		},
		"Match one deployment against an image": {
			images: []*storage.Image{
				newImage("", "docker.io/library/nginx:1.2.3"),
			},
			expectedIDs: []string{"uuid-1"},
		},
		"Match multiple deployment against multiple images": {
			images: []*storage.Image{
				newImage("", "docker.io/library/nginx:1.2.3"),
				newImage("", "private-registry.io/nginx:1.2.3"),
			},
			expectedIDs: []string{"uuid-1", "uuid-2"},
		},
		"Match multiple deployments against one image id": {
			images: []*storage.Image{
				newImage("image-uuid-1", ""),
			},
			expectedIDs: []string{"uuid-1", "uuid-2"},
		},
		"Match multiple deployments against multiple image ids": {
			images: []*storage.Image{
				newImage("image-uuid-1", ""),
				newImage("image-uuid-2", ""),
			},
			expectedIDs: []string{"uuid-1", "uuid-2", "uuid-3"},
		},
		"No match": {
			images: []*storage.Image{
				newImage("", "no-match"),
			},
			expectedIDs: []string{},
		},
		"No match by id": {
			images: []*storage.Image{
				newImage("no-match", ""),
			},
			expectedIDs: []string{},
		},
		"Match one deployment against multiple images": {
			images: []*storage.Image{
				newImage("", "no-match"),
				newImage("", "private-registry.io/nginx:1.2.3"),
			},
			expectedIDs: []string{"uuid-2"},
		},
		"Match multiple deployments against a valid image id and a no-match": {
			images: []*storage.Image{
				newImage("no-match", ""),
				newImage("image-uuid-1", ""),
			},
			expectedIDs: []string{"uuid-1", "uuid-2"},
		},
		"Match against mixed images": {
			images: []*storage.Image{
				newImage("", "docker.io/library/nginx:1.2.3"),
				newImage("image-uuid-2", ""),
			},
			expectedIDs: []string{"uuid-1", "uuid-3"},
		},
		"Match against same image id with different paths": {
			images: []*storage.Image{
				newImage("image-uuid-1", "docker.io/library/nginx:1.2.3"),
			},
			expectedIDs: []string{"uuid-1", "uuid-2"},
		},
	}
	for testName, c := range cases {
		s.Run(testName, func() {
			ids := s.deploymentStore.FindDeploymentIDsByImages(c.images)
			s.Equal(len(c.expectedIDs), len(ids))
			s.ElementsMatch(c.expectedIDs, ids)
		})
	}
}

func (s *deploymentStoreSuite) Test_DeleteAllDeployments() {
	testCases := []struct {
		before []*v1.Deployment
		after  []*v1.Deployment
	}{
		{
			before: []*v1.Deployment{
				makeDeploymentObject("before1", "test-ns", "uuid-1"),
			},
		},
		{
			after: []*v1.Deployment{
				makeDeploymentObject("after1", "test-ns", "uuid-2"),
			},
		},
		{
			before: []*v1.Deployment{
				makeDeploymentObject("before1", "test-ns", "uuid-1"),
				makeDeploymentObject("before2", "test-ns", "uuid-2"),
			},
			after: []*v1.Deployment{
				makeDeploymentObject("after1", "test-ns", "uuid-3"),
			},
		},
		{
			before: []*v1.Deployment{
				makeDeploymentObject("same", "test-ns", "uuid-1"),
			},
			after: []*v1.Deployment{
				makeDeploymentObject("same", "test-ns", "uuid-1"),
			},
		},
		{
			before: []*v1.Deployment{
				makeDeploymentObject("before1", "old-ns", "uuid-1"),
			},
			after: []*v1.Deployment{
				makeDeploymentObject("after1", "new-ns", "uuid-2"),
			},
		},
	}

	for _, testCase := range testCases {
		s.Run(fmt.Sprintf("Create %d before %d after", len(testCase.before), len(testCase.after)), func() {
			s.namespaceStore = newNamespaceStore()
			s.namespaceStore.addNamespace(&storage.NamespaceMetadata{Name: "test-ns", Id: "1"})
			s.deploymentStore = newDeploymentStore()
			s.mockPodLister = &mockPodLister{}

			for _, before := range testCase.before {
				s.deploymentStore.addOrUpdateDeployment(s.createDeploymentWrap(before))
			}

			s.deploymentStore.Cleanup()

			for _, before := range testCase.before {
				s.Assert().Nil(s.deploymentStore.Get(string(before.GetUID())))
			}

			s.Assert().Equal(0, s.deploymentStore.CountDeploymentsForNamespace("test-ns"))
			s.Assert().Equal(0, s.deploymentStore.CountDeploymentsForNamespace("old-ns"))
			s.Assert().Equal(0, s.deploymentStore.CountDeploymentsForNamespace("new-ns"))

			for _, after := range testCase.after {
				s.deploymentStore.addOrUpdateDeployment(s.createDeploymentWrap(after))
			}

			for _, after := range testCase.after {
				s.Assert().NotNil(s.deploymentStore.Get(string(after.GetUID())))
			}

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

func TestDeploymentStore_ReconcileDelete(t *testing.T) {
	const (
		existingDeployID1 = "deployment-1"
		missingDeployID1  = "not-existing-deployment"
	)
	depl1 := &deploymentWrap{
		Deployment: &storage.Deployment{
			Id:                            existingDeployID1,
			Name:                          "depl1",
			Hash:                          0,
			Type:                          "",
			Namespace:                     "ns",
			NamespaceId:                   "",
			OrchestratorComponent:         false,
			Replicas:                      0,
			Labels:                        nil,
			PodLabels:                     nil,
			LabelSelector:                 nil,
			Created:                       nil,
			ClusterId:                     "cluster1",
			ClusterName:                   "",
			Containers:                    nil,
			Annotations:                   nil,
			Priority:                      0,
			Inactive:                      false,
			ImagePullSecrets:              nil,
			ServiceAccount:                "",
			ServiceAccountPermissionLevel: 0,
			AutomountServiceAccountToken:  false,
			HostNetwork:                   false,
			HostPid:                       false,
			HostIpc:                       false,
			RuntimeClass:                  "",
			Tolerations:                   nil,
			Ports:                         nil,
			StateTimestamp:                0,
			RiskScore:                     0,
		},
		registryOverride: "",
		original:         nil,
		portConfigs:      nil,
		pods:             nil,
		registryStore:    nil,
		isBuilt:          false,
		mutex:            sync.RWMutex{},
	}
	require.NoError(t, depl1.updateHash(), "failed updating the hash for test deployment")

	storage1 := make(map[string]*deploymentWrap)
	storage1[existingDeployID1] = depl1
	t.Logf("Deployment hash: %d", depl1.GetHash())

	type args struct {
		resType string
		resID   string
		resHash uint64
	}
	tests := map[string]struct {
		storage map[string]*deploymentWrap
		args    args
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		"Exisiting deployment with matching hash should yield no reconciliation events": {
			storage: storage1,
			args: args{
				resType: deduper.TypeDeployment.String(),
				resID:   existingDeployID1,
				resHash: depl1.GetHash(),
			},
			want: "",
			wantErr: func(t assert.TestingT, err error, args ...interface{}) bool {
				return assert.Nil(t, err, args)
			},
		},
		"Deployment that cannot be found should be deleted from Central": {
			storage: storage1,
			args: args{
				resType: deduper.TypeDeployment.String(),
				resID:   missingDeployID1,
				resHash: depl1.GetHash(),
			},
			want: missingDeployID1,
			wantErr: func(t assert.TestingT, err error, args ...interface{}) bool {
				return assert.Nil(t, err, args)
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ds := &DeploymentStore{
				lock:        sync.RWMutex{},
				deployments: tt.storage,
			}
			got, err := ds.ReconcileDelete(tt.args.resType, tt.args.resID, tt.args.resHash)
			if !tt.wantErr(t, err, fmt.Sprintf("ReconcileDelete(%v, %v, %v)", tt.args.resType, tt.args.resID, tt.args.resHash)) {
				return
			}
			assert.Equalf(t, tt.want, got, "ReconcileDelete(%v, %v, %v)", tt.args.resType, tt.args.resID, tt.args.resHash)
		})
	}
}
