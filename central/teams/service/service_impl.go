package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/convert/storagetov1"
	"github.com/stackrox/rox/central/convert/v1tostorage"
	"github.com/stackrox/rox/central/teams/datastore"
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
	_ v1.TeamServiceServer = (*serviceImpl)(nil)

	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Access)): {
			"/v1.TeamService/ListTeams",
			"/v1.TeamService/GetTeam",
		},
		user.With(permissions.Modify(resources.Access)): {
			"/v1.TeamService/AddTeam",
		},
	})
)

type serviceImpl struct {
	v1.UnimplementedTeamServiceServer

	ds datastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterTeamServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterTeamServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) ListTeams(ctx context.Context, _ *v1.Empty) (*v1.ListTeamsResponse, error) {
	teams, err := s.ds.ListTeams(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.ListTeamsResponse{Teams: storagetov1.Teams(teams)}, nil
}

func (s *serviceImpl) GetTeam(ctx context.Context, req *v1.ResourceByID) (*v1.GetTeamResponse, error) {
	team, exists, err := s.ds.GetTeam(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errox.NotFound.Newf("team with id %q not found", req.GetId())
	}
	return &v1.GetTeamResponse{Team: storagetov1.Team(team)}, nil
}

func (s *serviceImpl) AddTeam(ctx context.Context, request *v1.AddTeamRequest) (*v1.AddTeamResponse, error) {
	team, err := s.ds.AddTeam(ctx, v1tostorage.Team(request.GetTeam()))
	if err != nil {
		return nil, err
	}
	return &v1.AddTeamResponse{Team: storagetov1.Team(team)}, nil
}
