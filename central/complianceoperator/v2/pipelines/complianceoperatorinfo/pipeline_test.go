package complianceoperatorinfo

import (
	"context"
	"fmt"
	"testing"

	"github.com/Masterminds/semver/v3"
	managerMocks "github.com/stackrox/rox/central/complianceoperator/v2/compliancemanager/mocks"
	coIntegrationMocks "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestPipeline(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite
	pipeline        *pipelineImpl
	manager         *managerMocks.MockManager
	coIntegrationDS *coIntegrationMocks.MockDataStore
	mockCtrl        *gomock.Controller
}

func (suite *PipelineTestSuite) SetupSuite() {
	suite.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		suite.T().Skip("Skip tests when ComplianceEnhancements disabled")
		suite.T().SkipNow()
	}
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.manager = managerMocks.NewMockManager(suite.mockCtrl)
	suite.coIntegrationDS = coIntegrationMocks.NewMockDataStore(suite.mockCtrl)
	suite.pipeline = NewPipeline(suite.manager).(*pipelineImpl)
}

func (suite *PipelineTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *PipelineTestSuite) TestComplianceInfoMsgFromSensor() {
	complianceMsg := &central.ComplianceOperatorInfo{
		Version:   minimalComplianceOperatorVersion,
		Namespace: fixtureconsts.Namespace1,
		TotalDesiredPodsOpt: &central.ComplianceOperatorInfo_TotalDesiredPods{
			TotalDesiredPods: 5,
		},
		TotalReadyPodsOpt: &central.ComplianceOperatorInfo_TotalReadyPods{
			TotalReadyPods: 2,
		},
	}

	statusErrors := []string{"compliance operator not ready. Only 2 pods are ready when 5 are desired."}
	expectedInfo := &storage.ComplianceIntegration{
		Version:             minimalComplianceOperatorVersion,
		ClusterId:           fixtureconsts.Cluster1,
		ComplianceNamespace: fixtureconsts.Namespace1,
		StatusErrors:        statusErrors,
		OperatorStatus:      storage.COStatus_UNHEALTHY,
	}

	suite.manager.EXPECT().ProcessComplianceOperatorInfo(gomock.Any(), expectedInfo).Return(nil).Times(1)

	err := suite.pipeline.Run(context.Background(), fixtureconsts.Cluster1, &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_ComplianceOperatorInfo{
			ComplianceOperatorInfo: complianceMsg,
		},
	}, nil)
	suite.NoError(err)
}

func (suite *PipelineTestSuite) TestComplianceInfoMinimalRequiredVersion() {
	minVersion, err := semver.NewVersion(minimalComplianceOperatorVersion)
	suite.NoError(err)
	suite.NotNil(minVersion)

	testCases := map[string]struct {
		version        string
		expectedStatus storage.COStatus
		expectedErrors []string
	}{
		"invalid version": {
			version:        "not-valid",
			expectedStatus: storage.COStatus_UNHEALTHY,
			expectedErrors: []string{"invalid compliance operator version \"not-valid\""},
		},
		"older version": {
			version:        "v1.5.1",
			expectedStatus: storage.COStatus_UNHEALTHY,
			expectedErrors: []string{fmt.Sprintf("compliance operator version \"v1.5.1\" is not supported. Minimal required version is %q", minimalComplianceOperatorVersion)},
		},
		"min version": {
			version:        minimalComplianceOperatorVersion,
			expectedStatus: storage.COStatus_HEALTHY,
		},
		"newer patch version": {
			version:        minVersion.IncPatch().String(),
			expectedStatus: storage.COStatus_HEALTHY,
		},
		"newer minor version": {
			version:        minVersion.IncMinor().String(),
			expectedStatus: storage.COStatus_HEALTHY,
		},
		"newer major version": {
			version:        minVersion.IncMajor().String(),
			expectedStatus: storage.COStatus_HEALTHY,
		},
		"older version without prefix": {
			version:        "1.5.1",
			expectedStatus: storage.COStatus_UNHEALTHY,
			expectedErrors: []string{fmt.Sprintf("compliance operator version \"1.5.1\" is not supported. Minimal required version is %q", minimalComplianceOperatorVersion)},
		},
		"newer version without prefix": {
			version:        "99.99.99",
			expectedStatus: storage.COStatus_HEALTHY,
		},
	}

	for name, tc := range testCases {
		suite.T().Run(name, func(tt *testing.T) {
			complianceMsg := &central.ComplianceOperatorInfo{
				Version:   tc.version,
				Namespace: fixtureconsts.Namespace1,
				TotalDesiredPodsOpt: &central.ComplianceOperatorInfo_TotalDesiredPods{
					TotalDesiredPods: 1,
				},
				TotalReadyPodsOpt: &central.ComplianceOperatorInfo_TotalReadyPods{
					TotalReadyPods: 1,
				},
			}

			expectedInfo := &storage.ComplianceIntegration{
				Version:             tc.version,
				ClusterId:           fixtureconsts.Cluster1,
				ComplianceNamespace: fixtureconsts.Namespace1,
				StatusErrors:        tc.expectedErrors,
				OperatorStatus:      tc.expectedStatus,
			}

			suite.manager.EXPECT().ProcessComplianceOperatorInfo(gomock.Any(), expectedInfo).Return(nil).Times(1)
			err := suite.pipeline.Run(context.Background(), fixtureconsts.Cluster1, &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_ComplianceOperatorInfo{
					ComplianceOperatorInfo: complianceMsg,
				},
			}, nil)
			suite.NoError(err)
		})
	}
}
