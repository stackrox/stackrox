package compliancemanager

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type pipelineTestCase struct {
	desc                         string
	setMocksAndGetComplianceInfo func()
	complianceInfoGen            func() *storage.ComplianceIntegration
	isErrorTest                  bool
}

func TestComplianceManager(t *testing.T) {
	suite.Run(t, new(complianceManagerTestSuite))
}

type complianceManagerTestSuite struct {
	suite.Suite

	hasWriteCtx context.Context
	noAccessCtx context.Context

	mockCtrl      *gomock.Controller
	integrationDS *mocks.MockDataStore
	manager       Manager
}

func (suite *complianceManagerTestSuite) SetupSuite() {
	suite.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		suite.T().Skip("Skip tests when ComplianceEnhancements disabled")
		suite.T().SkipNow()
	}
}

func (suite *complianceManagerTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))
	suite.noAccessCtx = sac.WithNoAccess(context.Background())

	suite.integrationDS = mocks.NewMockDataStore(suite.mockCtrl)
	suite.manager = New(nil, suite.integrationDS)
}

func (suite *complianceManagerTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *complianceManagerTestSuite) TestProcessComplianceOperatorInfo() {
	cases := []pipelineTestCase{
		{
			desc: "Error retrieving data",
			setMocksAndGetComplianceInfo: func() {
				query := search.NewQueryBuilder().
					AddExactMatches(search.ClusterID, fixtureconsts.Cluster1).ProtoQuery()

				suite.integrationDS.EXPECT().GetComplianceIntegrations(gomock.Any(), query).Return(nil, errors.New("Unable to retrieve data")).Times(1)
			},
			complianceInfoGen: func() *storage.ComplianceIntegration {
				return &storage.ComplianceIntegration{
					Version:   "22",
					ClusterId: fixtureconsts.Cluster1,
					Namespace: fixtureconsts.Namespace1,
				}
			},
			isErrorTest: true,
		},
		{
			desc: "Add integration",
			setMocksAndGetComplianceInfo: func() {
				query := search.NewQueryBuilder().
					AddExactMatches(search.ClusterID, fixtureconsts.Cluster1).ProtoQuery()

				suite.integrationDS.EXPECT().GetComplianceIntegrations(gomock.Any(), query).Return(nil, nil).Times(1)

				expectedInfo := &storage.ComplianceIntegration{
					Version:   "22",
					ClusterId: fixtureconsts.Cluster1,
					Namespace: fixtureconsts.Namespace1,
				}
				suite.integrationDS.EXPECT().AddComplianceIntegration(gomock.Any(), expectedInfo).Return(uuid.NewV4().String(), nil).Times(1)
			},
			complianceInfoGen: func() *storage.ComplianceIntegration {
				return &storage.ComplianceIntegration{
					Version:   "22",
					ClusterId: fixtureconsts.Cluster1,
					Namespace: fixtureconsts.Namespace1,
				}
			},
			isErrorTest: false,
		},
		{
			desc: "Update integration",
			setMocksAndGetComplianceInfo: func() {
				query := search.NewQueryBuilder().
					AddExactMatches(search.ClusterID, fixtureconsts.Cluster1).ProtoQuery()

				expectedInfo := &storage.ComplianceIntegration{
					Id:        uuid.NewV4().String(),
					Version:   "22",
					ClusterId: fixtureconsts.Cluster1,
					Namespace: fixtureconsts.Namespace1,
				}

				suite.integrationDS.EXPECT().GetComplianceIntegrations(gomock.Any(), query).Return([]*storage.ComplianceIntegration{expectedInfo}, nil).Times(1)

				suite.integrationDS.EXPECT().UpdateComplianceIntegration(gomock.Any(), expectedInfo).Return(nil).Times(1)
			},
			complianceInfoGen: func() *storage.ComplianceIntegration {
				return &storage.ComplianceIntegration{
					Version:   "22",
					ClusterId: fixtureconsts.Cluster1,
					Namespace: fixtureconsts.Namespace1,
				}
			},
			isErrorTest: false,
		},
	}

	for _, tc := range cases {
		suite.T().Run(tc.desc, func(t *testing.T) {
			// Setup the mock calls for this case
			tc.setMocksAndGetComplianceInfo()

			err := suite.manager.ProcessComplianceOperatorInfo(suite.hasWriteCtx, tc.complianceInfoGen())
			if tc.isErrorTest {
				suite.Require().NotNil(err)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}
