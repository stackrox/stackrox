package service

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	integrationMocks "github.com/stackrox/rox/central/imageintegration/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestValidateIntegration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	clusterDatastore := clusterMocks.NewMockDataStore(ctrl)
	clusterDatastore.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{}, nil).AnyTimes()

	integrationDatastore := integrationMocks.NewMockDataStore(ctrl)

	s := &serviceImpl{clusterDatastore: clusterDatastore, datastore: integrationDatastore}

	// Test name and categories validation
	assert.Error(t, s.validateIntegration(context.TODO(), &storage.ImageIntegration{}))

	assert.Error(t, s.validateIntegration(context.TODO(), &storage.ImageIntegration{
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
	}))

	// Test should be successful
	integrationDatastore.EXPECT().GetImageIntegrations(gomock.Any(), &v1.GetImageIntegrationsRequest{Name: "name"}).Return([]*storage.ImageIntegration{}, nil)
	assert.NoError(t, s.validateIntegration(context.TODO(), &storage.ImageIntegration{
		Name:       "name",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
	}))

	// Test name scenarios

	integrationDatastore.EXPECT().GetImageIntegrations(gomock.Any(), &v1.GetImageIntegrationsRequest{Name: "name"}).Return([]*storage.ImageIntegration{{Id: "id", Name: "name"}}, nil).AnyTimes()
	// Duplicate name with different ID should fail
	assert.Error(t, s.validateIntegration(context.TODO(), &storage.ImageIntegration{
		Id:         "diff",
		Name:       "name",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
	}))

	// Duplicate name with same ID should succeed
	assert.NoError(t, s.validateIntegration(context.TODO(), &storage.ImageIntegration{
		Id:         "id",
		Name:       "name",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
	}))
}
