package service

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestConvertStorageIntegrationToV2(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	clusterDatastore := clusterMocks.NewMockDataStore(mockCtrl)
	mockClusterName := "mock-cluster"

	var cases = []struct {
		testname     string
		integration  *storage.ComplianceIntegration
		expected     *apiV2.ComplianceIntegration
		clusterError bool
	}{
		{
			testname: "Integration conversion",
			integration: &storage.ComplianceIntegration{
				Id:           uuid.NewDummy().String(),
				Version:      "22",
				ClusterId:    fixtureconsts.Cluster1,
				Namespace:    fixtureconsts.Namespace1,
				NamespaceId:  fixtureconsts.Namespace1,
				StatusErrors: []string{"Error 1", "Error 2", "Error 3"},
			},
			expected: &apiV2.ComplianceIntegration{
				Id:           uuid.NewDummy().String(),
				Version:      "22",
				ClusterId:    fixtureconsts.Cluster1,
				ClusterName:  mockClusterName,
				Namespace:    fixtureconsts.Namespace1,
				StatusErrors: []string{"Error 1", "Error 2", "Error 3"},
			},
			clusterError: false,
		},
		{
			testname: "Integration conversion with cluster error",
			integration: &storage.ComplianceIntegration{
				Id:           uuid.NewDummy().String(),
				Version:      "22",
				ClusterId:    fixtureconsts.Cluster1,
				Namespace:    fixtureconsts.Namespace1,
				NamespaceId:  fixtureconsts.Namespace1,
				StatusErrors: []string{"Error 1", "Error 2", "Error 3"},
			},
			expected:     nil,
			clusterError: true,
		},
	}

	for _, c := range cases {
		t.Run(c.testname, func(t *testing.T) {
			if c.clusterError {
				clusterDatastore.EXPECT().GetClusterName(gomock.Any(), c.integration.GetClusterId()).Return("", false, errors.New("test can't find cluster name")).Times(1)
			} else {
				clusterDatastore.EXPECT().GetClusterName(gomock.Any(), c.integration.GetClusterId()).Return(mockClusterName, true, nil).Times(1)
			}

			converted, err := convertStorageIntegrationToV2(context.Background(), c.integration, clusterDatastore)
			if c.clusterError {
				assert.NotNil(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, c.expected, converted)
		})
	}
}
