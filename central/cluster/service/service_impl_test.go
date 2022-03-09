package service

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/cluster/datastore"
	datastoreMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	probeSourcesMocks "github.com/stackrox/rox/central/probesources/mocks"
	"github.com/stackrox/rox/pkg/buildinfo/testbuildinfo"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/suite"
)

func TestClusterService(t *testing.T) {
	suite.Run(t, new(ClusterServiceTestSuite))
}

type ClusterServiceTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller
	ei       *envisolator.EnvIsolator

	dataStore datastore.DataStore
}

var _ suite.TearDownTestSuite = (*ClusterServiceTestSuite)(nil)

func (suite *ClusterServiceTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.dataStore = datastoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.ei = envisolator.NewEnvIsolator(suite.T())

	suite.ei.Setenv("ROX_IMAGE_FLAVOR", "rhacs")
	testbuildinfo.SetForTest(suite.T())
	testutils.SetExampleVersion(suite.T())
}

func (suite *ClusterServiceTestSuite) TearDownTest() {
	suite.ei.RestoreAll()
	suite.mockCtrl.Finish()
}

func (suite *ClusterServiceTestSuite) TestGetClusterDefaults() {

	cases := map[string]struct {
		kernelSupportAvailable bool
	}{
		"No kernel suppport": {
			kernelSupportAvailable: false,
		},
		"With kernel suppport": {
			kernelSupportAvailable: true,
		},
	}
	flavor := defaults.GetImageFlavorFromEnv()
	for name, testCase := range cases {
		suite.Run(name, func() {
			ps := probeSourcesMocks.NewMockProbeSources(suite.mockCtrl)
			ps.EXPECT().AnyAvailable(gomock.Any()).Times(1).Return(testCase.kernelSupportAvailable, nil)
			clusterService := New(suite.dataStore, nil, ps)

			defaults, err := clusterService.GetClusterDefaultValues(context.Background(), nil)
			suite.NoError(err)
			suite.Equal(flavor.MainImageNoTag(), defaults.GetMainImageRepository())
			suite.Equal(flavor.CollectorFullImageNoTag(), defaults.GetCollectorImageRepository())
			suite.Equal(testCase.kernelSupportAvailable, defaults.GetKernelSupportAvailable())
		})
	}
}
