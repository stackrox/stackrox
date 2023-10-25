package postgres

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
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

	plop1 := storage.ProcessListeningOnPort{
		Endpoint: &storage.ProcessListeningOnPort_Endpoint{
			Port:     80,
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
		DeploymentId: fixtureconsts.Deployment1,
		PodId:        fixtureconsts.PodName1,
		PodUid:       fixtureconsts.PodUID1,
		Signal: &storage.ProcessSignal{
			ExecFilePath: execFilePath1,
		},
	}

	plop2 := storage.ProcessListeningOnPort{
		Endpoint: &storage.ProcessListeningOnPort_Endpoint{
			Port:     1234,
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
		DeploymentId: fixtureconsts.Deployment1,
		PodId:        fixtureconsts.PodName1,
		PodUid:       fixtureconsts.PodUID1,
		Signal: &storage.ProcessSignal{
			ExecFilePath: execFilePath1,
		},
	}

	plop3 := storage.ProcessListeningOnPort{
		Endpoint: &storage.ProcessListeningOnPort_Endpoint{
			Port:     1234,
			Protocol: storage.L4Protocol_L4_PROTOCOL_UDP,
		},
		DeploymentId: fixtureconsts.Deployment1,
		PodId:        fixtureconsts.PodName1,
		PodUid:       fixtureconsts.PodUID1,
		Signal: &storage.ProcessSignal{
			ExecFilePath: execFilePath1,
		},
	}

	plop4 := storage.ProcessListeningOnPort{
		Endpoint: &storage.ProcessListeningOnPort_Endpoint{
			Port:     1234,
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
		DeploymentId: fixtureconsts.Deployment1,
		PodId:        fixtureconsts.PodName1,
		PodUid:       fixtureconsts.PodUID1,
		Signal: &storage.ProcessSignal{
			ExecFilePath: execFilePath2,
		},
	}

	plop5 := storage.ProcessListeningOnPort{
		Endpoint: &storage.ProcessListeningOnPort_Endpoint{
			Port:     1234,
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
		DeploymentId: fixtureconsts.Deployment1,
		PodId:        fixtureconsts.PodName2,
		PodUid:       fixtureconsts.PodUID2,
		Signal: &storage.ProcessSignal{
			ExecFilePath: execFilePath1,
		},
	}

	plops := []*storage.ProcessListeningOnPort{&plop3, &plop5, &plop1, &plop2, &plop4}

	sortPlops(plops)

	expectedSortedPlops := []*storage.ProcessListeningOnPort{&plop1, &plop2, &plop3, &plop4, &plop5}

	suite.Equal(expectedSortedPlops, plops)
}
