package expiry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	clusterDSMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"github.com/stackrox/rox/central/credentialexpiry/service"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var errTest = errors.New("test")

type mockService struct{}

func (ms *mockService) AuthFuncOverride(context.Context, string) (context.Context, error) {
	panic("unimplemented")
}

func (ms *mockService) GetCertExpiry(_ context.Context, req *v1.GetCertExpiry_Request) (*v1.GetCertExpiry_Response, error) {
	if req.GetComponent() == v1.GetCertExpiry_SCANNER_V4 {
		return nil, errTest
	}
	return nil, nil
}

func (ms *mockService) RegisterServiceHandler(context.Context, *runtime.ServeMux, *grpc.ClientConn) error {
	panic("unimplemented")
}

func (ms *mockService) RegisterServiceServer(*grpc.Server) {
	panic("unimplemented")
}

var _ service.Service = (*mockService)(nil)

func Test_track(t *testing.T) {
	var s mockService
	ctrl := gomock.NewController(t)
	clusters := clusterDSMocks.NewMockDataStore(ctrl)
	clusters.EXPECT().WalkClusters(gomock.Any(), gomock.Any()).Return(nil)

	components := make([]string, 0, len(v1.GetCertExpiry_Component_name))
	for f := range track(context.Background(), &s, clusters) {
		components = append(components, f.component)
	}
	assert.ElementsMatch(t, []string{"SCANNER", "CENTRAL_DB", "CENTRAL"}, components,
		"should have no UNKNOWN and SCANNER_V4")
}

func Test_track_securedClusters(t *testing.T) {
	var s mockService
	ctrl := gomock.NewController(t)
	clusters := clusterDSMocks.NewMockDataStore(ctrl)

	connected := &storage.Cluster{
		Name: "connected-cluster",
		Status: &storage.ClusterStatus{
			CertExpiryStatus: &storage.ClusterCertExpiryStatus{
				SensorCertExpiry: timestamppb.New(time.Now().Add(24 * time.Hour)),
			},
		},
	}
	neverConnected := &storage.Cluster{Name: "never-connected-cluster"}

	clusters.EXPECT().WalkClusters(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, fn func(*storage.Cluster) error) error {
			for _, cluster := range []*storage.Cluster{connected, neverConnected} {
				if err := fn(cluster); err != nil {
					return err
				}
			}
			return nil
		})

	var securedClusterFindings []*finding
	for f, err := range track(context.Background(), &s, clusters) {
		assert.NoError(t, err)
		if f.component == securedClusterComponent {
			securedClusterFindings = append(securedClusterFindings, f)
		}
	}

	if assert.Len(t, securedClusterFindings, 1, "the never-connected cluster should not produce a finding") {
		assert.Equal(t, "connected-cluster", securedClusterFindings[0].name)
		assert.InDelta(t, 24, securedClusterFindings[0].hoursUntilExpiration, 1)
	}
}
