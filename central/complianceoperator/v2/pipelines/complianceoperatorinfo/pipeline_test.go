package complianceoperatorinfo

import (
	"context"
	"fmt"
	"os"
	"testing"

	managerMocks "github.com/stackrox/rox/central/complianceoperator/v2/compliancemanager/mocks"
	coIntegrationMocks "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stretchr/testify/assert"
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
		Version:   env.ComplianceMinimalSupportedVersion.VersionSetting().String(),
		Namespace: fixtureconsts.Namespace1,
		TotalDesiredPodsOpt: &central.ComplianceOperatorInfo_TotalDesiredPods{
			TotalDesiredPods: 5,
		},
		TotalReadyPodsOpt: &central.ComplianceOperatorInfo_TotalReadyPods{
			TotalReadyPods: 2,
		},
	}

	statusErrors := []string{"Compliance operator is not ready. Only 2 pods out of desired 5 are ready."}
	expectedInfo := &storage.ComplianceIntegration{
		Version:             env.ComplianceMinimalSupportedVersion.VersionSetting().String(),
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
	defaultVersion := env.ComplianceMinimalSupportedVersion.DefaultValue()

	patchVersion := defaultVersion.IncPatch()
	minorVersion := defaultVersion.IncMinor()
	majorVersion := defaultVersion.IncMajor()

	testCases := map[string]struct {
		operatorVersion string
		envVersion      string
		expectedStatus  storage.COStatus
		expectedErrors  []string
	}{
		"invalid operatorVersion": {
			operatorVersion: "not-valid",
			expectedStatus:  storage.COStatus_UNHEALTHY,
			expectedErrors:  []string{"The installed compliance operator version \"not-valid\" is invalid."},
		},
		"older operatorVersion": {
			operatorVersion: "v1.5.1",
			expectedStatus:  storage.COStatus_UNHEALTHY,
			expectedErrors:  []string{fmt.Sprintf("The installed compliance operator version \"1.5.1\" is unsupported. The minimum required version is %q.", defaultVersion.String())},
		},
		"min operatorVersion": {
			operatorVersion: defaultVersion.String(),
			expectedStatus:  storage.COStatus_HEALTHY,
		},
		"newer patch operatorVersion": {
			operatorVersion: patchVersion.String(),
			expectedStatus:  storage.COStatus_HEALTHY,
		},
		"newer minor operatorVersion": {
			operatorVersion: minorVersion.String(),
			expectedStatus:  storage.COStatus_HEALTHY,
		},
		"newer major operatorVersion": {
			operatorVersion: majorVersion.String(),
			expectedStatus:  storage.COStatus_HEALTHY,
		},
		"older operatorVersion without prefix": {
			operatorVersion: "1.5.1",
			expectedStatus:  storage.COStatus_UNHEALTHY,
			expectedErrors:  []string{fmt.Sprintf("The installed compliance operator version \"1.5.1\" is unsupported. The minimum required version is %q.", defaultVersion.String())},
		},
		"newer operatorVersion without prefix": {
			operatorVersion: "99.99.99",
			expectedStatus:  storage.COStatus_HEALTHY,
		},
		"invalid env version": {
			operatorVersion: "v1.2.0",
			envVersion:      "not-valid",
			expectedStatus:  storage.COStatus_UNHEALTHY,
			expectedErrors:  []string{fmt.Sprintf("The installed compliance operator version \"1.2.0\" is unsupported. The minimum required version is %q.", defaultVersion.String())},
		},
		"newer operatorVersion from env version": {
			operatorVersion: "v2.2.0",
			envVersion:      "v2.1.0",
			expectedStatus:  storage.COStatus_HEALTHY,
		},
		"older operatorVersion from env version": {
			operatorVersion: "v2.1.0",
			envVersion:      "v2.2.0",
			expectedStatus:  storage.COStatus_UNHEALTHY,
			expectedErrors:  []string{"The installed compliance operator version \"2.1.0\" is unsupported. The minimum required version is \"2.2.0\"."},
		},
		"env version below minimum uses default version": {
			operatorVersion: "v1.2.0",
			envVersion:      "v1.2.0",
			expectedStatus:  storage.COStatus_UNHEALTHY,
			expectedErrors:  []string{fmt.Sprintf("The installed compliance operator version \"1.2.0\" is unsupported. The minimum required version is %q.", defaultVersion.String())},
		},
	}

	for name, tc := range testCases {
		suite.T().Run(name, func(tt *testing.T) {
			assert.NoError(tt, os.Setenv("ROX_COMPLIANCE_MINIMAL_SUPPORTED_OPERATOR_VERSION", tc.envVersion))

			complianceMsg := &central.ComplianceOperatorInfo{
				Version:   tc.operatorVersion,
				Namespace: fixtureconsts.Namespace1,
				TotalDesiredPodsOpt: &central.ComplianceOperatorInfo_TotalDesiredPods{
					TotalDesiredPods: 1,
				},
				TotalReadyPodsOpt: &central.ComplianceOperatorInfo_TotalReadyPods{
					TotalReadyPods: 1,
				},
			}

			expectedInfo := &storage.ComplianceIntegration{
				Version:             tc.operatorVersion,
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
