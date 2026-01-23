package service

import (
	"testing"

	clusterDSMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_telemetryServiceClient(t *testing.T) {
	t.Run("nil identity returns no-op option", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		clusters := clusterDSMocks.NewMockDataStore(ctrl)

		option := telemetryServiceClient(nil, clusters)

		opts := telemeter.ApplyOptions([]telemeter.Option{option})
		assert.Empty(t, opts.UserID)
		assert.Empty(t, opts.ClientID)
		assert.Nil(t, opts.Traits)
	})

	t.Run("UNKNOWN_SERVICE returns WithUserID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		id := mocks.NewMockIdentity(ctrl)
		id.EXPECT().Service().Return(&storage.ServiceIdentity{
			Type: storage.ServiceType_UNKNOWN_SERVICE,
		}).AnyTimes()
		id.EXPECT().UID().Return("test-user-id")
		clusters := clusterDSMocks.NewMockDataStore(ctrl)

		option := telemetryServiceClient(id, clusters)

		opts := telemeter.ApplyOptions([]telemeter.Option{option})
		assert.Equal(t, "test-user-id", opts.UserID)
		assert.Empty(t, opts.AnonymousID)
	})

	t.Run("default service type returns WithClient", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		id := mocks.NewMockIdentity(ctrl)
		id.EXPECT().Service().Return(&storage.ServiceIdentity{
			Type: storage.ServiceType_CENTRAL_SERVICE,
			Id:   "central-id-123",
		}).AnyTimes()
		clusters := clusterDSMocks.NewMockDataStore(ctrl)

		option := telemetryServiceClient(id, clusters)

		opts := telemeter.ApplyOptions([]telemeter.Option{option})
		assert.Equal(t, "central-id-123", opts.ClientID)
		assert.Equal(t, "CENTRAL_SERVICE", opts.ClientType)
		assert.Empty(t, opts.ClientVersion)
		assert.Equal(t, "central-id-123", opts.AnonymousID)
	})

	t.Run("SENSOR_SERVICE returns WithClient with cluster info", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		id := mocks.NewMockIdentity(ctrl)
		id.EXPECT().Service().Return(&storage.ServiceIdentity{
			Type: storage.ServiceType_SENSOR_SERVICE,
			Id:   "cluster-id-789",
		}).AnyTimes()
		clusters := clusterDSMocks.NewMockDataStore(ctrl)
		clusters.EXPECT().GetCluster(gomock.Any(), "cluster-id-789").Return(&storage.Cluster{
			Id:        "cluster-id-789",
			Name:      "test-cluster",
			MainImage: "quay.io/stackrox-io/main:4.0.0",
		}, true, nil)

		option := telemetryServiceClient(id, clusters)

		opts := telemeter.ApplyOptions([]telemeter.Option{option})
		assert.Equal(t, "cluster-id-789", opts.ClientID)
		assert.Equal(t, "Secured Cluster", opts.ClientType)
		assert.Equal(t, "quay.io/stackrox-io/main:4.0.0", opts.ClientVersion)
	})
}
