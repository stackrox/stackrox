package resources

import (
	"fmt"
	"sort"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/selector"
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
		"deployment", "", s.mockPodLister, s.namespaceStore, hierarchyFromPodLister(s.mockPodLister), "", orchestratornamespaces.NewOrchestratorNamespaces())
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

func (s *deploymentStoreSuite) Test_BuildDeployments_CachedDependencies() {
	defaultExposure := []map[service.PortRef][]*storage.PortConfig_ExposureInfo{
		{
			service.PortRefOf(stubService()): []*storage.PortConfig_ExposureInfo{{
				Level:       storage.PortConfig_EXTERNAL,
				ServiceName: "test.service",
				ServicePort: 5432,
			}},
		},
		{
			service.PortRefOf(stubService()): []*storage.PortConfig_ExposureInfo{{
				Level:       storage.PortConfig_HOST,
				ServiceName: "test2.service",
				ServicePort: 2345,
				ExternalIps: []string{"a.com", "b.com"},
			}},
		},
	}

	defaultExposureUnordered := []map[service.PortRef][]*storage.PortConfig_ExposureInfo{
		{
			service.PortRefOf(stubService()): []*storage.PortConfig_ExposureInfo{{
				Level:       storage.PortConfig_HOST,
				ServiceName: "test2.service",
				ServicePort: 2345,
				ExternalIps: []string{"b.com", "a.com"},
			}},
		},
		{
			service.PortRefOf(stubService()): []*storage.PortConfig_ExposureInfo{{
				Level:       storage.PortConfig_EXTERNAL,
				ServiceName: "test.service",
				ServicePort: 5432,
			}},
		},
	}

	dependenciesX := store.Dependencies{
		PermissionLevel: storage.PermissionLevel_CLUSTER_ADMIN,
		Exposures:       defaultExposure,
	}

	dependenciesXUnordered := store.Dependencies{
		PermissionLevel: storage.PermissionLevel_CLUSTER_ADMIN,
		Exposures:       defaultExposureUnordered,
	}

	dependenciesY := store.Dependencies{
		PermissionLevel: storage.PermissionLevel_NONE,
		Exposures:       defaultExposure,
	}

	testCases := map[string]struct {
		orderedDependencies        []store.Dependencies
		orderedExpectedPointerSame []bool
	}{
		"No dependencies changed returns cached deployment": {
			orderedDependencies:        []store.Dependencies{dependenciesX, dependenciesX},
			orderedExpectedPointerSame: []bool{true},
		},
		"Dependency changed returns a new deployment object": {
			orderedDependencies:        []store.Dependencies{dependenciesX, dependenciesY},
			orderedExpectedPointerSame: []bool{false},
		},
		"Multiple events with only one dependency change doesn't cause multiple clones": {
			orderedDependencies:        []store.Dependencies{dependenciesX, dependenciesY, dependenciesY, dependenciesY},
			orderedExpectedPointerSame: []bool{false, true, true}, // should build new object once and then return it for subsequent calls
		},
		"Dependencies with mixed ordered in fields should not cause new object to be built": {
			orderedDependencies:        []store.Dependencies{dependenciesX, dependenciesXUnordered},
			orderedExpectedPointerSame: []bool{true},
		},
	}

	for name, testCase := range testCases {
		s.Run(name, func() {
			uid := uuid.NewV4().String()
			wrap := s.createDeploymentWrap(makeDeploymentObject("test-deployment", "test-ns", types.UID(uid)))
			s.deploymentStore.addOrUpdateDeployment(wrap)

			objs := make([]*storage.Deployment, len(testCase.orderedDependencies))
			var err error
			for i := 0; i < len(testCase.orderedDependencies); i++ {
				objs[i], _, err = s.deploymentStore.BuildDeploymentWithDependencies(uid, testCase.orderedDependencies[i])
				s.Require().NoError(err)
			}

			for i, exp := range testCase.orderedExpectedPointerSame {
				if exp {
					s.Assert().Samef(objs[i], objs[i+1], "Comparing objects %d and %d failed", i, i+1)
				} else {
					s.Assert().NotSamef(objs[i], objs[i+1], "Comparing objects %d and %d failed", i, i+1)
				}
			}

			s.deploymentStore.Cleanup()
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

	deployment, _, err := s.deploymentStore.BuildDeploymentWithDependencies(uid, store.Dependencies{
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
	_, _, err := s.deploymentStore.BuildDeploymentWithDependencies("some-uuid", store.Dependencies{
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

var (
	namespaceName            = "test-ns"
	deleteWithReferenceCases = map[string]struct {
		deploymentsToAdd           []*v1.Deployment
		deploymentsToRemove        []*v1.Deployment
		deploymentsToReference     []string
		deploymentsToDereference   []string
		expectedDeletedDeployments []string
		expectedMarkedDeployments  []string
	}{
		"All deleted": {
			deploymentsToAdd: []*v1.Deployment{
				makeDeploymentObject("dep-1", namespaceName, "1"),
				makeDeploymentObject("dep-2", namespaceName, "2"),
			},
			deploymentsToRemove: []*v1.Deployment{
				makeDeploymentObject("dep-1", namespaceName, "1"),
				makeDeploymentObject("dep-2", namespaceName, "2"),
			},
			deploymentsToReference:     []string{"1", "2"},
			deploymentsToDereference:   []string{"1", "2"},
			expectedDeletedDeployments: []string{"1", "2"},
		},
		"All deleted no extra reference": {
			deploymentsToAdd: []*v1.Deployment{
				makeDeploymentObject("dep-1", namespaceName, "1"),
				makeDeploymentObject("dep-2", namespaceName, "2"),
			},
			deploymentsToRemove: []*v1.Deployment{
				makeDeploymentObject("dep-1", namespaceName, "1"),
				makeDeploymentObject("dep-2", namespaceName, "2"),
			},
			expectedDeletedDeployments: []string{"1", "2"},
		},
		"All marked as deleted": {
			deploymentsToAdd: []*v1.Deployment{
				makeDeploymentObject("dep-1", namespaceName, "1"),
				makeDeploymentObject("dep-2", namespaceName, "2"),
			},
			deploymentsToRemove: []*v1.Deployment{
				makeDeploymentObject("dep-1", namespaceName, "1"),
				makeDeploymentObject("dep-2", namespaceName, "2"),
			},
			deploymentsToReference:    []string{"1", "2"},
			expectedMarkedDeployments: []string{"1", "2"},
		},
		"One deleted one marked as deleted": {
			deploymentsToAdd: []*v1.Deployment{
				makeDeploymentObject("dep-1", namespaceName, "1"),
				makeDeploymentObject("dep-2", namespaceName, "2"),
			},
			deploymentsToRemove: []*v1.Deployment{
				makeDeploymentObject("dep-1", namespaceName, "1"),
				makeDeploymentObject("dep-2", namespaceName, "2"),
			},
			deploymentsToReference:     []string{"1"},
			expectedMarkedDeployments:  []string{"1"},
			expectedDeletedDeployments: []string{"2"},
		},
	}
	deleteCases = map[string]struct {
		deploymentsToAdd           []*v1.Deployment
		deploymentsToRemove        []*v1.Deployment
		expectedDeletedDeployments []string
	}{
		"All deleted": {
			deploymentsToAdd: []*v1.Deployment{
				makeDeploymentObject("dep-1", namespaceName, "1"),
				makeDeploymentObject("dep-2", namespaceName, "2"),
			},
			deploymentsToRemove: []*v1.Deployment{
				makeDeploymentObject("dep-1", namespaceName, "1"),
				makeDeploymentObject("dep-2", namespaceName, "2"),
			},
			expectedDeletedDeployments: []string{"1", "2"},
		},
		"One deleted": {
			deploymentsToAdd: []*v1.Deployment{
				makeDeploymentObject("dep-1", namespaceName, "1"),
				makeDeploymentObject("dep-2", namespaceName, "2"),
			},
			deploymentsToRemove: []*v1.Deployment{
				makeDeploymentObject("dep-1", namespaceName, "1"),
			},
			expectedDeletedDeployments: []string{"1"},
		},
	}
)

func (s *deploymentStoreSuite) Test_DeleteDeployments() {
	for name, tc := range deleteCases {
		s.Run(name, func() {
			s.namespaceStore = newNamespaceStore()
			s.namespaceStore.addNamespace(&storage.NamespaceMetadata{Name: namespaceName, Id: "1"})
			s.deploymentStore = newDeploymentStore()

			for _, deploymentToAdd := range tc.deploymentsToAdd {
				s.deploymentStore.addOrUpdateDeployment(s.createDeploymentWrap(deploymentToAdd))
			}

			for _, deploymentToRemove := range tc.deploymentsToRemove {
				s.deploymentStore.removeDeployment(s.createDeploymentWrap(deploymentToRemove))
			}

			for _, expectedDeleted := range tc.expectedDeletedDeployments {
				s.Assert().Nil(s.deploymentStore.Get(expectedDeleted))
			}
		})
	}
}

func (s *deploymentStoreSuite) Test_OnNamespaceDeleted() {
	deployments := []*deploymentWrap{
		s.createDeploymentWrap(makeDeploymentObject("dep-1", namespaceName, "1")),
		s.createDeploymentWrap(makeDeploymentObject("dep-2", namespaceName, "2")),
	}

	s.namespaceStore = newNamespaceStore()
	s.namespaceStore.addNamespace(&storage.NamespaceMetadata{Name: namespaceName, Id: "1"})
	s.deploymentStore = newDeploymentStore()
	for _, dep := range deployments {
		s.deploymentStore.addOrUpdateDeployment(dep)
	}

	s.deploymentStore.OnNamespaceDeleted(namespaceName)

	for _, dep := range deployments {
		s.Assert().Nil(s.deploymentStore.Get(dep.GetId()))
	}
	s.Assert().Len(s.deploymentStore.deploymentIDs[namespaceName], 0)
}

func (s *deploymentStoreSuite) Test_DeleteDeploymentsWithReferences() {
	s.T().Setenv(features.SensorCapturesIntermediateEvents.EnvVar(), "true")
	if !features.SensorCapturesIntermediateEvents.Enabled() {
		s.T().Skipf("Skip tests when %s is disabled", features.SensorCapturesIntermediateEvents.EnvVar())
		s.T().SkipNow()
	}
	for name, tc := range deleteWithReferenceCases {
		s.Run(name, func() {
			s.namespaceStore = newNamespaceStore()
			s.namespaceStore.addNamespace(&storage.NamespaceMetadata{Name: namespaceName, Id: "1"})
			s.deploymentStore = newDeploymentStore()

			for _, deploymentToAdd := range tc.deploymentsToAdd {
				s.deploymentStore.addOrUpdateDeployment(s.createDeploymentWrap(deploymentToAdd))
			}

			for _, referenceToAdd := range tc.deploymentsToReference {
				s.deploymentStore.AddReference(referenceToAdd)
			}

			for _, deploymentToRemove := range tc.deploymentsToRemove {
				s.deploymentStore.removeDeployment(s.createDeploymentWrap(deploymentToRemove))
			}

			for _, referenceToRemove := range tc.deploymentsToDereference {
				s.deploymentStore.RemoveReference(referenceToRemove)
			}

			for _, expectedDeleted := range tc.expectedDeletedDeployments {
				s.Assert().Nil(s.deploymentStore.Get(expectedDeleted))
			}

			for _, expectedMarked := range tc.expectedMarkedDeployments {
				wrap := s.deploymentStore.getWrap(expectedMarked)
				s.Require().NotNil(wrap)
				s.Assert().True(wrap.IsMarkedAsDeleted())
			}
			s.Assert().Len(s.deploymentStore.deploymentIDs[namespaceName], len(tc.expectedMarkedDeployments))
		})
	}
}

func (s *deploymentStoreSuite) Test_OnNamespaceDeletedWithReferences() {
	s.T().Setenv(features.SensorCapturesIntermediateEvents.EnvVar(), "true")
	if !features.SensorCapturesIntermediateEvents.Enabled() {
		s.T().Skipf("Skip tests when %s is disabled", features.SensorCapturesIntermediateEvents.EnvVar())
		s.T().SkipNow()
	}
	for name, tc := range deleteWithReferenceCases {
		s.Run(name, func() {
			s.namespaceStore = newNamespaceStore()
			s.namespaceStore.addNamespace(&storage.NamespaceMetadata{Name: namespaceName, Id: "1"})
			s.deploymentStore = newDeploymentStore()

			for _, deploymentToAdd := range tc.deploymentsToAdd {
				s.deploymentStore.addOrUpdateDeployment(s.createDeploymentWrap(deploymentToAdd))
			}

			for _, referenceToAdd := range tc.deploymentsToReference {
				s.deploymentStore.AddReference(referenceToAdd)
			}

			for _, referenceToRemove := range tc.deploymentsToDereference {
				s.deploymentStore.RemoveReference(referenceToRemove)
			}

			s.deploymentStore.OnNamespaceDeleted(namespaceName)

			for _, expectedDeleted := range tc.expectedDeletedDeployments {
				s.Assert().Nil(s.deploymentStore.Get(expectedDeleted))
			}

			for _, expectedMarked := range tc.expectedMarkedDeployments {
				wrap := s.deploymentStore.getWrap(expectedMarked)
				s.Require().NotNil(wrap)
				s.Assert().True(wrap.IsMarkedAsDeleted())
			}

			s.Assert().Len(s.deploymentStore.deploymentIDs[namespaceName], len(tc.expectedMarkedDeployments))
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
