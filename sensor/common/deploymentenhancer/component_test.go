package deploymentenhancer

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/registry"
	"github.com/stackrox/rox/sensor/common/service"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/common/store/mocks"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
	"github.com/stretchr/testify/suite"
	"go.uber.org/goleak"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/util/intstr"
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
	defer assertNoGoroutineLeaks(s.T())
	s.T().Cleanup(s.mockCtrl.Finish)
}

func assertNoGoroutineLeaks(t *testing.T) {
	goleak.VerifyNone(t,
		// Ignore a known leak: https://github.com/DataDog/dd-trace-go/issues/1469
		goleak.IgnoreTopFunction("github.com/golang/glog.(*fileSink).flushDaemon"),
	)
}

func (s *ComponentTestSuite) TestComponentLifecycle() {
	de := CreateEnhancer(s.mockStoreProvider)
	s.NoError(de.Start())
	de.Stop(nil)
	s.NoError(de.Start())
	de.Stop(nil)
}

func (s *ComponentTestSuite) TestEnhanceDeployment() {
	var ei []map[service.PortRef][]*storage.PortConfig_ExposureInfo
	ex := map[service.PortRef][]*storage.PortConfig_ExposureInfo{{Port: intstr.IntOrString{IntVal: 42}}: make([]*storage.PortConfig_ExposureInfo, 0)}
	ei = append(ei, ex)
	s.mockStoreProvider = mockStoreProvider{
		rbac: s.rbacStore,
		srv:  s.srvStore,
		dep:  &resources.DeploymentStore{},
	}
	s.rbacStore.EXPECT().GetPermissionLevelForDeployment(gomock.Any()).Return(storage.PermissionLevel_DEFAULT)
	s.srvStore.EXPECT().GetExposureInfos(gomock.Any(), gomock.Any()).Return(ei)

	de := DeploymentEnhancer{
		storeProvider: s.mockStoreProvider,
	}
	d := storage.Deployment{
		Id:        uuid.NewV4().String(),
		Name:      "testDeployment",
		Namespace: "testns",
	}

	de.enhanceDeployment(&d)

	s.Equal(storage.PermissionLevel_DEFAULT, d.GetServiceAccountPermissionLevel())
	s.Contains(d.GetPorts(), &storage.PortConfig{ContainerPort: 42})
	s.Empty(s.mockStoreProvider.Deployments().GetAll(), "enhanceDeployment mustn't change or write to the deployment store")
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

	actual := de.enhanceDeployments(generateDeploymentMsg("1", 4))

	expected := 4
	s.Len(actual, expected)
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
	dep  store.DeploymentStore
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
