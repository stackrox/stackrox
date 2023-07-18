package replay

import (
	"testing"

	centralDebug "github.com/stackrox/rox/sensor/debugger/central"
	"github.com/stackrox/rox/sensor/debugger/k8s"
	"github.com/stackrox/rox/sensor/tests/replay"
	"github.com/stretchr/testify/suite"
)

// We need a test file per events file (replay_resources_test and replay_alerts_test).
// Before we had these two in a table test but, since both were using the same sensor's
// instance, there was some flakiness due to problems cleaning up sensor's state.
func TestReplayResourceEvents(t *testing.T) {
	suite.Run(t, new(ReplayResourcesSuite))
}

type ReplayResourcesSuite struct {
	suite.Suite
	fakeClient  *k8s.ClientSet
	fakeCentral *centralDebug.FakeService
}

var _ suite.SetupAllSuite = (*ReplayResourcesSuite)(nil)
var _ replay.Suite = (*ReplayResourcesSuite)(nil)

func (suite *ReplayResourcesSuite) SetupSuite() {
	replay.SetupTest(suite)
}

func (suite *ReplayResourcesSuite) GetFakeClient() *k8s.ClientSet {
	return suite.fakeClient
}
func (suite *ReplayResourcesSuite) SetFakeClient(cl *k8s.ClientSet) {
	suite.fakeClient = cl
}

func (suite *ReplayResourcesSuite) GetFakeCentral() *centralDebug.FakeService {
	return suite.fakeCentral
}

func (suite *ReplayResourcesSuite) SetFakeCentral(central *centralDebug.FakeService) {
	suite.fakeCentral = central
}

func (suite *ReplayResourcesSuite) GetT() *testing.T {
	return suite.T()
}

func (suite *ReplayResourcesSuite) Test_ReplayEvents() {
	suite.T().Skipf("Replay tests disabled")
	writer := replay.StartTest(suite)
	defer writer.Close()

	k8sEventsFile := "../data/safety-net-resources-k8s-trace.jsonl"
	sensorOutputFile := "../data/safety-net-resources-central-out.bin"

	suite.T().Run("Safety net test: Resources", func(t *testing.T) {
		replay.RunReplayTest(t, suite, writer, k8sEventsFile, sensorOutputFile)
	})
}
