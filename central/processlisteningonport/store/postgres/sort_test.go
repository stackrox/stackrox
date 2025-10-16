package postgres

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/suite"
)

type SortSuite struct {
	suite.Suite
}

func TestSortSuite(t *testing.T) {
	suite.Run(t, new(SortSuite))
}

func (suite *SortSuite) TestSortVarious() {

	execFilePath1 := "app"
	execFilePath2 := "zap"

	pe := &storage.ProcessListeningOnPort_Endpoint{}
	pe.SetPort(80)
	pe.SetProtocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	ps := &storage.ProcessSignal{}
	ps.SetExecFilePath(execFilePath1)
	plop1 := &storage.ProcessListeningOnPort{}
	plop1.SetEndpoint(pe)
	plop1.SetDeploymentId(fixtureconsts.Deployment1)
	plop1.SetPodId(fixtureconsts.PodName1)
	plop1.SetPodUid(fixtureconsts.PodUID1)
	plop1.SetSignal(ps)

	pe2 := &storage.ProcessListeningOnPort_Endpoint{}
	pe2.SetPort(1234)
	pe2.SetProtocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	ps2 := &storage.ProcessSignal{}
	ps2.SetExecFilePath(execFilePath1)
	plop2 := &storage.ProcessListeningOnPort{}
	plop2.SetEndpoint(pe2)
	plop2.SetDeploymentId(fixtureconsts.Deployment1)
	plop2.SetPodId(fixtureconsts.PodName1)
	plop2.SetPodUid(fixtureconsts.PodUID1)
	plop2.SetSignal(ps2)

	pe3 := &storage.ProcessListeningOnPort_Endpoint{}
	pe3.SetPort(1234)
	pe3.SetProtocol(storage.L4Protocol_L4_PROTOCOL_UDP)
	ps3 := &storage.ProcessSignal{}
	ps3.SetExecFilePath(execFilePath1)
	plop3 := &storage.ProcessListeningOnPort{}
	plop3.SetEndpoint(pe3)
	plop3.SetDeploymentId(fixtureconsts.Deployment1)
	plop3.SetPodId(fixtureconsts.PodName1)
	plop3.SetPodUid(fixtureconsts.PodUID1)
	plop3.SetSignal(ps3)

	pe4 := &storage.ProcessListeningOnPort_Endpoint{}
	pe4.SetPort(1234)
	pe4.SetProtocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	ps4 := &storage.ProcessSignal{}
	ps4.SetExecFilePath(execFilePath2)
	plop4 := &storage.ProcessListeningOnPort{}
	plop4.SetEndpoint(pe4)
	plop4.SetDeploymentId(fixtureconsts.Deployment1)
	plop4.SetPodId(fixtureconsts.PodName1)
	plop4.SetPodUid(fixtureconsts.PodUID1)
	plop4.SetSignal(ps4)

	pe5 := &storage.ProcessListeningOnPort_Endpoint{}
	pe5.SetPort(1234)
	pe5.SetProtocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	plop5 := &storage.ProcessListeningOnPort{}
	plop5.SetEndpoint(pe5)
	plop5.SetDeploymentId(fixtureconsts.Deployment1)
	plop5.SetPodId(fixtureconsts.PodName2)
	plop5.SetPodUid(fixtureconsts.PodUID2)

	ps5 := &storage.ProcessSignal{}
	ps5.SetExecFilePath(execFilePath1)
	plop6 := &storage.ProcessListeningOnPort{}
	plop6.SetDeploymentId(fixtureconsts.Deployment1)
	plop6.SetPodId(fixtureconsts.PodName2)
	plop6.SetPodUid(fixtureconsts.PodUID2)
	plop6.SetSignal(ps5)

	pe6 := &storage.ProcessListeningOnPort_Endpoint{}
	pe6.SetPort(1234)
	pe6.SetProtocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	ps6 := &storage.ProcessSignal{}
	ps6.SetExecFilePath(execFilePath1)
	plop7 := &storage.ProcessListeningOnPort{}
	plop7.SetEndpoint(pe6)
	plop7.SetDeploymentId(fixtureconsts.Deployment1)
	plop7.SetPodId(fixtureconsts.PodName2)
	plop7.SetPodUid(fixtureconsts.PodUID2)
	plop7.SetSignal(ps6)

	plops := []*storage.ProcessListeningOnPort{&plop3, &plop7, &plop5, &plop1, &plop6, &plop2, &plop4}

	sortPlops(plops)

	expectedSortedPlops := []*storage.ProcessListeningOnPort{&plop1, &plop2, &plop3, &plop4, &plop5, &plop6, &plop7}

	protoassert.SlicesEqual(suite.T(), expectedSortedPlops, plops)
}
