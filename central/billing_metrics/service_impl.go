package billingmetrics

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	bmstore "github.com/stackrox/rox/central/billing_metrics/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc"
)

var (
	log        = logging.LoggerForModule()
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		allow.Anonymous(): {
			"/v1.MaximumValueService/GetMaximum",
		},
		user.With(permissions.Modify(resources.Administration)): {
			"/v1.MaximumValueService/PostMaximum",
			"/v1.MaximumValueService/DeleteMaximum",
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
	v1.RegisterMaximumValueServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterMaximumValueServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetMaximumValue returns the publicly available config
func (s *serviceImpl) GetMaximum(ctx context.Context, m *v1.MaximumValueRequest) (*v1.MaximumValueResponse, error) {
	maximum, ok, err := s.store.Get(ctx, m.Metric)
	if err != nil || !ok {
		return nil, err
	}
	return &v1.MaximumValueResponse{Metric: maximum.Metric, Value: maximum.Value, Ts: maximum.Ts}, nil
}

// GetMaximumValue returns the publicly available config
func (s *serviceImpl) PostMaximum(ctx context.Context, m *v1.MaximumValueUpdateRequest) (*v1.Empty, error) {
	v := &storage.Maximus{Metric: m.Metric, Value: m.Value, Ts: m.Ts}
	return nil, s.store.Upsert(ctx, v)
}

// DeleteMaximumValue returns the publicly available config
func (s *serviceImpl) DeleteMaximum(ctx context.Context, m *v1.MaximumValueRequest) (*v1.Empty, error) {
	return nil, s.store.Delete(ctx, m.Metric)
}
