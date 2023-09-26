package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/policysync/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"google.golang.org/grpc"
)

var (
	_ v1.PolicySyncServiceServer = (*serviceImpl)(nil)

	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.WorkflowAdministration)): {
			"/v1.PolicySyncService/GetPolicyRequest",
		},
		user.With(permissions.Modify(resources.WorkflowAdministration)): {
			"/v1.PolicySyncService/PostPolicyRequest",
		},
	})
)

type serviceImpl struct {
	v1.UnimplementedPolicySyncServiceServer

	ds datastore.DataStore
}

func (s *serviceImpl) RegisterServiceServer(server *grpc.Server) {
	v1.RegisterPolicySyncServiceServer(server, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterPolicySyncServiceHandler(ctx, mux, conn)
}

func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GetPolicyRequest(ctx context.Context, _ *v1.Empty) (*v1.GetPolicySyncResponse, error) {
	sync, exists, err := s.ds.GetPolicySync(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve policy sync")
	}

	if !exists {
		return &v1.GetPolicySyncResponse{}, nil
	}

	return &v1.GetPolicySyncResponse{
		Sync: toV1Proto(sync),
	}, nil
}

func (s *serviceImpl) PostPolicyRequest(ctx context.Context, request *v1.PostPolicySyncRequest) (*v1.Empty, error) {
	if request == nil {
		return nil, errox.InvalidArgs.CausedBy("policy sync missing")
	}

	if err := s.ds.UpsertPolicySync(ctx, toStorageProto(request.GetPolicySync())); err != nil {
		return nil, errors.Wrap(err, "upserting policy sync")
	}

	return nil, nil
}
