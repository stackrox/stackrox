package dependency

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common/ingestion"
	mocksStore "github.com/stackrox/rox/sensor/common/store/mocks"
	"github.com/stretchr/testify/require"
)

func givenDeployment(ns, id string) *storage.Deployment {
	return &storage.Deployment{
		Namespace: ns,
		Id: id,
	}
}

func Test_AddDeploymentToGraph(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	deploymentStore := mocksStore.NewMockDeploymentStore(mockCtrl)
	netpolStore := mocksStore.NewMockNetworkPolicyStore(mockCtrl)
	podStore := mocksStore.NewMockPodStore(mockCtrl)

	mockResources := &ingestion.ResourceStore{
		Deployments:   deploymentStore,
		NetworkPolicy: netpolStore,
		PodStore:      podStore,
	}

	d1 := givenDeployment("example", "d1")
	deploymentStore.EXPECT().Get(gomock.Eq("d1")).Return(d1)

	g := NewGraph(mockResources)


	snapshot := g.GenerateSnapshotFromUpsert("Deployment", "example", "d1")

	require.Len(t, snapshot.TopLevelNodes, 1)
	require.Equal(t, "Deployment", snapshot.TopLevelNodes[0].Kind)
	require.Len(t, snapshot.TopLevelNodes[0].Children, 0)
}
