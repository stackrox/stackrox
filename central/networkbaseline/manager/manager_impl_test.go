package manager

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/central/networkbaseline/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stretchr/testify/suite"
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

///// Helper functions to make test code less verbose.

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

func depPeer(id int, properties ...*storage.NetworkBaselineConnectionProperties) *storage.NetworkBaselinePeer {
	return &storage.NetworkBaselinePeer{
		Entity: &storage.NetworkEntity{Info: &storage.NetworkEntityInfo{
			Type: storage.NetworkEntityInfo_DEPLOYMENT,
			Id:   depID(id),
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
