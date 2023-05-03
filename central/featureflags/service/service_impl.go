package service

import (
	"context"
	"sort"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		allow.Anonymous(): {
			// This endpoint is used by the UI to populate framework
			// information which is done at UI start when there is no
			// authenticated user yet. It should stay anonymous / public
			// as long as the UI needs it at startup time.
			"/v1.FeatureFlagService/GetFeatureFlags",
		},
	})
)

type serviceImpl struct {
	v1.UnimplementedFeatureFlagServiceServer
}

func (s *serviceImpl) GetFeatureFlags(context.Context, *v1.Empty) (*v1.GetFeatureFlagsResponse, error) {
	resp := &v1.GetFeatureFlagsResponse{}
	for _, feature := range features.Flags {
		resp.FeatureFlags = append(resp.FeatureFlags, &v1.FeatureFlag{
			Name:    feature.Name(),
			EnvVar:  feature.EnvVar(),
			Enabled: feature.Enabled(),
		})
	}

	sort.Slice(resp.FeatureFlags, func(i, j int) bool {
		return resp.FeatureFlags[i].GetName() < resp.FeatureFlags[j].GetName()
	})
	return resp, nil
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterFeatureFlagServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterFeatureFlagServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}
