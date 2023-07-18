package billingmetrics

import (
	"context"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	bmstore "github.com/stackrox/rox/central/billingmetrics/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
	"google.golang.org/grpc"
)

var (
	log        = logging.LoggerForModule()
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		allow.Anonymous(): {
			"/v1.BillingMetricsService/GetMetrics",
		},
		user.With(permissions.Modify(resources.Administration)): {
			"/v1.BillingMetricsService/PutMetrics",
		},
	})
)

type serviceImpl struct {
	store bmstore.Store
}

// New returns a new Service instance using the given DataStore.
func New(datastore bmstore.Store) Service {
	return &serviceImpl{
		store: datastore,
	}
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterBillingMetricsServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterBillingMetricsServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GetMetrics(ctx context.Context, req *v1.BillingMetricsRequest) (*v1.BillingMetricsResponse, error) {
	metrics, err := s.store.Get(ctx, protoconv.ConvertTimestampToTimeOrNow(req.GetFrom()), protoconv.ConvertTimestampToTimeOrNow(req.GetTo()))
	if err != nil {
		return nil, fmt.Errorf("cannot get billing metrics: %w", err)
	}
	rec := make([]*v1.BillingMetricsResponse_BillingMetricsRecord, 0, len(metrics))
	for _, m := range metrics {
		rec = append(rec, &v1.BillingMetricsResponse_BillingMetricsRecord{Ts: m.Ts, Metrics: (*v1.SecuredResourcesMetrics)(m.Sr)})
	}
	return &v1.BillingMetricsResponse{Record: rec}, nil
}

func (s *serviceImpl) PutMetrics(ctx context.Context, m *v1.BillingMetricsInsertRequest) (*v1.Empty, error) {
	v := &storage.BillingMetricsRecord{Ts: m.GetTs(), Sr: &storage.BillingMetricsRecord_SecuredResources{
		Nodes:      m.GetMetrics().GetNodes(),
		Millicores: m.GetMetrics().GetMillicores(),
	}}

	if err := s.store.Insert(ctx, v); err != nil {
		return nil, fmt.Errorf("cannot insert metrics: %w", err)
	}
	return &v1.Empty{}, nil
}
