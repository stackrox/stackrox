package dependency

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common/ingestion"
	mocksStore "github.com/stackrox/rox/sensor/common/store/mocks"
	"github.com/stretchr/testify/suite"
)

func Test_DependencyGraphSuite(t *testing.T) {
	suite.Run(t, new(DependencyGraphSuite))
}

type DependencyGraphSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockResources *ingestion.ResourceStore

	deploymentStore *mocksStore.MockDeploymentStore
	netpolStore     *mocksStore.MockNetworkPolicyStore
	podStore        *mocksStore.MockPodStore

	graph *Graph
}

var _ suite.SetupTestSuite = &DependencyGraphSuite{}

func (s *DependencyGraphSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.deploymentStore = mocksStore.NewMockDeploymentStore(s.mockCtrl)
	s.netpolStore = mocksStore.NewMockNetworkPolicyStore(s.mockCtrl)
	s.podStore = mocksStore.NewMockPodStore(s.mockCtrl)

	s.mockResources = &ingestion.ResourceStore{
		Deployments:   s.deploymentStore,
		NetworkPolicy: s.netpolStore,
		PodStore:      s.podStore,
	}

	s.graph = NewGraph(s.mockResources)
}

func (s *DependencyGraphSuite) Test_SingleDeployment() {
	d1 := givenDeployment("example", "d1", nil)
	s.deploymentStore.EXPECT().Get(gomock.Eq("d1")).Return(d1)

	snapshot := s.graph.GenerateSnapshotFromUpsert("Deployment", "example", "d1")

	s.Require().Len(snapshot.TopLevelNodes, 1)
	s.Require().Equal("Deployment", snapshot.TopLevelNodes[0].Kind)
	s.Require().Len(snapshot.TopLevelNodes[0].Children, 0)
}

func (s *DependencyGraphSuite) Test_DeploymentWithOneNetPolicy() {
	d1 := givenDeployment("example", "d1", appLabel("test"))
	s.deploymentStore.EXPECT().Get(gomock.Eq("d1")).Return(d1)

	n1 := givenNetworkPolicy("example", "n1", appLabel("test"))
	s.netpolStore.EXPECT().Get(gomock.Eq("n1")).Return(n1)
	s.netpolStore.EXPECT().Find(gomock.Eq("example"), gomock.Eq(appLabel("test"))).
		Return(map[string]*storage.NetworkPolicy{"n1": n1})

	results := []*ClusterSnapshot{
		s.graph.GenerateSnapshotFromUpsert("NetworkPolicy", "example", "n1"),
		s.graph.GenerateSnapshotFromUpsert("Deployment",  "example", "d1"),
	}

	// In this case, both snapshots should look exactly the same. Given both resources are in
	// stores before we request `GenerateSnapshotFromUpsert`.
	for _, resultCase := range results {
		s.Require().Len(resultCase.TopLevelNodes, 1)
		s.Require().Equal("Deployment", resultCase.TopLevelNodes[0].Kind)
		s.Require().Len(resultCase.TopLevelNodes[0].Children, 1)
		s.Require().Equal("NetworkPolicy", resultCase.TopLevelNodes[0].Children[0].Kind)
	}
}

func appLabel(value string) map[string]string {
	return map[string]string{
		"app": value,
	}
}

func givenDeployment(ns, id string, labels map[string]string) *storage.Deployment {
	return &storage.Deployment{
		Labels:    labels,
		Namespace: ns,
		Id:        id,
	}
}

func givenNetworkPolicy(ns, id string, podSelector map[string]string) *storage.NetworkPolicy {
	return &storage.NetworkPolicy{
		Namespace: ns,
		Id:        id,
		Spec: &storage.NetworkPolicySpec{
			PodSelector: &storage.LabelSelector{
				MatchLabels: podSelector,
			},
		},
	}
}
