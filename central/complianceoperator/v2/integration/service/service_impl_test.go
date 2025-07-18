package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	"github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	mockClusterName = "mock-cluster"
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestComplianceIntegrationService(t *testing.T) {
	suite.Run(t, new(ComplianceIntegrationServiceTestSuite))
}

type ComplianceIntegrationServiceTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	ctx                            context.Context
	complianceIntegrationDataStore *mocks.MockDataStore
	clusterDatastore               *clusterMocks.MockDataStore
	service                        Service
}

func (s *ComplianceIntegrationServiceTestSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		s.T().Skip("Skip test when compliance enhancements are disabled")
		s.T().SkipNow()
	}

	s.mockCtrl = gomock.NewController(s.T())
	s.ctx = sac.WithAllAccess(context.Background())
	s.clusterDatastore = clusterMocks.NewMockDataStore(s.mockCtrl)
	s.complianceIntegrationDataStore = mocks.NewMockDataStore(s.mockCtrl)
	s.service = New(s.complianceIntegrationDataStore, s.clusterDatastore)
}

type sensorHealthStateReturn struct {
	healthStatus storage.ClusterHealthStatus_HealthStatusLabel
	found        bool
	err          error
}

func (s *ComplianceIntegrationServiceTestSuite) TestListComplianceIntegrations() {
	allAccessContext := sac.WithAllAccess(context.Background())
	testCases := []struct {
		desc              string
		query             *apiV2.RawQuery
		expectedQ         *v1.Query
		expectedCountQ    *v1.Query
		sensorHealthState sensorHealthStateReturn
	}{
		{
			desc:           "Empty query",
			query:          &apiV2.RawQuery{Query: ""},
			expectedQ:      search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
			expectedCountQ: search.EmptyQuery(),
			sensorHealthState: sensorHealthStateReturn{
				healthStatus: storage.ClusterHealthStatus_HEALTHY,
				found:        true,
				err:          nil,
			},
		},
		{
			desc:  "Query with search field",
			query: &apiV2.RawQuery{Query: "Cluster ID:id"},
			expectedQ: search.NewQueryBuilder().AddStrings(search.ClusterID, "id").
				WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
			expectedCountQ: search.NewQueryBuilder().AddStrings(search.ClusterID, "id").ProtoQuery(),
			sensorHealthState: sensorHealthStateReturn{
				healthStatus: storage.ClusterHealthStatus_HEALTHY,
				found:        true,
				err:          nil,
			},
		},
		{
			desc: "Query with custom pagination",
			query: &apiV2.RawQuery{
				Query:      "",
				Pagination: &apiV2.Pagination{Limit: 1},
			},
			expectedQ:      search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(1)).ProtoQuery(),
			expectedCountQ: search.EmptyQuery(),
			sensorHealthState: sensorHealthStateReturn{
				healthStatus: storage.ClusterHealthStatus_HEALTHY,
				found:        true,
				err:          nil,
			},
		},
		{
			desc:           "Fetch cluster failed",
			query:          &apiV2.RawQuery{Query: ""},
			expectedQ:      search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
			expectedCountQ: search.EmptyQuery(),
			sensorHealthState: sensorHealthStateReturn{
				err: errors.New("DB error"),
			},
		},
		{
			desc:           "Cluster not found",
			query:          &apiV2.RawQuery{Query: ""},
			expectedQ:      search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
			expectedCountQ: search.EmptyQuery(),
			sensorHealthState: sensorHealthStateReturn{
				found: false,
				err:   nil,
			},
		},
		{
			desc:           "Sensor connection is not established",
			query:          &apiV2.RawQuery{Query: ""},
			expectedQ:      search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
			expectedCountQ: search.EmptyQuery(),
			sensorHealthState: sensorHealthStateReturn{
				healthStatus: storage.ClusterHealthStatus_DEGRADED,
				found:        true,
				err:          nil,
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			expectedResp := &apiV2.ListComplianceIntegrationsResponse{
				Integrations: []*apiV2.ComplianceIntegration{
					{
						Id:                  uuid.NewDummy().String(),
						Version:             "22",
						ClusterId:           fixtureconsts.Cluster1,
						ClusterName:         mockClusterName,
						Namespace:           fixtureconsts.Namespace1,
						StatusErrors:        []string{"Error 1", "Error 2", "Error 3"},
						OperatorInstalled:   true,
						Status:              apiV2.COStatus_HEALTHY,
						ClusterPlatformType: apiV2.ClusterPlatformType_OPENSHIFT_CLUSTER,
						ClusterProviderType: apiV2.ClusterProviderType_OCP,
					},
				},
				TotalCount: 6,
			}

			// Adjust expected response for sensor connection status
			if tc.sensorHealthState.err != nil {
				expectedResp.Integrations[0].StatusErrors = append(expectedResp.Integrations[0].StatusErrors, fmt.Sprintf(fmtGetClusterErr, mockClusterName))
			} else if !tc.sensorHealthState.found {
				expectedResp.Integrations[0].StatusErrors = append(expectedResp.Integrations[0].StatusErrors, fmt.Sprintf(fmtGetClusterNotFound, mockClusterName))
			} else if tc.sensorHealthState.healthStatus != storage.ClusterHealthStatus_HEALTHY {
				expectedResp.Integrations[0].StatusErrors = append(expectedResp.Integrations[0].StatusErrors, fmt.Sprintf(fmtGetClusterUnhealthy, mockClusterName))
			}

			s.complianceIntegrationDataStore.EXPECT().GetComplianceIntegration(gomock.Any(), gomock.Any()).Return(&storage.ComplianceIntegration{
				Id:                  uuid.NewDummy().String(),
				Version:             "22",
				ClusterId:           fixtureconsts.Cluster1,
				ComplianceNamespace: fixtureconsts.Namespace1,
				StatusErrors:        []string{"Error 1", "Error 2", "Error 3"},
			}, true, nil).Times(1)

			s.clusterDatastore.EXPECT().WalkClusters(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, fn func(c *storage.Cluster) error) error {
				// Getting clusters from DB failed.
				if tc.sensorHealthState.err != nil {
					return errors.New("DB error")
				}

				storedClusters := []*storage.Cluster{
					{Id: fixtureconsts.Cluster2, HealthStatus: &storage.ClusterHealthStatus{SensorHealthStatus: storage.ClusterHealthStatus_UNINITIALIZED}},
					{Id: fixtureconsts.Cluster3, HealthStatus: &storage.ClusterHealthStatus{SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY}},
				}
				if tc.sensorHealthState.found {
					storedClusters = append(storedClusters, &storage.Cluster{Id: fixtureconsts.Cluster1, HealthStatus: &storage.ClusterHealthStatus{SensorHealthStatus: tc.sensorHealthState.healthStatus}})
				}

				for _, cluster := range storedClusters {
					_ = fn(cluster)
				}
				return nil
			})

			s.complianceIntegrationDataStore.EXPECT().GetComplianceIntegrationsView(allAccessContext, tc.expectedQ).
				Return([]*datastore.IntegrationDetails{{
					ID:                                uuid.NewDummy().String(),
					Version:                           "22",
					OperatorInstalled:                 pointers.Bool(true),
					OperatorStatus:                    pointers.Pointer(storage.COStatus_HEALTHY),
					ClusterID:                         fixtureconsts.Cluster1,
					ClusterName:                       mockClusterName,
					Type:                              pointers.Pointer(storage.ClusterType_OPENSHIFT_CLUSTER),
					StatusProviderMetadataClusterType: pointers.Pointer(storage.ClusterMetadata_OCP),
				},
				}, nil).Times(1)

			s.complianceIntegrationDataStore.EXPECT().CountIntegrations(allAccessContext, tc.expectedCountQ).
				Return(6, nil).Times(1)

			configs, err := s.service.ListComplianceIntegrations(allAccessContext, tc.query)
			s.NoError(err)
			protoassert.Equal(s.T(), expectedResp, configs)
		})
	}
}
