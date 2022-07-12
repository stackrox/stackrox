package dependency

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/sensor/common/ingestion"
	mocksStore "github.com/stackrox/rox/sensor/common/store/mocks"
)

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

	NewGraph(mockResources)
}
