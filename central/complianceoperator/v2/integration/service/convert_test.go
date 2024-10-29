package service

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	complianceMocks "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore/mocks"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestConvertStorageIntegrationToV2(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	complianceDatastore := complianceMocks.NewMockDataStore(mockCtrl)
	mockClusterName := "mock-cluster"
	testID := uuid.NewDummy().String()

	var cases = []struct {
		testname     string
		integration  *storage.ComplianceIntegration
		view         *datastore.IntegrationDetails
		expected     *apiV2.ComplianceIntegration
		clusterError bool
	}{
		{
			testname: "Integration conversion",
			integration: &storage.ComplianceIntegration{
				Id:                  testID,
				Version:             "22",
				ClusterId:           fixtureconsts.Cluster1,
				ComplianceNamespace: fixtureconsts.Namespace1,
				StatusErrors:        []string{"Error 1", "Error 2", "Error 3"},
				OperatorInstalled:   true,
			},
			view: &datastore.IntegrationDetails{
				ID:                                testID,
				Version:                           "22",
				OperatorInstalled:                 pointers.Bool(true),
				OperatorStatus:                    pointers.Pointer(storage.COStatus_HEALTHY),
				ClusterID:                         fixtureconsts.Cluster1,
				ClusterName:                       mockClusterName,
				Type:                              pointers.Pointer(storage.ClusterType_OPENSHIFT_CLUSTER),
				StatusProviderMetadataClusterType: pointers.Pointer(storage.ClusterMetadata_OCP),
			},
			expected: &apiV2.ComplianceIntegration{
				Id:                  testID,
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
			clusterError: false,
		},
		{
			testname: "Integration conversion with cluster error",
			integration: &storage.ComplianceIntegration{
				Id:                  testID,
				Version:             "22",
				ClusterId:           fixtureconsts.Cluster1,
				ComplianceNamespace: fixtureconsts.Namespace1,
				StatusErrors:        []string{"Error 1", "Error 2", "Error 3"},
			},
			view: &datastore.IntegrationDetails{
				ID:                                testID,
				Version:                           "22",
				OperatorInstalled:                 pointers.Bool(true),
				OperatorStatus:                    pointers.Pointer(storage.COStatus_HEALTHY),
				ClusterID:                         testconsts.Cluster1,
				ClusterName:                       mockClusterName,
				Type:                              pointers.Pointer(storage.ClusterType_OPENSHIFT_CLUSTER),
				StatusProviderMetadataClusterType: pointers.Pointer(storage.ClusterMetadata_OCP),
			},
			expected:     nil,
			clusterError: true,
		},
		{
			testname: "Integration conversion with nil pointers",
			integration: &storage.ComplianceIntegration{
				Id:                  testID,
				Version:             "22",
				ClusterId:           fixtureconsts.Cluster1,
				ComplianceNamespace: fixtureconsts.Namespace1,
				StatusErrors:        []string{"Error 1", "Error 2", "Error 3"},
				OperatorInstalled:   true,
			},
			view: &datastore.IntegrationDetails{
				ID:                                testID,
				Version:                           "22",
				OperatorInstalled:                 nil,
				OperatorStatus:                    nil,
				ClusterID:                         fixtureconsts.Cluster1,
				ClusterName:                       mockClusterName,
				Type:                              nil,
				StatusProviderMetadataClusterType: nil,
			},
			expected: &apiV2.ComplianceIntegration{
				Id:                  testID,
				Version:             "22",
				ClusterId:           fixtureconsts.Cluster1,
				ClusterName:         mockClusterName,
				Namespace:           fixtureconsts.Namespace1,
				StatusErrors:        []string{"Error 1", "Error 2", "Error 3"},
				OperatorInstalled:   false,
				Status:              apiV2.COStatus_UNHEALTHY,
				ClusterPlatformType: apiV2.ClusterPlatformType_GENERIC_CLUSTER,
				ClusterProviderType: apiV2.ClusterProviderType_UNSPECIFIED,
			},
			clusterError: false,
		},
	}

	for _, c := range cases {
		t.Run(c.testname, func(t *testing.T) {
			if c.clusterError {
				complianceDatastore.EXPECT().GetComplianceIntegration(gomock.Any(), c.view.ID).Return(nil, false, errors.New("test can't find compliance integration")).Times(1)
			} else {
				complianceDatastore.EXPECT().GetComplianceIntegration(gomock.Any(), c.view.ID).Return(c.integration, true, nil).Times(1)
			}

			converted, clusterFound, err := convertStorageIntegrationToV2(context.Background(), c.view, complianceDatastore)
			if c.clusterError {
				assert.NotNil(t, err)
				assert.False(t, clusterFound)
			} else {
				assert.NoError(t, err)
				assert.True(t, clusterFound)
			}
			protoassert.Equal(t, c.expected, converted)
		})
	}
}
