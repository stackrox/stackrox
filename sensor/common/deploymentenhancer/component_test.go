package deploymentenhancer

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/registry"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/common/store/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestComponent(t *testing.T) {
	suite.Run(t, &ComponentTestSuite{})
}

type ComponentTestSuite struct {
	suite.Suite

	rbacStore *mocks.MockRBACStore
	srvStore  *mocks.MockServiceStore
	depStore  *mocks.MockDeploymentStore

	mockCtrl          *gomock.Controller
	mockStoreProvider store.Provider
}

func (s *ComponentTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.rbacStore = mocks.NewMockRBACStore(s.mockCtrl)
	s.srvStore = mocks.NewMockServiceStore(s.mockCtrl)
	s.depStore = mocks.NewMockDeploymentStore(s.mockCtrl)

	s.mockStoreProvider = mockStoreProvider{
		rbac: s.rbacStore,
		srv:  s.srvStore,
		dep:  s.depStore,
	}
}

func (s *ComponentTestSuite) TearDownTest() {
	s.T().Cleanup(s.mockCtrl.Finish)
}

func (s *ComponentTestSuite) TestExtractAndEnrichDeployments() {
	s.rbacStore.EXPECT().GetPermissionLevelForDeployment(gomock.Any()).AnyTimes()
	s.srvStore.EXPECT().GetExposureInfos(gomock.Any(), gomock.Any()).AnyTimes()
	s.depStore.EXPECT().EnhanceDeploymentReadOnly(gomock.Any(), gomock.Any()).AnyTimes()
	dQueue := make(chan *central.DeploymentEnhancementRequest, 10)
	de := DeploymentEnhancer{
		responsesC:       make(chan *message.ExpiringMessage),
		deploymentsQueue: dQueue,
		storeProvider:    s.mockStoreProvider,
	}

	actual := de.enhanceDeployments(generateDeploymentMsg("1", 4))

	s.Len(actual, 4, "Expected %v deployments, got %v", 4, len(actual))
}

func generateDeploymentMsg(id string, noOfDeployments int) *central.DeploymentEnhancementRequest {
	d := make([]*storage.Deployment, noOfDeployments)
	for i := 0; i < noOfDeployments; i++ {
		d[i] = &storage.Deployment{Id: uuid.NewV4().String()}
	}
	return &central.DeploymentEnhancementRequest{
		Msg: &central.DeploymentEnhancementMessage{
			Id:          id,
			Deployments: d,
		},
	}
}

type mockStoreProvider struct {
	rbac *mocks.MockRBACStore
	srv  *mocks.MockServiceStore
	dep  *mocks.MockDeploymentStore
}

func (m mockStoreProvider) RBAC() store.RBACStore {
	return m.rbac
}

func (m mockStoreProvider) Services() store.ServiceStore {
	return m.srv
}

func (m mockStoreProvider) Deployments() store.DeploymentStore {
	return m.dep
}

func (m mockStoreProvider) Registries() *registry.Store {
	return nil
}

func (m mockStoreProvider) Pods() store.PodStore {
	return nil
}

func (m mockStoreProvider) NetworkPolicies() store.NetworkPolicyStore {
	return nil
}

func (m mockStoreProvider) ServiceAccounts() store.ServiceAccountStore {
	return nil
}

func (m mockStoreProvider) EndpointManager() store.EndpointManager {
	return nil
}

func (m mockStoreProvider) Entities() *clusterentities.Store {
	return nil
}
