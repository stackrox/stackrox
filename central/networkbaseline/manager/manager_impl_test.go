package manager

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/central/networkbaseline/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stretchr/testify/suite"
)

var (
	allAllowedCtx = sac.WithAllAccess(context.Background())
)

type fakeDS struct {
	baselines map[string]*storage.NetworkBaseline

	// All methods not overriden in this struct will panic.
	datastore.DataStore
}

func (f *fakeDS) UpsertNetworkBaselines(ctx context.Context, baselines []*storage.NetworkBaseline) error {
	for _, baseline := range baselines {
		f.baselines[baseline.GetDeploymentId()] = baseline
	}
	return nil
}

func (f *fakeDS) Walk(ctx context.Context, fn func(baseline *storage.NetworkBaseline) error) error {
	for _, baseline := range f.baselines {
		if err := fn(baseline); err != nil {
			return err
		}
	}
	return nil
}

func (f *fakeDS) DeleteNetworkBaseline(ctx context.Context, deploymentID string) error {
	delete(f.baselines, deploymentID)
	return nil
}

func TestManager(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}

type ManagerTestSuite struct {
	suite.Suite

	ds            *fakeDS
	m             Manager
	currTestStart timestamp.MicroTS
}

func (suite *ManagerTestSuite) SetupTest() {
	suite.currTestStart = timestamp.Now()
}

func (suite *ManagerTestSuite) TearDownTest() {
}

func (suite *ManagerTestSuite) mustInitManager(initialBaselines ...*storage.NetworkBaseline) {
	suite.ds = &fakeDS{baselines: make(map[string]*storage.NetworkBaseline)}
	for _, baseline := range initialBaselines {
		baseline.ObservationPeriodEnd = timestamp.Now().Add(env.NetworkBaselineObservationPeriod.DurationSetting()).GogoProtobuf()
		suite.ds.baselines[baseline.GetDeploymentId()] = baseline
	}
	var err error
	suite.m, err = New(suite.ds)
	suite.Require().NoError(err)
}

func depID(id int) string {
	return fmt.Sprintf("DEP%03d", id)
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

func (suite *ManagerTestSuite) initBaselinesForDeployments(ids ...int) {
	for _, id := range ids {
		suite.Require().NoError(suite.m.ProcessDeploymentCreate(depID(id), clusterID(id), ns(id)))
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
		cloned := baseline.Clone()
		actualObsEnd := timestamp.FromProtobuf(cloned.GetObservationPeriodEnd())
		suite.True(actualObsEnd.After(obsPeriodStart), "Actual obs end: %v, expected obs window: %v-%v", actualObsEnd.GoTime(), obsPeriodStart.GoTime(), obsPeriodEnd.GoTime())
		suite.True(obsPeriodEnd.After(actualObsEnd), "Actual obs end: %v, expected obs window: %v-%v", actualObsEnd.GoTime(), obsPeriodStart.GoTime(), obsPeriodEnd.GoTime())
		cloned.ObservationPeriodEnd = nil
		baselinesWithoutObsPeriod = append(baselinesWithoutObsPeriod, cloned)
	}
	suite.ElementsMatch(baselinesWithoutObsPeriod, baselines)
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
	suite.processFlowUpdate([]networkgraph.NetworkConnIndicator{
		// This conn is valid and should get incorporated into the baseline.
		{
			SrcEntity: networkgraph.Entity{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				ID:   depID(1),
			},
			DstEntity: networkgraph.Entity{
				Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				ID:   "EXTERNALENTITYID",
			},
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			DstPort:  1,
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
					Id:   "EXTERNALENTITYID",
				}},
				Properties: []*storage.NetworkBaselineConnectionProperties{properties(false, 1)},
			},
			&storage.NetworkBaselinePeer{
				Entity: &storage.NetworkEntity{Info: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_INTERNET,
					Id:   "INTERNETTZZ",
				}},
				Properties: []*storage.NetworkBaselineConnectionProperties{properties(false, 13)},
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
			Peer: &v1.NetworkBaselinePeer{
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
}

func (suite *ManagerTestSuite) TestDeploymentDelete() {
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

func (suite *ManagerTestSuite) TestDeleteWithExtSrcPeer() {
	suite.mustInitManager(
		baselineWithPeers(
			1,
			depPeer(2, properties(false, 52)),
			extSrcPeer(3, properties(false, 443)),
		),
		baselineWithPeers(2, depPeer(1, properties(true, 52))),
	)
	suite.assertBaselinesAre(
		baselineWithPeers(
			1,
			depPeer(2, properties(false, 52)),
			extSrcPeer(3, properties(false, 443)),
		),
		baselineWithPeers(2, depPeer(1, properties(true, 52))),
	)
	// Make sure deleting a deployment does not trigger an error even if we have ext src peer
	suite.Nil(suite.m.ProcessDeploymentDelete(depID(2)))
	suite.assertBaselinesAre(baselineWithPeers(1, extSrcPeer(3, properties(false, 443))))
}

///// Helper functions to make test code less verbose.

func ctxWithAccessToWrite(id int) context.Context {
	return sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.NetworkBaseline),
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
		Peer: &v1.NetworkBaselinePeer{
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

func conns(indicators ...networkgraph.NetworkConnIndicator) []networkgraph.NetworkConnIndicator {
	return indicators
}

func emptyBaseline(id int) *storage.NetworkBaseline {
	return &storage.NetworkBaseline{
		DeploymentId: depID(id),
		ClusterId:    clusterID(id),
		Namespace:    ns(id),
	}
}

func baselineWithPeers(id int, peers ...*storage.NetworkBaselinePeer) *storage.NetworkBaseline {
	baseline := emptyBaseline(id)
	baseline.Peers = peers
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
		}},
		Properties: properties,
	}
}

func extSrcPeer(id int, properties ...*storage.NetworkBaselineConnectionProperties) *storage.NetworkBaselinePeer {
	return &storage.NetworkBaselinePeer{
		Entity: &storage.NetworkEntity{
			Info: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				Id:   extSrcID(id),
			}},
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
