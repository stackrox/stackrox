package dependency

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common/ingestion"
	"github.com/stackrox/rox/sensor/common/selector"
	"github.com/stackrox/rox/sensor/common/store"
	mocksStore "github.com/stackrox/rox/sensor/common/store/mocks"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
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
	netpolStore     store.NetworkPolicyStore
	podStore        *mocksStore.MockPodStore

	graph *Graph
}

var _ suite.SetupTestSuite = &DependencyGraphSuite{}

func (s *DependencyGraphSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.deploymentStore = mocksStore.NewMockDeploymentStore(s.mockCtrl)
	s.netpolStore = resources.NewNetworkPolicyStore()
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
	s.deploymentStore.EXPECT().Get(gomock.Eq("d1")).
		AnyTimes().
		Return(d1)

	snapshot := s.graph.GenerateSnapshotFromUpsert("Deployment", "example", "d1")

	s.Require().Len(snapshot.TopLevelNodes, 1)
	s.Require().Equal("Deployment", snapshot.TopLevelNodes[0].Kind)
	s.Require().Len(snapshot.TopLevelNodes[0].Children, 0)
}

func (s *DependencyGraphSuite) Test_DeploymentWithOneNetPolicy() {
	d1 := givenDeployment("example", "d1", appLabel("test"))
	s.deploymentStore.EXPECT().Get(gomock.Eq("d1")).
		AnyTimes().Return(d1)
	s.deploymentStore.EXPECT().GetMatchingDeployments(gomock.Eq("example"), gomock.Any()).
		AnyTimes().
		Return([]*storage.Deployment{d1})

	n1 := givenNetworkPolicy("example", "n1", appLabel("test"))
	s.netpolStore.Upsert(n1)

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

var (
	appTest1Selector = selector.CreateSelector(appLabel("test"), selector.EmptyMatchesEverything())
	appTest2Selector = selector.CreateSelector(appLabel("test2"), selector.EmptyMatchesEverything())
)

func (s *DependencyGraphSuite) Test_NetworkPolicyRelationRemoved() {
	d1 := givenDeployment("example", "d1", appLabel("test"))
	s.deploymentStore.EXPECT().Get(gomock.Eq("d1")).
		AnyTimes().Return(d1)
	s.deploymentStore.EXPECT().GetMatchingDeployments(gomock.Eq("example"), gomock.Eq(appTest1Selector)).
		AnyTimes().
		Return([]*storage.Deployment{d1})

	n1 := givenNetworkPolicy("example", "n1", appLabel("test"))
	s.netpolStore.Upsert(n1)

	// A snapshot w/ a deployment with a single child
	_ = s.graph.GenerateSnapshotFromUpsert("Deployment",  "example", "d1")
	_ = s.graph.GenerateSnapshotFromUpsert("NetworkPolicy",  "example", "n1")

	s.deploymentStore.EXPECT().GetMatchingDeployments(gomock.Eq("example"), gomock.Eq(appTest2Selector)).
		AnyTimes().
		Return([]*storage.Deployment{})

	// n1 now has a different label selector
	n1 = givenNetworkPolicy("example", "n1", appLabel("test2"))
	s.netpolStore.Upsert(n1)

	snapshot := s.graph.GenerateSnapshotFromUpsert("NetworkPolicy", "example", "n1")

	// Since this was disconnected, now we need 2 top level nodes. One is the deployment
	// and the other one is the orphaned NetworkPolicy. Since the NetworkPolicy event
	// needs to be sent to central, we need to make sure that this is also considered and
	// a cluster segment returned.
	s.Require().Len(snapshot.TopLevelNodes, 2)

	s.atLeastOneMathces("NetworkPolicy snapshot", snapshot, func(n *SnapshotNode) bool {
		if n.Kind == "NetworkPolicy" {
			s.Assert().Len(n.Children, 0)
			return true
		}
		return false
	})

	s.atLeastOneMathces("Deployment snapshot", snapshot, func(n *SnapshotNode) bool {
		if n.Kind == "Deployment" {
			s.Assert().Len(n.Children, 0)
			return true
		}
		return false
	})
}

func (s *DependencyGraphSuite) atLeastOneMathces(msg string, result *ClusterSnapshot, fn func(n *SnapshotNode) bool) {
	for _, r := range result.TopLevelNodes {
		if fn(r) {
			return
		}
	}
	s.Failf("expected at least one matching condition, found none: %s", msg)
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
