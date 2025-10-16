package deploymentenhancer

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils/goleak"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/registry"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/common/store/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
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
	goleak.AssertNoGoroutineLeaks(s.T())
	s.T().Cleanup(s.mockCtrl.Finish)
}

func (s *ComponentTestSuite) TestComponentLifecycle() {
	de := CreateEnhancer(s.mockStoreProvider)
	s.NoError(de.Start())
	de.Stop()
	s.NoError(de.Start())
	de.Stop()
}

func (s *ComponentTestSuite) TestEnhanceDeploymentsWithMessage() {
	s.rbacStore.EXPECT().GetPermissionLevelForDeployment(gomock.Any()).AnyTimes()
	s.srvStore.EXPECT().GetExposureInfos(gomock.Any(), gomock.Any()).AnyTimes()
	s.depStore.EXPECT().EnhanceDeploymentReadOnly(gomock.Any(), gomock.Any()).AnyTimes()
	dQueue := make(chan *central.DeploymentEnhancementRequest, 10)
	de := DeploymentEnhancer{
		responsesC:       make(chan *message.ExpiringMessage),
		deploymentsQueue: dQueue,
		storeProvider:    s.mockStoreProvider,
	}

	s.Len(de.enhanceDeployments(generateDeploymentMsg("1", 4)), 4)
}

func (s *ComponentTestSuite) TestEnhanceDeploymentsEmptyMessages() {
	cases := map[string]struct {
		msg *central.DeploymentEnhancementRequest
	}{
		"Empty Message": {
			msg: &central.DeploymentEnhancementRequest{},
		},
		"No Deployments": {
			msg: central.DeploymentEnhancementRequest_builder{Msg: central.DeploymentEnhancementMessage_builder{Id: uuid.NewV4().String()}.Build()}.Build(),
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			dQueue := make(chan *central.DeploymentEnhancementRequest, 10)
			de := DeploymentEnhancer{
				responsesC:       make(chan *message.ExpiringMessage),
				deploymentsQueue: dQueue,
				storeProvider:    s.mockStoreProvider,
			}
			de.enhanceDeployments(c.msg)
		})
	}
}

func (s *ComponentTestSuite) TestMsgQueueOverfill() {
	de := DeploymentEnhancer{
		responsesC:       make(chan *message.ExpiringMessage),
		deploymentsQueue: make(chan *central.DeploymentEnhancementRequest, 1),
		storeProvider:    s.mockStoreProvider,
	}
	s.NoError(de.ProcessMessage(s.T().Context(), generateMsgToSensor()))

	// As there is no reader, the second call has to error out
	s.ErrorContains(de.ProcessMessage(s.T().Context(), generateMsgToSensor()), "DeploymentEnhancer queue has reached its limit of")
}

func generateMsgToSensor() *central.MsgToSensor {
	mts := &central.MsgToSensor{}
	mts.SetDeploymentEnhancementRequest(proto.ValueOrDefault(generateDeploymentMsg(uuid.NewV4().String(), 1)))
	return mts
}

func generateDeploymentMsg(id string, noOfDeployments int) *central.DeploymentEnhancementRequest {
	d := make([]*storage.Deployment, noOfDeployments)
	for i := 0; i < noOfDeployments; i++ {
		deployment := &storage.Deployment{}
		deployment.SetId(uuid.NewV4().String())
		d[i] = deployment
	}
	dem := &central.DeploymentEnhancementMessage{}
	dem.SetId(id)
	dem.SetDeployments(d)
	der := &central.DeploymentEnhancementRequest{}
	der.SetMsg(dem)
	return der
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
