package manager

import (
	"context"
	"fmt"
	"testing"
	"time"

	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	queueMocks "github.com/stackrox/rox/central/deployment/queue/mocks"
	"github.com/stackrox/rox/central/networkbaseline/datastore"
	networkEntityDSMock "github.com/stackrox/rox/central/networkgraph/entity/datastore/mocks"
	treeMocks "github.com/stackrox/rox/central/networkgraph/entity/networktree/mocks"
	networkFlowDSMocks "github.com/stackrox/rox/central/networkgraph/flow/datastore/mocks"
	networkPolicyMocks "github.com/stackrox/rox/central/networkpolicies/datastore/mocks"
	connectionMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/networkgraph/networkbaseline"
	"github.com/stackrox/rox/pkg/networkgraph/testutils"
	"github.com/stackrox/rox/pkg/networkgraph/tree"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	allAllowedCtx = sac.WithAllAccess(context.Background())
)

type fakeDS struct {
	baselines map[string]*storage.NetworkBaseline

	// All methods not overriden in this struct will panic.
	datastore.DataStore
}

func (f *fakeDS) UpsertNetworkBaselines(_ context.Context, baselines []*storage.NetworkBaseline) error {
	for _, baseline := range baselines {
		f.baselines[baseline.GetDeploymentId()] = baseline
	}
	return nil
}

func (f *fakeDS) Walk(_ context.Context, fn func(baseline *storage.NetworkBaseline) error) error {
	for _, baseline := range f.baselines {
		if err := fn(baseline); err != nil {
			return err
		}
	}
	return nil
}

func (f *fakeDS) DeleteNetworkBaseline(_ context.Context, deploymentID string) error {
	delete(f.baselines, deploymentID)
	return nil
}

func (f *fakeDS) DeleteNetworkBaselines(_ context.Context, deploymentIDs []string) error {
	for _, id := range deploymentIDs {
		delete(f.baselines, id)
	}
	return nil
}

func (f *fakeDS) GetNetworkBaseline(_ context.Context, deploymentID string) (*storage.NetworkBaseline, bool, error) {
	if baseline, ok := f.baselines[deploymentID]; ok {
		return baseline, true, nil
	}
	return nil, false, nil
}

func TestManager(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}

type ManagerTestSuite struct {
	suite.Suite

	ds                         *fakeDS
	networkEntities            *networkEntityDSMock.MockEntityDataStore
	deploymentDS               *deploymentMocks.MockDataStore
	networkPolicyDS            *networkPolicyMocks.MockDataStore
	clusterFlows               *networkFlowDSMocks.MockClusterDataStore
	flowStore                  *networkFlowDSMocks.MockFlowDataStore
	connectionManager          *connectionMocks.MockManager
	deploymentObservationQueue *queueMocks.MockDeploymentObservationQueue
	treeManager                *treeMocks.MockManager

	m             Manager
	currTestStart timestamp.MicroTS
	mockCtrl      *gomock.Controller
}

func (suite *ManagerTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.networkEntities = networkEntityDSMock.NewMockEntityDataStore(suite.mockCtrl)
	suite.currTestStart = timestamp.Now()
	suite.deploymentDS = deploymentMocks.NewMockDataStore(suite.mockCtrl)
	suite.networkPolicyDS = networkPolicyMocks.NewMockDataStore(suite.mockCtrl)
	suite.clusterFlows = networkFlowDSMocks.NewMockClusterDataStore(suite.mockCtrl)
	suite.flowStore = networkFlowDSMocks.NewMockFlowDataStore(suite.mockCtrl)
	suite.connectionManager = connectionMocks.NewMockManager(suite.mockCtrl)
	suite.deploymentObservationQueue = queueMocks.NewMockDeploymentObservationQueue(suite.mockCtrl)
	suite.treeManager = treeMocks.NewMockManager(suite.mockCtrl)
}

func (suite *ManagerTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *ManagerTestSuite) mustInitManager(initialBaselines ...*storage.NetworkBaseline) {
	suite.ds = &fakeDS{baselines: make(map[string]*storage.NetworkBaseline)}
	for _, baseline := range initialBaselines {
		baseline.ObservationPeriodEnd = protoconv.ConvertMicroTSToProtobufTS(getNewObservationPeriodEnd())
		suite.ds.baselines[baseline.GetDeploymentId()] = baseline
	}

	var err error

	suite.networkPolicyDS.EXPECT().GetNetworkPolicies(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	suite.treeManager.EXPECT().GetDefaultNetworkTree(gomock.Any()).Return(nil).AnyTimes()

	suite.m, err = New(suite.ds, suite.networkEntities, suite.deploymentDS, suite.networkPolicyDS, suite.clusterFlows, suite.connectionManager, suite.treeManager)
	suite.Require().NoError(err)
}

func depID(id int) string {
	return fmt.Sprintf("DEP%03d", id)
}

func depName(id int) string {
	return fmt.Sprintf("DEPNAME%03d", id)
}

func clusterID(id int) string {
	return fmt.Sprintf("CLUSTER%d", id)
}

func ns(id int) string {
	return fmt.Sprintf("NS%d", id)
}

func extSrcID(id int) string {
	return fmt.Sprintf("EXTSRC%03d", id)
}

func extSrcName(id int) string {
	return fmt.Sprintf("EXTSRCNAME%03d", id)
}

func (suite *ManagerTestSuite) mustGetBaseline(baselineID int) *storage.NetworkBaseline {
	baseline, found, err := suite.ds.GetNetworkBaseline(managerCtx, depID(baselineID))
	suite.True(found)
	suite.Nil(err)
	return baseline
}

func (suite *ManagerTestSuite) mustGetObserationPeriod(baselineID int) timestamp.MicroTS {
	baseline := suite.mustGetBaseline(baselineID)
	return timestamp.FromProtobuf(baseline.GetObservationPeriodEnd())
}

func (suite *ManagerTestSuite) initBaselinesForDeployments(ids ...int) {
	for _, id := range ids {
		suite.deploymentDS.EXPECT().GetDeployment(gomock.Any(), depID(id)).Return(
			&storage.Deployment{
				Id:        depID(id),
				Name:      depName(id),
				ClusterId: clusterID(id),
				Namespace: ns(id),
			}, true, nil,
		).AnyTimes()
		suite.clusterFlows.EXPECT().GetFlowStore(gomock.Any(), clusterID(id)).Return(suite.flowStore, nil).AnyTimes()
		suite.flowStore.EXPECT().GetFlowsForDeployment(gomock.Any(), depID(id), false).Return(nil, nil).AnyTimes()
		suite.treeManager.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), clusterID(id)).Return(nil).AnyTimes()
		suite.Require().NoError(suite.m.ProcessDeploymentCreate(depID(id), depName(id), clusterID(id), ns(id)))
		suite.Require().NoError(suite.m.CreateNetworkBaseline(depID(id)))
	}
}

func (suite *ManagerTestSuite) processFlowUpdate(inObsPeriodFlows []networkgraph.NetworkConnIndicator, outsideObsPeriodFlows []networkgraph.NetworkConnIndicator) {
	connsToTS := make(map[networkgraph.NetworkConnIndicator]timestamp.MicroTS, len(inObsPeriodFlows)+len(outsideObsPeriodFlows))

	inObsTS := suite.currTestStart.Add(env.NetworkBaselineObservationPeriod.DurationSetting()).Add(-1 * time.Minute)
	for _, flow := range inObsPeriodFlows {
		_, exists := connsToTS[flow]
		suite.Require().False(exists)
		connsToTS[flow] = inObsTS
	}

	outsideObsTS := suite.currTestStart.Add(env.NetworkBaselineObservationPeriod.DurationSetting()).Add(1 * time.Minute)
	for _, flow := range outsideObsPeriodFlows {
		_, exists := connsToTS[flow]
		suite.Require().False(exists)
		connsToTS[flow] = outsideObsTS
	}

	suite.Require().NoError(suite.m.ProcessFlowUpdate(connsToTS))
}

func (suite *ManagerTestSuite) assertBaselinesAre(baselines ...*storage.NetworkBaseline) {
	baselinesWithoutObsPeriod := make([]*storage.NetworkBaseline, 0, len(suite.ds.baselines))
	obsPeriodStart := suite.currTestStart.Add(env.NetworkBaselineObservationPeriod.DurationSetting())
	// Assume that the test takes no longer than one minute.
	obsPeriodEnd := obsPeriodStart.Add(time.Minute)
	for _, baseline := range suite.ds.baselines {
		cloned := baseline.CloneVT()
		actualObsEnd := timestamp.FromProtobuf(cloned.GetObservationPeriodEnd())
		suite.True(actualObsEnd.After(obsPeriodStart), "Actual obs end: %v, expected obs window: %v-%v", actualObsEnd.GoTime(), obsPeriodStart.GoTime(), obsPeriodEnd.GoTime())
		suite.True(obsPeriodEnd.After(actualObsEnd), "Actual obs end: %v, expected obs window: %v-%v", actualObsEnd.GoTime(), obsPeriodStart.GoTime(), obsPeriodEnd.GoTime())
		cloned.ObservationPeriodEnd = nil
		baselinesWithoutObsPeriod = append(baselinesWithoutObsPeriod, cloned)
	}

	protoassert.ElementsMatch(suite.T(), baselines, baselinesWithoutObsPeriod)
}

func (suite *ManagerTestSuite) TestFlowsUpdateForOtherEntityTypes() {
	suite.mustInitManager()
	suite.initBaselinesForDeployments(1, 2, 3)
	suite.assertBaselinesAre(emptyBaseline(1), emptyBaseline(2), emptyBaseline(3))
	suite.processFlowUpdate(conns(depToDepConn(1, 2, 52)), conns(depToDepConn(2, 3, 51)))
	suite.assertBaselinesAre(
		baselineWithPeers(1, depPeer(2, properties(false, 52))),
		baselineWithPeers(2, depPeer(1, properties(true, 52))),
		emptyBaseline(3),
	)
	suite.networkEntities.EXPECT().GetEntity(gomock.Any(), extSrcID(10)).Return(
		&storage.NetworkEntity{
			Info: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				Id:   extSrcID(10),
				Desc: &storage.NetworkEntityInfo_ExternalSource_{
					ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
						Name:   extSrcName(10),
						Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{Cidr: "11.0.0.0/32"},
					},
				},
			},
		}, true, nil)

	suite.processFlowUpdate([]networkgraph.NetworkConnIndicator{
		// This conn is valid and should get incorporated into the baseline.
		{
			SrcEntity: networkgraph.Entity{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				ID:   depID(1),
			},
			DstEntity: networkgraph.Entity{
				Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				ID:   extSrcID(10),
			},
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			DstPort:  1,
		},
		// This conn is valid and should get incorporated into the baseline
		// and should be anonymised to the internet
		{
			SrcEntity: networkgraph.Entity{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				ID:   depID(1),
			},
			DstEntity: networkgraph.Entity{
				Type:       storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				ID:         extSrcID(11),
				Discovered: true,
			},
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			DstPort:  12,
		},
		// This conn is valid and should get incorporated into the baseline.
		{
			SrcEntity: networkgraph.Entity{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				ID:   depID(1),
			},
			DstEntity: networkgraph.Entity{
				Type: storage.NetworkEntityInfo_INTERNET,
				ID:   "INTERNETTZZ",
			},
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			DstPort:  13,
		},
		// This conn is also valid and should get incorporated into the baseline.
		{
			SrcEntity: networkgraph.Entity{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				ID:   depID(1),
			},
			DstEntity: networkgraph.Entity{
				Type: storage.NetworkEntityInfo_INTERNAL_ENTITIES,
				ID:   "INTERNALZZ",
			},
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			DstPort:  13,
		},
		// This is to a listen endpoint, so it should not get incorporated.
		{
			SrcEntity: networkgraph.Entity{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				ID:   depID(1),
			},
			DstEntity: networkgraph.Entity{
				Type: storage.NetworkEntityInfo_LISTEN_ENDPOINT,
				ID:   "LISTEN",
			},
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			DstPort:  1,
		},
		// Entities without ids should get ignored.
		{
			SrcEntity: networkgraph.Entity{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				ID:   depID(1),
			},
			DstEntity: networkgraph.Entity{
				Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				ID:   "",
			},
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			DstPort:  1,
		},
	}, nil)
	suite.assertBaselinesAre(
		baselineWithPeers(1, depPeer(2, properties(false, 52)),
			&storage.NetworkBaselinePeer{
				Entity: &storage.NetworkEntity{Info: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
					Id:   extSrcID(10),
					Desc: &storage.NetworkEntityInfo_ExternalSource_{
						ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
							Name:   extSrcName(10),
							Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{Cidr: "11.0.0.0/32"},
						},
					},
				}},
				Properties: []*storage.NetworkBaselineConnectionProperties{properties(false, 1)},
			},
			&storage.NetworkBaselinePeer{
				Entity: &storage.NetworkEntity{Info: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_INTERNAL_ENTITIES,
					Id:   "INTERNALZZ",
				}},
				Properties: []*storage.NetworkBaselineConnectionProperties{properties(false, 13)},
			},
			&storage.NetworkBaselinePeer{
				Entity: &storage.NetworkEntity{Info: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_INTERNET,
					Id:   "INTERNETTZZ",
				}},
				Properties: []*storage.NetworkBaselineConnectionProperties{properties(false, 13)},
			},
			&storage.NetworkBaselinePeer{
				Entity: &storage.NetworkEntity{Info: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_INTERNET,
					Id:   networkgraph.InternetEntity().ID,
				}},
				Properties: []*storage.NetworkBaselineConnectionProperties{properties(false, 12)},
			},
		),
		baselineWithPeers(2, depPeer(1, properties(true, 52))),
		emptyBaseline(3),
	)
}

func (suite *ManagerTestSuite) TestFlowsUpdate() {
	suite.mustInitManager()
	suite.initBaselinesForDeployments(1, 2, 3)
	suite.assertBaselinesAre(emptyBaseline(1), emptyBaseline(2), emptyBaseline(3))
	suite.processFlowUpdate(conns(depToDepConn(1, 2, 52)), conns(depToDepConn(2, 3, 51)))
	suite.assertBaselinesAre(
		baselineWithPeers(1, depPeer(2, properties(false, 52))),
		baselineWithPeers(2, depPeer(1, properties(true, 52))),
		emptyBaseline(3),
	)
	suite.processFlowUpdate(conns(depToDepConn(2, 3, 51), depToDepConn(3, 1, 443), depToDepConn(4, 1, 512)), nil)
	suite.assertBaselinesAre(
		baselineWithPeers(1, depPeer(2, properties(false, 52)), depPeer(3, properties(true, 443))),
		baselineWithPeers(2, depPeer(1, properties(true, 52)), depPeer(3, properties(false, 51))),
		baselineWithPeers(3, depPeer(1, properties(false, 443)), depPeer(2, properties(true, 51))),
	)
}

func (suite *ManagerTestSuite) TestRepeatedCreates() {
	suite.mustInitManager()

	suite.initBaselinesForDeployments(1, 2, 3)
	suite.assertBaselinesAre(emptyBaseline(1), emptyBaseline(2), emptyBaseline(3))
	suite.initBaselinesForDeployments(1) // Should be a no-op
	suite.assertBaselinesAre(emptyBaseline(1), emptyBaseline(2), emptyBaseline(3))
	suite.processFlowUpdate(conns(depToDepConn(1, 2, 52)), conns(depToDepConn(2, 3, 51)))
	suite.assertBaselinesAre(
		baselineWithPeers(1, depPeer(2, properties(false, 52))),
		baselineWithPeers(2, depPeer(1, properties(true, 52))),
		emptyBaseline(3),
	)
	suite.initBaselinesForDeployments(1) // Should be a no-op
	suite.assertBaselinesAre(
		baselineWithPeers(1, depPeer(2, properties(false, 52))),
		baselineWithPeers(2, depPeer(1, properties(true, 52))),
		emptyBaseline(3),
	)
}

func (suite *ManagerTestSuite) TestResilienceToRestarts() {
	suite.networkPolicyDS.EXPECT().GetNetworkPolicies(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	// This simulates the case where the datastore has some preexisting baselines
	// at the time the manager starts. The manager must load them on init.
	suite.mustInitManager(
		baselineWithPeers(1, depPeer(2, properties(false, 52))),
		baselineWithPeers(2, depPeer(1, properties(true, 52))),
		emptyBaseline(3),
	)
	suite.assertBaselinesAre(
		baselineWithPeers(1, depPeer(2, properties(false, 52))),
		baselineWithPeers(2, depPeer(1, properties(true, 52))),
		emptyBaseline(3),
	)
	suite.initBaselinesForDeployments(4)
	suite.assertBaselinesAre(
		baselineWithPeers(1, depPeer(2, properties(false, 52))),
		baselineWithPeers(2, depPeer(1, properties(true, 52))),
		emptyBaseline(3),
		emptyBaseline(4),
	)
	suite.processFlowUpdate(conns(depToDepConn(2, 3, 51), depToDepConn(3, 1, 443), depToDepConn(4, 1, 512)), nil)
	suite.assertBaselinesAre(
		baselineWithPeers(1, depPeer(2, properties(false, 52)), depPeer(3, properties(true, 443)), depPeer(4, properties(true, 512))),
		baselineWithPeers(2, depPeer(1, properties(true, 52)), depPeer(3, properties(false, 51))),
		baselineWithPeers(3, depPeer(1, properties(false, 443)), depPeer(2, properties(true, 51))),
		baselineWithPeers(4, depPeer(1, properties(false, 512))),
	)
}

func (suite *ManagerTestSuite) TestConcurrentUpdates() {
	suite.networkPolicyDS.EXPECT().GetNetworkPolicies(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	suite.mustInitManager(emptyBaseline(1))
	suite.assertBaselinesAre(emptyBaseline(1))
	var wg sync.WaitGroup
	const elems = 50
	for i := 2; i <= elems; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			suite.initBaselinesForDeployments(idx)
			suite.processFlowUpdate(conns(depToDepConn(1, idx, uint32(idx+2))), conns(depToDepConn(1, idx, uint32(idx+3))))
		}(i)
	}
	wg.Wait()
	var expectedBaselines []*storage.NetworkBaseline
	firstPeers := make([]*storage.NetworkBaselinePeer, 0, elems-1)
	for i := 2; i <= elems; i++ {
		firstPeers = append(firstPeers, depPeer(i, properties(false, uint32(i+2))))
		expectedBaselines = append(expectedBaselines, baselineWithPeers(i, depPeer(1, properties(true, uint32(i+2)))))
	}
	expectedBaselines = append(expectedBaselines, baselineWithPeers(1, firstPeers...))
	suite.assertBaselinesAre(expectedBaselines...)
}

func (suite *ManagerTestSuite) TestUpdateBaselineStatus() {
	suite.networkPolicyDS.EXPECT().GetNetworkPolicies(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	// Seed with a set of baselines.
	suite.mustInitManager(
		baselineWithPeers(1, depPeer(2, properties(false, 52)), depPeer(3, properties(true, 512))),
		baselineWithPeers(2, depPeer(1, properties(true, 52))),
		baselineWithPeers(3, depPeer(1, properties(false, 512))),
	)

	// No deployment ID -- should fail.
	suite.Error(suite.m.ProcessBaselineStatusUpdate(allAllowedCtx, &v1.ModifyBaselineStatusForPeersRequest{
		Peers: []*v1.NetworkBaselinePeerStatus{protoPeerStatus(v1.NetworkBaselinePeerStatus_BASELINE, 2, 52, true)},
	}))

	// Non existent deployment ID -- should fail.
	suite.Error(suite.m.ProcessBaselineStatusUpdate(allAllowedCtx,
		modifyPeersReq(10, protoPeerStatus(v1.NetworkBaselinePeerStatus_BASELINE, 2, 52, true)),
	))

	// Referencing a non-existent deployment ID as a peer -- should fail.
	suite.Error(suite.m.ProcessBaselineStatusUpdate(allAllowedCtx,
		modifyPeersReq(1, protoPeerStatus(v1.NetworkBaselinePeerStatus_BASELINE, 20, 52, true)),
	))

	// Trying to add a listen endpoint as a peer -- should fail.
	suite.Error(suite.m.ProcessBaselineStatusUpdate(allAllowedCtx,
		modifyPeersReq(1, &v1.NetworkBaselinePeerStatus{
			Peer: &v1.NetworkBaselineStatusPeer{
				Entity: &v1.NetworkBaselinePeerEntity{
					Id:   "",
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
				},
				Port:     52,
				Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
				Ingress:  true,
			},
			Status: v1.NetworkBaselinePeerStatus_ANOMALOUS,
		},
		)))

	// SAC enforcement: should not be able to modify other deployment.
	suite.Error(suite.m.ProcessBaselineStatusUpdate(ctxWithAccessToWrite(2),
		modifyPeersReq(1, protoPeerStatus(v1.NetworkBaselinePeerStatus_BASELINE, 2, 443, true)),
	))

	// Everything from below should work, since we will use correct SAC.

	// Add a flow to baseline. Check baselines have been modified as expected.
	suite.NoError(suite.m.ProcessBaselineStatusUpdate(ctxWithAccessToWrite(1),
		modifyPeersReq(1, protoPeerStatus(v1.NetworkBaselinePeerStatus_BASELINE, 2, 443, true)),
	))

	suite.assertBaselinesAre(
		baselineWithPeers(1, depPeer(2, properties(true, 443), properties(false, 52)), depPeer(3, properties(true, 512))),
		baselineWithPeers(2, depPeer(1, properties(true, 52), properties(false, 443))),
		baselineWithPeers(3, depPeer(1, properties(false, 512))),
	)

	// Add the same flow to the baseline -- should be no difference in baselines.
	suite.NoError(suite.m.ProcessBaselineStatusUpdate(ctxWithAccessToWrite(1),
		modifyPeersReq(1, protoPeerStatus(v1.NetworkBaselinePeerStatus_BASELINE, 2, 443, true)),
	))
	suite.assertBaselinesAre(
		baselineWithPeers(1, depPeer(2, properties(true, 443), properties(false, 52)), depPeer(3, properties(true, 512))),
		baselineWithPeers(2, depPeer(1, properties(true, 52), properties(false, 443))),
		baselineWithPeers(3, depPeer(1, properties(false, 512))),
	)

	// Mark a random new flow as anomalous, ensure it shows as forbidden.
	suite.NoError(suite.m.ProcessBaselineStatusUpdate(ctxWithAccessToWrite(2),
		modifyPeersReq(2, protoPeerStatus(v1.NetworkBaselinePeerStatus_ANOMALOUS, 3, 8443, true)),
	))
	suite.assertBaselinesAre(
		baselineWithPeers(1, depPeer(2, properties(true, 443), properties(false, 52)), depPeer(3, properties(true, 512))),
		wrapWithForbidden(
			baselineWithPeers(2, depPeer(1, properties(true, 52), properties(false, 443))),
			depPeer(3, properties(true, 8443)),
		),
		wrapWithForbidden(
			baselineWithPeers(3, depPeer(1, properties(false, 512))),
			depPeer(2, properties(false, 8443)),
		),
	)

	// Mark an existing baseline flow as anomalous, ensure it gets removed from the baseline
	// and shows as forbidden.
	suite.NoError(suite.m.ProcessBaselineStatusUpdate(ctxWithAccessToWrite(2),
		modifyPeersReq(2, protoPeerStatus(v1.NetworkBaselinePeerStatus_ANOMALOUS, 1, 52, true)),
	))
	suite.assertBaselinesAre(
		wrapWithForbidden(
			baselineWithPeers(1, depPeer(2, properties(true, 443)), depPeer(3, properties(true, 512))),
			depPeer(2, properties(false, 52)),
		),
		wrapWithForbidden(baselineWithPeers(2, depPeer(1, properties(false, 443))),
			depPeer(1, properties(true, 52)), depPeer(3, properties(true, 8443)),
		),
		wrapWithForbidden(baselineWithPeers(3, depPeer(1, properties(false, 512))),
			depPeer(2, properties(false, 8443)),
		),
	)

	// Add an external discovered anomalous flow
	suite.NoError(suite.m.ProcessBaselineStatusUpdate(ctxWithAccessToWrite(1),
		modifyPeersReq(1, protoExternalPeerStatus(v1.NetworkBaselinePeerStatus_ANOMALOUS, 1, 52, true, true)),
	))

	suite.assertBaselinesAre(
		wrapWithForbidden(
			baselineWithPeers(1, depPeer(2, properties(true, 443)), depPeer(3, properties(true, 512))),
			depPeer(2, properties(false, 52)),
			// added anomalous internet flow (the external flow is anonymized to the internet)
			internetPeer(properties(true, 52)),
		),
		wrapWithForbidden(baselineWithPeers(2, depPeer(1, properties(false, 443))),
			depPeer(1, properties(true, 52)), depPeer(3, properties(true, 8443)),
		),
		wrapWithForbidden(baselineWithPeers(3, depPeer(1, properties(false, 512))),
			depPeer(2, properties(false, 8443)),
		),
	)

	// Then make it a baseline flow

	suite.NoError(suite.m.ProcessBaselineStatusUpdate(ctxWithAccessToWrite(1),
		modifyPeersReq(1, protoExternalPeerStatus(v1.NetworkBaselinePeerStatus_BASELINE, 1, 52, true, true)),
	))

	suite.assertBaselinesAre(
		wrapWithForbidden(
			baselineWithPeers(1, depPeer(2, properties(true, 443)), depPeer(3, properties(true, 512)), internetPeer(properties(true, 52))),
			depPeer(2, properties(false, 52)),
		),
		wrapWithForbidden(baselineWithPeers(2, depPeer(1, properties(false, 443))),
			depPeer(1, properties(true, 52)), depPeer(3, properties(true, 8443)),
		),
		wrapWithForbidden(baselineWithPeers(3, depPeer(1, properties(false, 512))),
			depPeer(2, properties(false, 8443)),
		),
	)
}

func (suite *ManagerTestSuite) TestDeploymentDelete() {
	suite.networkPolicyDS.EXPECT().GetNetworkPolicies(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	suite.mustInitManager(
		wrapWithForbidden(
			baselineWithPeers(1, depPeer(2, properties(false, 52))),
			depPeer(3, properties(true, 443)),
		),
		wrapWithForbidden(
			baselineWithPeers(2, depPeer(1, properties(true, 52))),
			depPeer(3, properties(true, 443)),
		),
		wrapWithForbidden(
			emptyBaseline(3),
			depPeer(1, properties(false, 443)),
			depPeer(2, properties(false, 443)),
		),
	)
	suite.assertBaselinesAre(
		wrapWithForbidden(
			baselineWithPeers(1, depPeer(2, properties(false, 52))),
			depPeer(3, properties(true, 443)),
		),
		wrapWithForbidden(
			baselineWithPeers(2, depPeer(1, properties(true, 52))),
			depPeer(3, properties(true, 443)),
		),
		wrapWithForbidden(
			emptyBaseline(3),
			depPeer(1, properties(false, 443)),
			depPeer(2, properties(false, 443)),
		),
	)
	// First delete dep 3 to verify deletion of forbidden peers
	suite.Nil(suite.m.ProcessDeploymentDelete(depID(3)))
	suite.assertBaselinesAre(
		baselineWithPeers(1, depPeer(2, properties(false, 52))),
		baselineWithPeers(2, depPeer(1, properties(true, 52))),
	)
	// Then delete another dep to verify deletion of allowed peers
	suite.Nil(suite.m.ProcessDeploymentDelete(depID(2)))
	suite.assertBaselinesAre(
		emptyBaseline(1),
	)
}

func (suite *ManagerTestSuite) TestDeploymentDelete_WithoutBaseline() {
	suite.mustInitManager(
		baselineWithPeers(1, depPeer(2, properties(false, 52))),
		wrapWithForbidden(baselineWithPeers(3),
			depPeer(2, properties(false, 52))),
	)

	// DeploymentID 2 should be in the internal map, but no baseline created yet.
	suite.Require().NoError(suite.m.ProcessDeploymentCreate(depID(2), depName(2), clusterID(2), ns(2)))

	suite.Require().NoError(suite.m.ProcessDeploymentDelete(depID(2)))

	// Should remove DeploymentID 2 from other baselines (BaselinedPeers and ForbiddenPeers) even if its baseline was never created
	suite.assertBaselinesAre(
		baselineWithPeers(1),
		baselineWithPeers(3),
	)
}

func (suite *ManagerTestSuite) TestDeleteWithExtSrcPeer() {
	suite.networkPolicyDS.EXPECT().GetNetworkPolicies(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	suite.mustInitManager(
		baselineWithPeers(
			1,
			depPeer(2, properties(false, 52)),
			extSrcPeer(3, "11.0.0.0/32", properties(false, 443)),
		),
		baselineWithPeers(2, depPeer(1, properties(true, 52))),
	)
	suite.assertBaselinesAre(
		baselineWithPeers(
			1,
			depPeer(2, properties(false, 52)),
			extSrcPeer(3, "11.0.0.0/32", properties(false, 443)),
		),
		baselineWithPeers(2, depPeer(1, properties(true, 52))),
	)
	// Make sure deleting a deployment does not trigger an error even if we have ext src peer
	suite.Nil(suite.m.ProcessDeploymentDelete(depID(2)))
	suite.assertBaselinesAre(baselineWithPeers(1, extSrcPeer(3, "11.0.0.0/32", properties(false, 443))))
}

func (suite *ManagerTestSuite) TestValidEntityTypesMatch() {
	validTypes := make([]storage.NetworkEntityInfo_Type, 0, len(networkbaseline.ValidBaselinePeerEntityTypes))
	for t := range networkbaseline.ValidBaselinePeerEntityTypes {
		validTypes = append(validTypes, t)
	}

	// Make sure all the variables relying on entity types have the correct set of valid entity types
	types := make([]storage.NetworkEntityInfo_Type, 0, len(networkgraph.EntityTypeToName))
	for t := range networkgraph.EntityTypeToName {
		types = append(types, t)
	}
	suite.ElementsMatch(validTypes, types)

	types = types[:0]
	for t := range networkbaseline.EntityTypeToEntityInfoDesc {
		types = append(types, t)
	}
	suite.ElementsMatch(validTypes, types)
}

func (suite *ManagerTestSuite) TestProcessNetworkPolicyUpdate() {
	suite.networkPolicyDS.EXPECT().GetNetworkPolicies(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	suite.mustInitManager(
		baselineWithPeers(1, depPeer(2, properties(false, 52))),
		baselineWithPeers(2, depPeer(1, properties(true, 52))),
	)
	suite.assertBaselinesAre(
		baselineWithPeers(1, depPeer(2, properties(false, 52))),
		baselineWithPeers(2, depPeer(1, properties(true, 52))),
	)
	originalObservationEnd := suite.mustGetObserationPeriod(1)

	// Now try process a network flow update
	matchLabels := map[string]string{"app": "test"}
	networkPolicy := getNetworkPolicy(matchLabels)
	// Check the query
	deploymentSearchQuery :=
		search.
			NewQueryBuilder().
			AddExactMatches(search.ClusterID, networkPolicy.GetClusterId()).
			AddExactMatches(search.Namespace, networkPolicy.GetNamespace()).
			ProtoQuery()
	suite.deploymentDS.EXPECT().SearchRawDeployments(gomock.Any(), deploymentSearchQuery).Return(
		[]*storage.Deployment{
			{
				Id:        depID(1),
				ClusterId: networkPolicy.GetClusterId(),
				Namespace: networkPolicy.GetNamespace(),
				PodLabels: matchLabels,
			},
		}, nil).AnyTimes()

	suite.Nil(suite.m.ProcessNetworkPolicyUpdate(managerCtx, central.ResourceAction_CREATE_RESOURCE, networkPolicy))
	// Make sure baseline contents other than observation period is not altered
	suite.assertBaselinesAre(
		baselineWithPeers(1, depPeer(2, properties(false, 52))),
		baselineWithPeers(2, depPeer(1, properties(true, 52))),
	)
	afterPolicyCreateObservationPeriod := suite.mustGetObserationPeriod(1)
	suite.True(afterPolicyCreateObservationPeriod.After(originalObservationEnd))

	// Now test dedupe of the network policy
	// Sending in the same policy should not change the observation period
	suite.Nil(suite.m.ProcessNetworkPolicyUpdate(managerCtx, central.ResourceAction_CREATE_RESOURCE, networkPolicy))
	afterDedupedPolicyUpdateObservationPeriod := suite.mustGetObserationPeriod(1)
	suite.Equal(afterDedupedPolicyUpdateObservationPeriod, afterPolicyCreateObservationPeriod)

	// But then if the action is different, we should still update the observation period
	suite.Nil(suite.m.ProcessNetworkPolicyUpdate(managerCtx, central.ResourceAction_UPDATE_RESOURCE, networkPolicy))
	afterActionChangeObservationPeriod := suite.mustGetObserationPeriod(1)
	suite.True(afterActionChangeObservationPeriod.After(afterPolicyCreateObservationPeriod))

	// Or changing the policy content should update the observation period as well
	rule := networkPolicy.GetSpec().GetIngress()[0]
	rule.Ports =
		append(
			rule.Ports,
			&storage.NetworkPolicyPort{
				Protocol: storage.Protocol_TCP_PROTOCOL,
				PortRef:  &storage.NetworkPolicyPort_Port{Port: 1234}})
	networkPolicy.Spec.Ingress = []*storage.NetworkPolicyIngressRule{rule}
	suite.Nil(suite.m.ProcessNetworkPolicyUpdate(managerCtx, central.ResourceAction_CREATE_RESOURCE, networkPolicy))
	afterPolicyRuleUpdateObservationPeriod := suite.mustGetObserationPeriod(1)
	suite.True(afterPolicyRuleUpdateObservationPeriod.After(afterActionChangeObservationPeriod))

	// Or changing the pod selector of the policy
	networkPolicy.Spec.PodSelector.MatchLabels["another_tag"] = "another_value"
	suite.Nil(suite.m.ProcessNetworkPolicyUpdate(managerCtx, central.ResourceAction_CREATE_RESOURCE, networkPolicy))
	afterPodSelectorUpdateObservationPeriod := suite.mustGetObserationPeriod(1)
	suite.True(afterPodSelectorUpdateObservationPeriod.After(afterPolicyRuleUpdateObservationPeriod))
}

func (suite *ManagerTestSuite) TestLockBaseline() {
	suite.networkPolicyDS.EXPECT().GetNetworkPolicies(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	suite.mustInitManager(
		baselineWithPeers(1, depPeer(2, properties(false, 52))),
		baselineWithPeers(2, depPeer(1, properties(true, 52))),
	)
	suite.assertBaselinesAre(
		baselineWithPeers(1, depPeer(2, properties(false, 52))),
		baselineWithPeers(2, depPeer(1, properties(true, 52))),
	)
	baseline1 := suite.mustGetBaseline(1)
	beforeLockUpdateState := baseline1.GetLocked()
	baseline1Copy := baseline1.CloneVT()
	baseline1Copy.Locked = !beforeLockUpdateState
	expectOneTimeCallToConnectionManagerWithBaseline(suite, baseline1Copy)

	suite.Nil(suite.m.ProcessBaselineLockUpdate(managerCtx, depID(1), !beforeLockUpdateState))
	afterLockUpdateState := suite.mustGetBaseline(1).GetLocked()
	suite.NotEqual(beforeLockUpdateState, afterLockUpdateState)
}

func (suite *ManagerTestSuite) TestProcessPostClusterDelete() {
	suite.networkPolicyDS.EXPECT().GetNetworkPolicies(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	suite.mustInitManager(
		baselineWithClusterAndPeers(
			1,
			10,
			depPeer(2, properties(false, 52)),
			depPeer(3, properties(true, 443)),
			depPeer(4, properties(false, 443))),
		baselineWithClusterAndPeers(
			2,
			10,
			depPeer(1, properties(true, 52)),
			depPeer(3, properties(true, 443))),
		baselineWithClusterAndPeers(
			3,
			11,
			depPeer(1, properties(false, 443)),
			depPeer(2, properties(false, 443))),
		wrapWithForbidden(
			baselineWithClusterAndPeers(4, 12),
			depPeer(1, properties(true, 443)),
		),
	)
	suite.assertBaselinesAre(
		baselineWithClusterAndPeers(
			1,
			10,
			depPeer(2, properties(false, 52)),
			depPeer(3, properties(true, 443)),
			depPeer(4, properties(false, 443))),
		baselineWithClusterAndPeers(
			2,
			10,
			depPeer(1, properties(true, 52)),
			depPeer(3, properties(true, 443))),
		baselineWithClusterAndPeers(
			3,
			11,
			depPeer(1, properties(false, 443)),
			depPeer(2, properties(false, 443))),
		wrapWithForbidden(
			baselineWithClusterAndPeers(4, 12),
			depPeer(1, properties(true, 443)),
		),
	)
	deletedIDs := []string{depID(1), depID(2)}
	suite.Nil(suite.m.ProcessPostClusterDelete(deletedIDs))
	suite.assertBaselinesAre(
		baselineWithClusterAndPeers(3, 11),
		baselineWithClusterAndPeers(4, 12),
	)
}

func (suite *ManagerTestSuite) TestBaselineSyncMsg() {
	suite.networkPolicyDS.EXPECT().GetNetworkPolicies(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	suite.mustInitManager(
		baselineWithPeers(1, depPeer(2, properties(false, 52))),
		baselineWithPeers(2, depPeer(1, properties(true, 52))),
	)
	suite.assertBaselinesAre(
		baselineWithPeers(1, depPeer(2, properties(false, 52))),
		baselineWithPeers(2, depPeer(1, properties(true, 52))),
	)
	// Lock state unchanged (unlocked). Does not expect a call to connection manager
	suite.Nil(suite.m.ProcessBaselineLockUpdate(managerCtx, depID(1), false))

	baseline1 := suite.mustGetBaseline(1)
	baseline1Copy := baseline1.CloneVT()
	baseline1Copy.Locked = true
	// Lock state changed from unlocked to locked, we should sync this baseline to sensor now
	expectOneTimeCallToConnectionManagerWithBaseline(suite, baseline1Copy)
	suite.Nil(suite.m.ProcessBaselineLockUpdate(managerCtx, depID(1), true))
	afterLockUpdateState := suite.mustGetBaseline(1).GetLocked()
	suite.True(afterLockUpdateState)

	// If it stays as locked, and some updates are made to the baseline, then we should also sync to sensor
	modifiedBaseline := baselineWithPeers(1)
	modifiedBaseline.Locked = baseline1Copy.GetLocked()
	modifiedBaseline.ObservationPeriodEnd = baseline1.ObservationPeriodEnd
	expectOneTimeCallToConnectionManagerWithBaseline(suite, modifiedBaseline)
	suite.Nil(suite.m.ProcessDeploymentDelete(depID(2)))

	// If baseline changed from locked to unlocked, we should also sync to sensor
	modifiedBaseline.Locked = false
	expectOneTimeCallToConnectionManagerWithBaseline(suite, modifiedBaseline)
	suite.Nil(suite.m.ProcessBaselineLockUpdate(managerCtx, depID(1), false))
	// And locked state should be updated
	afterLockUpdateState = suite.mustGetBaseline(1).GetLocked()
	suite.False(afterLockUpdateState)
}

func (suite *ManagerTestSuite) TestGetExternalNetworkPeers() {
	suite.mustInitManager()
	suite.deploymentDS.EXPECT().GetDeployment(gomock.Any(), gomock.Any()).Return(
		&storage.Deployment{
			Id:        fixtureconsts.Deployment1,
			ClusterId: fixtureconsts.Cluster1,
		}, true, nil,
	)

	entities := []*storage.NetworkEntity{
		testutils.GetExtSrcNetworkEntity("entity1", "1.2.3.4", "1.2.3.4/32", false, fixtureconsts.Cluster1, true),
		testutils.GetExtSrcNetworkEntity("entity2", "1.2.3.5", "1.2.3.5/32", false, fixtureconsts.Cluster1, true),
		testutils.GetExtSrcNetworkEntity("entity3", "1.2.3.6", "1.2.3.6/32", false, fixtureconsts.Cluster1, false),
		testutils.GetExtSrcNetworkEntity("entity4", "1.2.3.7", "1.2.3.7/32", false, fixtureconsts.Cluster1, false),
	}

	flows := []*storage.NetworkFlow{
		testutils.ExtFlow("entity1", fixtureconsts.Deployment1),
		testutils.ExtFlow("entity2", fixtureconsts.Deployment1),
		testutils.ExtFlow("entity3", fixtureconsts.Deployment1),
		// duplicate flow to test name aggregation
		testutils.ExtFlow("entity3", fixtureconsts.Deployment1),
	}

	mockTree := tree.NewDefaultNetworkTreeWrapper()
	for _, entity := range entities {
		suite.NoError(mockTree.Insert(entity.GetInfo()))
	}

	suite.treeManager.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), fixtureconsts.Cluster1).Return(mockTree).AnyTimes()
	suite.treeManager.EXPECT().GetDefaultNetworkTree(gomock.Any()).Return(nil).AnyTimes()
	suite.networkEntities.EXPECT().GetEntityByQuery(gomock.Any(), gomock.Any()).Return(entities, nil)
	suite.clusterFlows.EXPECT().GetFlowStore(gomock.Any(), fixtureconsts.Cluster1).Return(suite.flowStore, nil).AnyTimes()
	suite.flowStore.EXPECT().GetMatchingFlows(gomock.Any(), gomock.Any(), gomock.Any()).Return(flows, nil, nil).AnyTimes()

	expectedPeers := []*v1.NetworkBaselineStatusPeer{
		{
			Entity: &v1.NetworkBaselinePeerEntity{
				Id:         "entity1",
				Type:       storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				Name:       "1.2.3.4",
				Discovered: true,
			},
		},

		{
			Entity: &v1.NetworkBaselinePeerEntity{
				Id:         "entity2",
				Type:       storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				Name:       "1.2.3.5",
				Discovered: true,
			},
		},

		{
			Entity: &v1.NetworkBaselinePeerEntity{
				Id:         "entity3",
				Type:       storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				Name:       "1.2.3.6",
				Discovered: false,
			},
		},
	}

	result, err := suite.m.GetExternalNetworkPeers(allAllowedCtx, fixtureconsts.Deployment1, "", nil)
	suite.Nil(err)

	protoassert.ElementsMatch(suite.T(), expectedPeers, result)
}

func (suite *ManagerTestSuite) TestAddBaselineAnonymizeDiscoveredExternalSource() {
	discoveredExtSourceID := "discovered-external-src-1"

	mockDeployment := &storage.Deployment{
		Id:        fixtureconsts.Deployment1,
		Name:      "deployment1",
		ClusterId: fixtureconsts.Cluster1,
		Namespace: fixtureconsts.Namespace1,
		PodLabels: map[string]string{"app": "anonymize-test"},
	}

	suite.deploymentDS.EXPECT().GetDeployment(gomock.Any(), fixtureconsts.Deployment1).Return(mockDeployment, true, nil).Times(1)

	mockFlows := []*storage.NetworkFlow{
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					Id:   fixtureconsts.Deployment1,
				},
				DstEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
					Id:   discoveredExtSourceID,
				},
				DstPort:    443,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: protoconv.ConvertMicroTSToProtobufTS(timestamp.Now()),
		},
	}

	suite.clusterFlows.EXPECT().GetFlowStore(gomock.Any(), fixtureconsts.Cluster1).Return(suite.flowStore, nil).Times(1)
	suite.flowStore.EXPECT().GetFlowsForDeployment(gomock.Any(), fixtureconsts.Deployment1, false).Return(mockFlows, nil).Times(1)

	// Note: The 'Discovered' flag on the flow's DstEntity is what triggers anonymization.
	// The tree provides the descriptive info for the original (pre-anonymization) entity.
	discoveredEntityInfoFromTree := &storage.NetworkEntityInfo{
		Id:   discoveredExtSourceID,
		Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
		Desc: &storage.NetworkEntityInfo_ExternalSource_{
			ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
				Name: "8.8.8.8/32",
				Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
					Cidr: "8.8.8.8/32", // Example CIDR
				},
				Discovered: true,
			},
		},
	}

	defaultTree := tree.NewDefaultNetworkTreeWrapper()
	readOnlyTree, err := tree.NewNetworkTreeWrapper([]*storage.NetworkEntityInfo{discoveredEntityInfoFromTree})
	suite.Require().NoError(err)

	suite.treeManager.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), fixtureconsts.Cluster1).Return(readOnlyTree).Times(1)
	suite.treeManager.EXPECT().GetDefaultNetworkTree(gomock.Any()).Return(defaultTree).Times(1)

	suite.connectionManager.EXPECT().SendMessage(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	suite.mustInitManager()
	err = suite.m.CreateNetworkBaseline(fixtureconsts.Deployment1)
	suite.Require().NoError(err)

	baseline, found, err := suite.ds.GetNetworkBaseline(managerCtx, fixtureconsts.Deployment1)
	suite.Require().NoError(err)
	suite.Require().True(found)
	suite.Require().NotNil(baseline)
	suite.Require().NotEmpty(baseline.GetPeers(), "Baseline should have peers")

	var foundAnonymizedPeer bool
	for _, peer := range baseline.GetPeers() {
		info := peer.GetEntity().GetInfo()
		if info.GetId() == networkgraph.InternetExternalSourceID && info.GetType() == storage.NetworkEntityInfo_INTERNET {
			foundAnonymizedPeer = true
			// Check that the name also matches the standard Internet name
			suite.Assert().Equal(networkgraph.InternetExternalSourceID, info.GetId())
			suite.Assert().Equal(uint32(443), peer.GetProperties()[0].GetPort())
			suite.Assert().False(peer.GetProperties()[0].GetIngress()) // Egress from our deployment
			break
		}
	}
	suite.Require().True(foundAnonymizedPeer, "Expected anonymized Internet peer not found in baseline")

	// Also assert that the original discoveredExtSourceID is NOT present as a peer
	var foundOriginalDiscoveredPeer bool
	for _, peer := range baseline.GetPeers() {
		if peer.GetEntity().GetInfo().GetId() == discoveredExtSourceID {
			foundOriginalDiscoveredPeer = true
			break
		}
	}
	suite.Require().False(foundOriginalDiscoveredPeer, "Original discovered external source ID should not be a peer")
}

///// Helper functions to make test code less verbose.

func expectOneTimeCallToConnectionManagerWithBaseline(suite *ManagerTestSuite, baseline *storage.NetworkBaseline) {
	suite.
		connectionManager.
		EXPECT().
		SendMessage(
			baseline.GetClusterId(),
			&central.MsgToSensor{
				Msg: &central.MsgToSensor_NetworkBaselineSync{
					NetworkBaselineSync: &central.NetworkBaselineSync{
						NetworkBaselines: []*storage.NetworkBaseline{baseline},
					},
				},
			}).
		Return(nil)
}

func ctxWithAccessToWrite(id int) context.Context {
	return sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.DeploymentExtension),
		sac.ClusterScopeKeys(clusterID(id)),
		sac.NamespaceScopeKeys(ns(id)),
	))
}

func modifyPeersReq(id int, peers ...*v1.NetworkBaselinePeerStatus) *v1.ModifyBaselineStatusForPeersRequest {
	return &v1.ModifyBaselineStatusForPeersRequest{
		DeploymentId: depID(id),
		Peers:        peers,
	}
}

func protoPeerStatus(status v1.NetworkBaselinePeerStatus_Status, peerID int, port uint32, ingress bool) *v1.NetworkBaselinePeerStatus {
	return &v1.NetworkBaselinePeerStatus{
		Peer: &v1.NetworkBaselineStatusPeer{
			Entity: &v1.NetworkBaselinePeerEntity{
				Id:   depID(peerID),
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
			},
			Port:     port,
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			Ingress:  ingress,
		},
		Status: status,
	}
}

func protoExternalPeerStatus(status v1.NetworkBaselinePeerStatus_Status, peerID int, port uint32, ingress, discovered bool) *v1.NetworkBaselinePeerStatus {
	return &v1.NetworkBaselinePeerStatus{
		Peer: &v1.NetworkBaselineStatusPeer{
			Entity: &v1.NetworkBaselinePeerEntity{
				Id:         extSrcID(peerID),
				Name:       extSrcName(peerID),
				Type:       storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				Discovered: discovered,
			},
			Port:     port,
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			Ingress:  ingress,
		},
		Status: status,
	}
}

func conns(indicators ...networkgraph.NetworkConnIndicator) []networkgraph.NetworkConnIndicator {
	return indicators
}

func emptyBaseline(id int) *storage.NetworkBaseline {
	return &storage.NetworkBaseline{
		DeploymentId:   depID(id),
		ClusterId:      clusterID(id),
		Namespace:      ns(id),
		DeploymentName: depName(id),
	}
}

func baselineWithPeers(id int, peers ...*storage.NetworkBaselinePeer) *storage.NetworkBaseline {
	baseline := emptyBaseline(id)
	baseline.Peers = peers
	return baseline
}

func baselineWithClusterAndPeers(id, _clusterID int, peers ...*storage.NetworkBaselinePeer) *storage.NetworkBaseline {
	baseline := baselineWithPeers(id, peers...)
	baseline.ClusterId = clusterID(_clusterID)
	return baseline
}

func wrapWithForbidden(baseline *storage.NetworkBaseline, peers ...*storage.NetworkBaselinePeer) *storage.NetworkBaseline {
	baseline.ForbiddenPeers = peers
	return baseline
}

func depPeer(id int, properties ...*storage.NetworkBaselineConnectionProperties) *storage.NetworkBaselinePeer {
	return &storage.NetworkBaselinePeer{
		Entity: &storage.NetworkEntity{Info: &storage.NetworkEntityInfo{
			Type: storage.NetworkEntityInfo_DEPLOYMENT,
			Id:   depID(id),
			Desc: &storage.NetworkEntityInfo_Deployment_{
				Deployment: &storage.NetworkEntityInfo_Deployment{
					Name: depName(id),
				},
			},
		}},
		Properties: properties,
	}
}

func extSrcPeer(id int, cidr string, properties ...*storage.NetworkBaselineConnectionProperties) *storage.NetworkBaselinePeer {
	return &storage.NetworkBaselinePeer{
		Entity: &storage.NetworkEntity{
			Info: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				Id:   extSrcID(id),
				Desc: &storage.NetworkEntityInfo_ExternalSource_{
					ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
						Name:   extSrcName(id),
						Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{Cidr: cidr},
					},
				},
			}},
		Properties: properties,
	}
}

func internetPeer(properties ...*storage.NetworkBaselineConnectionProperties) *storage.NetworkBaselinePeer {
	return &storage.NetworkBaselinePeer{
		Entity: &storage.NetworkEntity{
			Info: &storage.NetworkEntityInfo{
				Id:   networkgraph.InternetExternalSourceID,
				Type: storage.NetworkEntityInfo_INTERNET,
			},
		},
		Properties: properties,
	}
}

func properties(ingress bool, port uint32) *storage.NetworkBaselineConnectionProperties {
	return &storage.NetworkBaselineConnectionProperties{
		Ingress:  ingress,
		Port:     port,
		Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
	}
}

func depToDepConn(srcID, dstID int, port uint32) networkgraph.NetworkConnIndicator {
	return networkgraph.NetworkConnIndicator{
		SrcEntity: networkgraph.Entity{
			Type: storage.NetworkEntityInfo_DEPLOYMENT,
			ID:   depID(srcID),
		},
		DstEntity: networkgraph.Entity{
			Type: storage.NetworkEntityInfo_DEPLOYMENT,
			ID:   depID(dstID),
		},
		Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		DstPort:  port,
	}
}

func getNetworkPolicy(matchLabels map[string]string) *storage.NetworkPolicy {
	networkPolicy := fixtures.GetNetworkPolicy()
	networkPolicy.Spec.PodSelector = &storage.LabelSelector{MatchLabels: matchLabels}
	// Add some ingress egress rule
	networkPolicy.Spec.Ingress = append(networkPolicy.Spec.Ingress, &storage.NetworkPolicyIngressRule{
		Ports: []*storage.NetworkPolicyPort{
			{
				Protocol: storage.Protocol_TCP_PROTOCOL,
				PortRef:  &storage.NetworkPolicyPort_Port{Port: 80},
			},
		},
		From: []*storage.NetworkPolicyPeer{
			{
				PodSelector: &storage.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
			},
		},
	})
	networkPolicy.Spec.Egress = append(networkPolicy.Spec.Egress, &storage.NetworkPolicyEgressRule{
		Ports: []*storage.NetworkPolicyPort{
			{
				Protocol: storage.Protocol_TCP_PROTOCOL,
				PortRef:  &storage.NetworkPolicyPort_Port{Port: 443},
			},
		},
		To: []*storage.NetworkPolicyPeer{
			{
				PodSelector: &storage.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
			},
		},
	})
	return networkPolicy
}
