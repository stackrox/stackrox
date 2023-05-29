package replay

import (
	"testing"

	centralDebug "github.com/stackrox/rox/sensor/debugger/central"
	"github.com/stackrox/rox/sensor/debugger/k8s"
	"github.com/stackrox/rox/sensor/tests/replay"
	"github.com/stretchr/testify/suite"
)

func TestReplayAlertEvents(t *testing.T) {
	suite.Run(t, new(ReplayAlertsSuite))
}

type ReplayAlertsSuite struct {
	suite.Suite
	fakeClient  *k8s.ClientSet
	fakeCentral *centralDebug.FakeService
}

var _ suite.SetupAllSuite = (*ReplayAlertsSuite)(nil)
var _ replay.Suite = (*ReplayAlertsSuite)(nil)

func (suite *ReplayAlertsSuite) SetupSuite() {
	replay.SetupTest(suite)
}

func (suite *ReplayAlertsSuite) GetFakeClient() *k8s.ClientSet {
	return suite.fakeClient
}
func (suite *ReplayAlertsSuite) SetFakeClient(cl *k8s.ClientSet) {
	suite.fakeClient = cl
}

func (suite *ReplayAlertsSuite) GetFakeCentral() *centralDebug.FakeService {
	return suite.fakeCentral
}

func (suite *ReplayAlertsSuite) SetFakeCentral(central *centralDebug.FakeService) {
	suite.fakeCentral = central
}

func (suite *ReplayAlertsSuite) GetT() *testing.T {
	return suite.T()
}

func (suite *ReplayAlertsSuite) Test_ReplayEvents() {
	writer := replay.StartTest(suite)
	defer writer.Close()

	k8sEventsFile := "../data/safety-net-alerts-k8s-trace.jsonl"
	sensorOutputFile := "../data/safety-net-alerts-central-out.bin"

	suite.T().Run("Safety net test: Alerts", func(t *testing.T) {
		replay.RunReplayTest(t, suite, writer, k8sEventsFile, sensorOutputFile)
	})
}
